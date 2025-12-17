package otp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type OTPService interface {
	GenerateAndSendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email, code string) (bool, error)
}

type otpService struct {
	repo OTPRepository
}

func NewOTPService(repo OTPRepository) OTPService {
	return &otpService{repo: repo}
}

func (s *otpService) GenerateAndSendOTP(ctx context.Context, email string) error {
	// Generate 6-digit OTP
	code := generateOTP(6)

	// OTP expires in 10 minutes
	expiresAt := time.Now().Add(10 * time.Minute)

	// Save OTP to database
	_, err := s.repo.CreateOTP(ctx, email, code, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	// Send OTP via SendGrid
	if err := s.sendOTPEmail(email, code); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	// Clean up expired OTPs
	_ = s.repo.DeleteExpiredOTPs(ctx)

	return nil
}

func (s *otpService) VerifyOTP(ctx context.Context, email, code string) (bool, error) {
	// Get latest unverified OTP for this email
	otp, err := s.repo.GetOTPByEmail(ctx, email)
	if err != nil {
		return false, errors.New("no OTP found for this email or OTP already verified")
	}

	// Check if OTP has expired
	if time.Now().After(otp.ExpiresAt) {
		return false, errors.New("OTP has expired")
	}

	// Verify the code
	if otp.Code != code {
		return false, errors.New("invalid OTP code")
	}

	// Mark OTP as verified
	if err := s.repo.MarkOTPAsVerified(ctx, otp.ID); err != nil {
		return false, fmt.Errorf("failed to mark OTP as verified: %w", err)
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
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		return errors.New("SENDGRID_API_KEY not set in environment")
	}

	// Get sender email from environment variable
	senderEmail := os.Getenv("SENDGRID_SENDER_EMAIL")
	if senderEmail == "" {
		return errors.New("SENDGRID_SENDER_EMAIL not set in environment")
	}

	senderName := os.Getenv("SENDGRID_SENDER_NAME")
	if senderName == "" {
		senderName = "Graveyard"
	}

	from := mail.NewEmail(senderName, senderEmail)
	to := mail.NewEmail("", toEmail)
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

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(apiKey)

	response, err := client.Send(message)
	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned status code %d: %s", response.StatusCode, response.Body)
	}

	return nil
}
