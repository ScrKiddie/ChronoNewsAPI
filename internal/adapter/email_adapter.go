package adapter

import (
	"chronoverseapi/internal/model"
	"fmt"
	"net/smtp"
)

type EmailAdapter struct {
}

func NewEmailAdapter() *EmailAdapter {
	return &EmailAdapter{}
}

func (e *EmailAdapter) Send(request *model.EmailData) error {
	auth := smtp.PlainAuth(
		"",
		request.Username,
		request.Password,
		request.SMTPHost,
	)

	headers := fmt.Sprintf("From: %s <%s>\r\n", request.FromName, request.FromEmail)
	headers += fmt.Sprintf("To: %s\r\n", request.To)
	headers += fmt.Sprintf("Subject: %s\r\n", request.Subject)
	headers += "Content-Type: text/html; charset=UTF-8\r\n\r\n"

	message := headers + request.Body

	address := fmt.Sprintf("%s:%d", request.SMTPHost, request.SMTPPort)
	err := smtp.SendMail(address, auth, request.FromEmail, []string{request.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
