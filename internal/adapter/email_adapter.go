package adapter

import (
	"chrononewsapi/internal/model"
	"fmt"
	"math/rand"
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

	boundary := fmt.Sprintf("boundary-%d-%d", rand.Int(), time.Now().UnixNano())
	headers := []string{
		fmt.Sprintf("From: %s <%s>", request.FromName, request.FromEmail),
		fmt.Sprintf("To: <%s>", request.To),
		fmt.Sprintf("Subject: %s", request.Subject),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		fmt.Sprintf("Message-ID: <%d@%s>", time.Now().UnixNano(), request.SMTPHost),
		"MIME-Version: 1.0",
		fmt.Sprintf(`Content-Type: multipart/alternative; boundary="%s"`, boundary),
	}

	var builder strings.Builder
	builder.WriteString(strings.Join(headers, "\r\n"))
	builder.WriteString("\r\n\r\n")
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	builder.WriteString(request.Body)
	builder.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	fullMessage := builder.String()
	address := fmt.Sprintf("%s:%d", request.SMTPHost, request.SMTPPort)

	err := smtp.SendMail(address, auth, request.FromEmail, []string{request.To}, []byte(fullMessage))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
