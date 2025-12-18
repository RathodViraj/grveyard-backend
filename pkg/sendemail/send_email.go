package sendemail

import (
	"errors"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type EmailService interface {
	SendEmail(subject, toEmail, plainTextContent, htmlContent string) error
}

type emailService struct {
	clinet      *sendgrid.Client
	senderEmail string
	senderName  string
}

func NewEmailService() EmailService {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	senderEmail := os.Getenv("SENDGRID_SENDER_EMAIL")
	senderName := os.Getenv("SENDGRID_SENDER_NAME")
	return &emailService{
		clinet:      sendgrid.NewSendClient(apiKey),
		senderEmail: senderEmail,
		senderName:  senderName,
	}
}

func (e *emailService) SendEmail(subject, toEmail, plainTextContent, htmlContent string) error {
	from := mail.NewEmail(e.senderName, e.senderEmail)
	to := mail.NewEmail("", toEmail)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	response, err := e.clinet.Send(message)
	if err != nil {
		return err
	}
	if response.StatusCode >= 400 {
		return errors.New("failed to send email")
	}
	return nil
}
