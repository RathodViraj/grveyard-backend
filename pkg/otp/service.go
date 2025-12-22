package otp

import (
	"context"
	"errors"
	"fmt"
	sendemail "grveyard/pkg/sendemail"
	"grveyard/pkg/users"
	"math/rand"
	"time"
)

type OTPService interface {
	GenerateAndSendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email, code string) (bool, error)
}

type otpService struct {
	repo     OTPRepository
	userRepo users.UserRepository
	es       sendemail.EmailService
}

func NewOTPService(repo OTPRepository, userRepo users.UserRepository, es sendemail.EmailService) OTPService {
	return &otpService{repo: repo, userRepo: userRepo, es: es}
}

func (s *otpService) GenerateAndSendOTP(ctx context.Context, email string) error {
	count, err := s.repo.CountOTPsInLastHour(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to check OTP count: %w", err)
	}

	if count >= 3 {
		return errors.New("too many OTP requests. Please try again later")
	}

	code := generateOTP(6)

	expiresAt := time.Now().Add(10 * time.Minute)

	_, err = s.repo.CreateOTP(ctx, email, code, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	if err := s.sendOTPEmail(email, code); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	_ = s.repo.DeleteExpiredOTPs(ctx)

	return nil
}

func (s *otpService) VerifyOTP(ctx context.Context, email, code string) (bool, error) {
	otp, err := s.repo.GetOTPByEmail(ctx, email)
	if err != nil {
		return false, errors.New("no OTP found for this email or OTP already verified")
	}

	if time.Now().After(otp.ExpiresAt) {
		return false, errors.New("OTP has expired")
	}

	if otp.Code != code {
		return false, errors.New("invalid OTP code")
	}

	if err := s.repo.MarkOTPAsVerified(ctx, otp.ID); err != nil {
		return false, fmt.Errorf("failed to mark OTP as verified: %w", err)
	}

	now := time.Now()
	if err := s.userRepo.UpdateVerifiedAtByEmail(ctx, email, now); err != nil {
		return false, fmt.Errorf("failed to update user verification: %w", err)
	}

	return true, nil
}

func generateOTP(length int) string {
	digits := "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		otp[i] = digits[rand.Intn(len(digits))]
	}
	return string(otp)
}

func (s *otpService) sendOTPEmail(toEmail, code string) error {
	subject := "Your OTP Code"
	plainTextContent := fmt.Sprintf("Your OTP code is: %s. This code will expire in 10 minutes.", code)
	htmlContent := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Your OTP Code</h2>
			<p>Your one-time password is:</p>
			<div style="font-size: 24px; font-weight: bold; color: #333; padding: 10px; background-color: #f5f5f5; border-radius: 5px; display: inline-block;">
				%s
			</div>
			<p>This code will expire in 10 minutes.</p>
			<p>If you didn't request this code, please ignore this email.</p>
		</div>
	`, code)

	err := s.es.SendEmail(subject, toEmail, plainTextContent, htmlContent)
	return err
}
