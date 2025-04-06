package adapter

import (
	"chronoverseapi/internal/model"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type EmailAdapter struct{}

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

	// Buat header email
	var headers []string
	headers = append(headers, fmt.Sprintf("From: %s <%s>", request.FromName, request.FromEmail))
	headers = append(headers, fmt.Sprintf("To: <%s>", request.To))
	headers = append(headers, fmt.Sprintf("Subject: %s", request.Subject))
	headers = append(headers, fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)))
	headers = append(headers, fmt.Sprintf("Message-ID: <%d@%s>", time.Now().UnixNano(), request.SMTPHost))
	headers = append(headers, "MIME-Version: 1.0")
	headers = append(headers, `Content-Type: multipart/alternative; boundary="boundary123"`)

	// Body email dengan versi plain text dan HTML
	body := strings.Join([]string{
		"--boundary123",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Versi teks email ini.",
		"",
		"--boundary123",
		"Content-Type: text/html; charset=UTF-8",
		"",
		request.Body,
		"",
		"--boundary123--",
	}, "\r\n")

	fullMessage := strings.Join(headers, "\r\n") + "\r\n\r\n" + body

	address := fmt.Sprintf("%s:%d", request.SMTPHost, request.SMTPPort)
	err := smtp.SendMail(address, auth, request.FromEmail, []string{request.To}, []byte(fullMessage))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
