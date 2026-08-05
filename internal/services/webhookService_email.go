package services

import (
	"context"
	"fmt"
	"strings"

	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/mailer"
)

const webhookTemplateEmail = "email"

func webhookEmailBody(name string, eventType string, rawPayloadBytes []byte) string {
	var parsed any
	body := strings.Builder{}
	fmt.Fprintf(&body, "Webhook: %s\nEvent: %s\n\n", name, eventType)
	if err := jsonx.Unmarshal(rawPayloadBytes, &parsed); err == nil {
		if pretty, err := jsonx.MarshalIndent(parsed, "", "  "); err == nil {
			body.Write(pretty)
			return body.String()
		}
	}
	body.Write(rawPayloadBytes)
	return body.String()
}

func (s *webhookService) sendEmail(ctx context.Context, wh *models.WebhookEntity, eventType string, rawPayloadBytes []byte) error {
	if s.settings == nil {
		return apperrors.New(apperrors.ErrInternalError, "Email delivery is not available")
	}
	config, err := s.settings.SMTP(ctx)
	if err != nil {
		return err
	}
	recipients, err := mailer.ParseRecipients(wh.URL)
	if err != nil {
		return apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	subject := fmt.Sprintf("[NovelHub] %s: %s", wh.Name, eventType)
	body := webhookEmailBody(wh.Name, eventType, rawPayloadBytes)
	client := mailer.NewSMTPMailer(config)
	for _, recipient := range recipients {
		if err := client.SendEmail(recipient, subject, body, nil); err != nil {
			return fmt.Errorf("failed to email webhook %q: %w", wh.Name, err)
		}
	}
	return nil
}
