package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
	"novelhub/pkg/worker"
)

type WebhookService interface {
	Create(ctx context.Context, req *request.CreateWebhookDto) (*models.WebhookEntity, error)
	GetByID(ctx context.Context, id string) (*models.WebhookEntity, error)
	ListAll(ctx context.Context) ([]*models.WebhookEntity, error)
	Update(ctx context.Context, id string, req *request.UpdateWebhookDto) (*models.WebhookEntity, error)
	Delete(ctx context.Context, id string) error
	TestPing(ctx context.Context, id string) error
	DispatchEvent(ctx context.Context, eventType string, payload any)
	ExecuteDispatch(ctx context.Context, webhookID, eventType string, payloadBytes []byte) error
}

type webhookService struct {
	repo       repositories.WebhookRepository
	jobQueue   *worker.Queue
	httpClient *http.Client
}

func NewWebhookService(repo repositories.WebhookRepository, jobQueue *worker.Queue) WebhookService {
	return &webhookService{
		repo:       repo,
		jobQueue:   jobQueue,
		httpClient: netx.NewSafeHTTPClient(10 * time.Second),
	}
}

func (s *webhookService) Create(ctx context.Context, req *request.CreateWebhookDto) (*models.WebhookEntity, error) {
	id := uuid.Must(uuid.NewV7()).String()
	templateType := strings.ToLower(req.TemplateType)
	if templateType == "" {
		templateType = "generic"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	entity := &models.WebhookEntity{
		ID:            id,
		Name:          req.Name,
		URL:           req.URL,
		TemplateType:  templateType,
		Secret:        req.Secret,
		CustomHeaders: req.CustomHeaders,
		Events:        req.Events,
		IsActive:      isActive,
	}

	return s.repo.Create(ctx, entity)
}

func (s *webhookService) GetByID(ctx context.Context, id string) (*models.WebhookEntity, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "webhook not found")
	}
	return entity, nil
}

func (s *webhookService) ListAll(ctx context.Context) ([]*models.WebhookEntity, error) {
	return s.repo.ListAll(ctx)
}

func (s *webhookService) Update(ctx context.Context, id string, req *request.UpdateWebhookDto) (*models.WebhookEntity, error) {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "webhook not found")
	}

	templateType := strings.ToLower(req.TemplateType)
	if templateType == "" {
		templateType = "generic"
	}

	entity := &models.WebhookEntity{
		ID:            id,
		Name:          req.Name,
		URL:           req.URL,
		TemplateType:  templateType,
		Secret:        req.Secret,
		CustomHeaders: req.CustomHeaders,
		Events:        req.Events,
		IsActive:      req.IsActive,
	}

	return s.repo.Update(ctx, entity)
}

func (s *webhookService) Delete(ctx context.Context, id string) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "webhook not found")
	}
	return s.repo.Delete(ctx, id)
}

func (s *webhookService) TestPing(ctx context.Context, id string) error {
	wh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "webhook not found")
	}

	pingPayload := map[string]any{
		"message":   "Ping test from NovelHub",
		"timestamp": time.Now().Unix(),
	}

	payloadBytes, err := jsonx.Marshal(pingPayload)
	if err != nil {
		return err
	}

	return s.sendHTTPRequest(ctx, wh, "ping.test", payloadBytes)
}

func (s *webhookService) DispatchEvent(ctx context.Context, eventType string, payload any) {
	if s.jobQueue == nil {
		return
	}

	activeWebhooks, err := s.repo.ListActive(ctx)
	if err != nil || len(activeWebhooks) == 0 {
		return
	}

	payloadBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return
	}

	for _, wh := range activeWebhooks {
		if !s.supportsEvent(wh.Events, eventType) {
			continue
		}

		jobPayload, _ := jsonx.Marshal(map[string]any{
			"webhook_id": wh.ID,
			"event_type": eventType,
			"data":       string(payloadBytes),
		})

		s.jobQueue.Enqueue(worker.Job{
			ID:      uuid.Must(uuid.NewV7()).String(),
			Type:    "webhook.dispatch",
			Payload: string(jobPayload),
		})
	}
}

func (s *webhookService) ExecuteDispatch(ctx context.Context, webhookID, eventType string, payloadBytes []byte) error {
	wh, err := s.repo.GetByID(ctx, webhookID)
	if err != nil || !wh.IsActive {
		return nil
	}
	return s.sendHTTPRequest(ctx, wh, eventType, payloadBytes)
}

func (s *webhookService) supportsEvent(events []string, event string) bool {
	for _, e := range events {
		if e == "*" || e == event {
			return true
		}
	}
	return false
}

func (s *webhookService) sendHTTPRequest(ctx context.Context, wh *models.WebhookEntity, eventType string, rawPayloadBytes []byte) error {
	formattedBody, contentType := s.formatBodyByTemplate(wh.TemplateType, eventType, rawPayloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(formattedBody))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "NovelHub-Webhook/1.0")
	req.Header.Set("X-NovelHub-Event", eventType)

	// HMAC SHA-256 Signature
	if wh.Secret != nil && *wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(*wh.Secret))
		mac.Write(formattedBody)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-NovelHub-Signature", "sha256="+signature)
	}

	// Custom Headers (JSON string mapping)
	if wh.CustomHeaders != nil && *wh.CustomHeaders != "" {
		var headers map[string]string
		if err := jsonx.UnmarshalString(*wh.CustomHeaders, &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook endpoint returned HTTP status %d: %s", resp.StatusCode, string(bodySnippet))
	}

	return nil
}

func (s *webhookService) formatBodyByTemplate(templateType, eventType string, rawPayloadBytes []byte) ([]byte, string) {
	var rawData map[string]any
	_ = jsonx.Unmarshal(rawPayloadBytes, &rawData)

	switch templateType {
	case "discord":
		desc := fmt.Sprintf("Event **%s** triggered in NovelHub.", eventType)
		if title, ok := rawData["title"].(string); ok && title != "" {
			desc = fmt.Sprintf("**Title:** %s\nEvent: %s", title, eventType)
		}

		discordMsg := map[string]any{
			"embeds": []map[string]any{
				{
					"title":       fmt.Sprintf("📚 NovelHub Notification: %s", eventType),
					"description": desc,
					"color":       5814782, // Discord Blurple
					"timestamp":   time.Now().Format(time.RFC3339),
				},
			},
		}
		b, _ := jsonx.Marshal(discordMsg)
		return b, "application/json"

	case "telegram":
		msgText := fmt.Sprintf("<b>📚 NovelHub Event: %s</b>\n<pre>%s</pre>", eventType, string(rawPayloadBytes))
		telegramMsg := map[string]any{
			"text":       msgText,
			"parse_mode": "HTML",
		}
		b, _ := jsonx.Marshal(telegramMsg)
		return b, "application/json"

	case "slack":
		slackMsg := map[string]any{
			"blocks": []map[string]any{
				{
					"type": "header",
					"text": map[string]string{
						"type": "plain_text",
						"text": fmt.Sprintf("📚 NovelHub: %s", eventType),
					},
				},
				{
					"type": "section",
					"text": map[string]string{
						"type": "mrkdwn",
						"text": fmt.Sprintf("Event `%s` triggered at %s", eventType, time.Now().Format(time.Kitchen)),
					},
				},
			},
		}
		b, _ := jsonx.Marshal(slackMsg)
		return b, "application/json"

	default: // generic
		genericEnvelope := map[string]any{
			"event":     eventType,
			"timestamp": time.Now().Unix(),
			"data":      rawData,
		}
		b, _ := jsonx.Marshal(genericEnvelope)
		return b, "application/json"
	}
}
