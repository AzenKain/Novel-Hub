package mailer

import (
	"testing"
)

func TestSMTPMailerValidation(t *testing.T) {
	mailer := NewSMTPMailer(SMTPConfig{
		Host: "",
		Port: 0,
	})
	err := mailer.SendEmail("test@example.com", "Subject", "Body", nil)
	if err == nil {
		t.Fatalf("expected error on incomplete SMTP config, got nil")
	}
}
