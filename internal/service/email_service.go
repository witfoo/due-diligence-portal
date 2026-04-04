package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/witfoo/due-diligence-portal/pkg/envconfig"
)

// EmailService sends notification emails via SMTP.
type EmailService struct {
	enabled  bool
	host     string
	port     string
	username string
	password string
	from     string
	useTLS   bool
}

// NewEmailService creates an EmailService from environment configuration.
func NewEmailService() *EmailService {
	return &EmailService{
		enabled:  envconfig.GetEnvBool("DD_SMTP_ENABLED", false),
		host:     envconfig.GetEnv("DD_SMTP_HOST", ""),
		port:     envconfig.GetEnv("DD_SMTP_PORT", "587"),
		username: envconfig.GetEnv("DD_SMTP_USERNAME", ""),
		password: envconfig.GetEnv("DD_SMTP_PASSWORD", ""),
		from:     envconfig.GetEnv("DD_SMTP_FROM", "noreply@example.com"),
		useTLS:   envconfig.GetEnvBool("DD_SMTP_TLS", true),
	}
}

// IsEnabled returns whether email sending is configured and enabled.
func (s *EmailService) IsEnabled() bool {
	return s.enabled && s.host != ""
}

// SendInvite sends an invite email to a new user.
func (s *EmailService) SendInvite(toEmail, inviteToken, portalURL string) error {
	subject := "You've been invited to the Due Diligence Portal"
	body := fmt.Sprintf(`You have been invited to access the Due Diligence Portal.

Click the link below to create your account:

%s/register?token=%s

This invitation will expire in 7 days.`, portalURL, inviteToken)

	return s.send(toEmail, subject, body)
}

// SendQANotification sends a notification about Q&A activity.
func (s *EmailService) SendQANotification(toEmail, subject, threadSubject, authorName, body string) error {
	emailBody := fmt.Sprintf(`%s posted a response in the Q&A thread "%s":

---
%s
---

Log in to the portal to reply.`, authorName, threadSubject, body)

	return s.send(toEmail, "Q&A: "+subject, emailBody)
}

// SendDocumentNotification sends a notification about a new document upload.
func (s *EmailService) SendDocumentNotification(toEmail, documentName, categoryName, uploaderName string) error {
	subject := fmt.Sprintf("New document uploaded: %s", documentName)
	body := fmt.Sprintf(`%s uploaded a new document to the %s category:

Document: %s

Log in to the portal to view it.`, uploaderName, categoryName, documentName)

	return s.send(toEmail, subject, body)
}

// SendNDASignedNotification notifies admins when an NDA is signed.
func (s *EmailService) SendNDASignedNotification(toEmail, signerName, signerEmail, templateName string) error {
	subject := fmt.Sprintf("NDA signed by %s", signerName)
	body := fmt.Sprintf(`%s (%s) has signed the NDA "%s".

Log in to the portal to view the signature details.`, signerName, signerEmail, templateName)

	return s.send(toEmail, subject, body)
}

func (s *EmailService) send(to, subject, body string) error {
	if !s.IsEnabled() {
		log.Printf("[INFO] Email disabled, would send to %s: %s",
			sanitizeLogValue(to), sanitizeLogValue(subject))
		return nil
	}

	// Sanitize all inputs to prevent header injection (CRLF) and log injection.
	sanitizedTo := sanitizeEmailHeader(to)
	sanitizedSubject := sanitizeEmailHeader(subject)
	sanitizedBody := sanitizeEmailBody(body)

	// Build message with sanitized components only.
	msg := buildEmailMessage(s.from, sanitizedTo, sanitizedSubject, sanitizedBody)

	addr := s.host + ":" + s.port

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	if s.useTLS {
		return s.sendWithTLS(addr, auth, s.from, sanitizedTo, msg)
	}
	return smtp.SendMail(addr, auth, s.from, []string{sanitizedTo}, msg)
}

// buildEmailMessage constructs an RFC 2822 message from sanitized parts.
func buildEmailMessage(from, to, subject, body string) []byte {
	headers := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n"
	return []byte(headers + body)
}

// sanitizeEmailHeader removes CR/LF characters that could enable header injection.
func sanitizeEmailHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}

// sanitizeLogValue replaces control characters for safe log output (CWE-117).
func sanitizeLogValue(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeEmailBody removes sequences that could be interpreted as header boundaries.
func sanitizeEmailBody(body string) string {
	// Remove any CRLF+header patterns that could break out of the body section.
	body = strings.ReplaceAll(body, "\r\n.\r\n", "\r\n..\r\n")
	return body
}

func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close: %w", err)
	}

	return client.Quit()
}
