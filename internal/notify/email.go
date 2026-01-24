package notify

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

// EmailService handles sending emails via SMTP
type EmailService struct {
	Config SMTPConfig
}

// NewEmailService creates a new email service
func NewEmailService(cfg SMTPConfig) *EmailService {
	return &EmailService{Config: cfg}
}

// Send sends an email to the specified recipients
func (e *EmailService) Send(to []string, subject, body string) error {
	if !e.Config.Enabled {
		return nil
	}

	if len(to) == 0 {
		return nil
	}

	// Build message
	headers := make(map[string]string)
	headers["From"] = e.Config.From
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect and send
	addr := fmt.Sprintf("%s:%d", e.Config.Host, e.Config.Port)

	auth := smtp.PlainAuth("", e.Config.User, e.Config.Password, e.Config.Host)

	// Try with STARTTLS first
	tlsConfig := &tls.Config{
		ServerName: e.Config.Host,
	}

	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Try STARTTLS
	if ok, _ := conn.Extension("STARTTLS"); ok {
		if tlsErr := conn.StartTLS(tlsConfig); tlsErr != nil {
			return fmt.Errorf("failed to start TLS: %w", tlsErr)
		}
	}

	// Authenticate
	if authErr := conn.Auth(auth); authErr != nil {
		return fmt.Errorf("failed to authenticate: %w", authErr)
	}

	// Set sender
	if mailErr := conn.Mail(e.Config.From); mailErr != nil {
		return fmt.Errorf("failed to set sender: %w", mailErr)
	}

	// Set recipients
	for _, addr := range to {
		if rcptErr := conn.Rcpt(strings.TrimSpace(addr)); rcptErr != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, rcptErr)
		}
	}

	// Send data
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return conn.Quit()
}

// ParseRecipients splits a comma-separated list of emails
func ParseRecipients(recipients string) []string {
	if recipients == "" {
		return nil
	}
	parts := strings.Split(recipients, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
