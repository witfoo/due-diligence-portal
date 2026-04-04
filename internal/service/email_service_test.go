package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmailService_Disabled(t *testing.T) {
	svc := &EmailService{enabled: false}
	assert.False(t, svc.IsEnabled())

	// Should not error when disabled — just logs.
	err := svc.SendInvite("test@test.com", "token123", "https://portal.example.com")
	assert.NoError(t, err)
}

func TestEmailService_EnabledButNoHost(t *testing.T) {
	svc := &EmailService{enabled: true, host: ""}
	assert.False(t, svc.IsEnabled())
}

func TestEmailService_EnabledWithHost(t *testing.T) {
	svc := &EmailService{enabled: true, host: "smtp.example.com"}
	assert.True(t, svc.IsEnabled())
}

func TestEmailService_SendInvite_Disabled(t *testing.T) {
	svc := &EmailService{enabled: false}
	err := svc.SendInvite("investor@example.com", "abc123", "https://portal.example.com")
	assert.NoError(t, err)
}

func TestEmailService_SendQANotification_Disabled(t *testing.T) {
	svc := &EmailService{enabled: false}
	err := svc.SendQANotification("admin@example.com", "Re: Financials", "Q3 Revenue", "Investor", "Can you explain?")
	assert.NoError(t, err)
}

func TestEmailService_SendDocumentNotification_Disabled(t *testing.T) {
	svc := &EmailService{enabled: false}
	err := svc.SendDocumentNotification("investor@example.com", "Q3 Financials.pdf", "Financials", "Admin")
	assert.NoError(t, err)
}

func TestEmailService_SendNDASignedNotification_Disabled(t *testing.T) {
	svc := &EmailService{enabled: false}
	err := svc.SendNDASignedNotification("admin@example.com", "John Smith", "john@vc.com", "Standard NDA")
	assert.NoError(t, err)
}
