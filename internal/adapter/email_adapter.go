package adapter

import (
	"chrononewsapi/internal/model"
	"crypto/tls"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type EmailAdapter struct{}

func NewEmailAdapter() *EmailAdapter {
	return &EmailAdapter{}
}

func (e *EmailAdapter) Send(request *model.EmailData) error {
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

	const maxRetries = 3
	const initialBackoff = 2 * time.Second
	const dialTimeout = 10 * time.Second

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		slog.Info("Attempting to send email", "attempt", attempt+1, "to", request.To)

		err = sendMailWithTimeout(address, request, fullMessage, dialTimeout)
		if err == nil {
			slog.Info("Email sent successfully", "to", request.To)
			return nil
		}

		slog.Warn("Failed to send email, retrying...", "attempt", attempt+1, "error", err)

		if attempt == maxRetries-1 {
			break
		}

		backoff := initialBackoff * time.Duration(math.Pow(2, float64(attempt)))
		time.Sleep(backoff)
	}

	return fmt.Errorf("failed to send email after %d attempts: %w", maxRetries, err)
}

func sendMailWithTimeout(addr string, request *model.EmailData, msg string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	// Set deadlines for the connection
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	if err := conn.SetReadDeadline(deadline); err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	// Check if TLS is required
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}

	auth := smtp.PlainAuth("", request.Username, request.Password, host)
	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(request.FromEmail); err != nil {
		return err
	}

	if err := client.Rcpt(request.To); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}
