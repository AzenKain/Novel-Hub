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
	"strconv"
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

func BuildBookWebhookPayload(book *models.BookEntity) map[string]any {
	if book == nil {
		return map[string]any{}
	}

	payload := map[string]any{
		"id":         book.ID,
		"title":      book.Title,
		"library_id": book.LibraryID,
		"status":     book.Status,
		"created_at": book.CreatedAt,
		"updated_at": book.UpdatedAt,
	}

	if book.AuthorName != nil && *book.AuthorName != "" {
		payload["author"] = *book.AuthorName
	}
	if book.Description != nil && *book.Description != "" {
		payload["description"] = *book.Description
	}
	if book.CoverURL != nil && *book.CoverURL != "" {
		payload["cover_url"] = *book.CoverURL
	}

	if book.MetadataJSON != nil && *book.MetadataJSON != "" {
		var meta struct {
			Creator     string      `json:"creator"`
			Creators    []string    `json:"creators"`
			Description string      `json:"description"`
			Publisher   string      `json:"publisher"`
			Publishers  []string    `json:"publishers"`
			Language    string      `json:"language"`
			Date        string      `json:"date"`
			Series      string      `json:"series"`
			SeriesIndex string      `json:"seriesIndex"`
			Subject     interface{} `json:"subject"`
		}
		if err := jsonx.UnmarshalString(*book.MetadataJSON, &meta); err == nil {
			if _, exists := payload["author"]; !exists {
				if meta.Creator != "" {
					payload["author"] = meta.Creator
				} else if len(meta.Creators) > 0 {
					payload["author"] = strings.Join(meta.Creators, ", ")
				}
			}
			if _, exists := payload["description"]; !exists && meta.Description != "" {
				payload["description"] = meta.Description
			}
			pub := strings.TrimSpace(meta.Publisher)
			if pub == "" && len(meta.Publishers) > 0 {
				pub = strings.Join(meta.Publishers, ", ")
			}
			if pub != "" {
				payload["publisher"] = pub
			}
			if meta.Language != "" {
				payload["language"] = meta.Language
			}
			if meta.Date != "" {
				payload["date"] = meta.Date
			}
			if meta.Series != "" {
				payload["series"] = meta.Series
			}
			if meta.SeriesIndex != "" {
				payload["series_index"] = meta.SeriesIndex
			}

			var tags []string
			switch v := meta.Subject.(type) {
			case string:
				if v != "" {
					tags = []string{v}
				}
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok && str != "" {
						tags = append(tags, str)
					}
				}
			}
			if len(tags) > 0 {
				payload["tags"] = tags
			}
		}
	}

	return payload
}

func (s *webhookService) formatBodyByTemplate(templateType, eventType string, rawPayloadBytes []byte) ([]byte, string) {
	var rawData map[string]any
	_ = jsonx.Unmarshal(rawPayloadBytes, &rawData)

	switch templateType {
	case "discord":
		eventTitles := map[string]string{
			"book.created":      "📚 New Book Added",
			"book.deleted":      "🗑️ Book Deleted",
			"metadata.updated":  "📝 Book Metadata Updated",
			"reading.completed": "🎉 Reading Completed",
		}
		titleText, ok := eventTitles[eventType]
		if !ok {
			titleText = fmt.Sprintf("📚 NovelHub Notification: %s", eventType)
		}

		color := 3447003 // Blue for created
		if eventType == "book.deleted" {
			color = 15158332 // Red
		} else if eventType == "reading.completed" {
			color = 3066993 // Green
		} else if eventType == "metadata.updated" {
			color = 15844367 // Gold
		}

		if customHex, ok := rawData["_embed_color"].(string); ok && strings.TrimSpace(customHex) != "" {
			hexStr := strings.TrimPrefix(customHex, "#")
			if parsedColor, err := strconv.ParseInt(hexStr, 16, 64); err == nil {
				color = int(parsedColor)
			}
		}

		embed := map[string]any{
			"title":     titleText,
			"color":     color,
			"timestamp": time.Now().Format(time.RFC3339),
			"footer": map[string]string{
				"text": fmt.Sprintf("NovelHub • Event: %s", eventType),
			},
		}

		titleTmpl := "📚 {title}"
		if customTmpl, ok := rawData["_title_template"].(string); ok && strings.TrimSpace(customTmpl) != "" {
			titleTmpl = customTmpl
		}

		if bookTitle, ok := rawData["title"].(string); ok && bookTitle != "" {
			embed["title"] = strings.ReplaceAll(titleTmpl, "{title}", bookTitle)
		}

		if desc, ok := rawData["description"].(string); ok && strings.TrimSpace(desc) != "" {
			descClean := strings.TrimSpace(desc)
			if len(descClean) > 500 {
				descClean = descClean[:497] + "..."
			}
			embed["description"] = descClean
		}

		if coverURL, ok := rawData["cover_url"].(string); ok && strings.TrimSpace(coverURL) != "" {
			embed["thumbnail"] = map[string]string{
				"url": coverURL,
			}
		}

		labels := map[string]string{
			"author":    "👤 Author",
			"publisher": "🏢 Publisher",
			"language":  "🌐 Language",
			"series":    "📖 Series",
			"date":      "📅 Release Date",
			"tags":      "🏷️ Tags",
			"event":     "⚡ Event",
		}
		if customLabels, ok := rawData["_field_labels"].(map[string]interface{}); ok {
			for k, v := range customLabels {
				if str, ok := v.(string); ok && strings.TrimSpace(str) != "" {
					labels[k] = str
				}
			}
		}

		var fields []map[string]any
		if author, ok := rawData["author"].(string); ok && author != "" {
			fields = append(fields, map[string]any{"name": labels["author"], "value": author, "inline": true})
		}
		if publisher, ok := rawData["publisher"].(string); ok && publisher != "" {
			fields = append(fields, map[string]any{"name": labels["publisher"], "value": publisher, "inline": true})
		}
		if lang, ok := rawData["language"].(string); ok && lang != "" {
			fields = append(fields, map[string]any{"name": labels["language"], "value": lang, "inline": true})
		}
		if series, ok := rawData["series"].(string); ok && series != "" {
			seriesVal := series
			if idx, ok := rawData["series_index"].(string); ok && idx != "" {
				seriesVal += fmt.Sprintf(" (Vol. %s)", idx)
			}
			fields = append(fields, map[string]any{"name": labels["series"], "value": seriesVal, "inline": true})
		}
		if dateVal, ok := rawData["date"].(string); ok && dateVal != "" {
			fields = append(fields, map[string]any{"name": labels["date"], "value": dateVal, "inline": true})
		}
		if tags, ok := rawData["tags"].([]interface{}); ok && len(tags) > 0 {
			var tagStrs []string
			for _, t := range tags {
				if ts, ok := t.(string); ok {
					tagStrs = append(tagStrs, ts)
				}
			}
			if len(tagStrs) > 0 {
				fields = append(fields, map[string]any{"name": labels["tags"], "value": strings.Join(tagStrs, ", "), "inline": false})
			}
		}
		fields = append(fields, map[string]any{"name": labels["event"], "value": fmt.Sprintf("`%s`", eventType), "inline": true})

		embed["fields"] = fields

		discordMsg := map[string]any{
			"embeds": []map[string]any{embed},
		}
		b, _ := jsonx.Marshal(discordMsg)
		return b, "application/json"

	case "telegram":
		bookTitle, _ := rawData["title"].(string)
		author, _ := rawData["author"].(string)
		publisher, _ := rawData["publisher"].(string)
		lang, _ := rawData["language"].(string)
		series, _ := rawData["series"].(string)
		desc, _ := rawData["description"].(string)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<b>📚 NovelHub Event: %s</b>\n", eventType))
		if bookTitle != "" {
			sb.WriteString(fmt.Sprintf("<b>📖 Book:</b> %s\n", bookTitle))
		}
		if author != "" {
			sb.WriteString(fmt.Sprintf("<b>👤 Author:</b> %s\n", author))
		}
		if publisher != "" {
			sb.WriteString(fmt.Sprintf("<b>🏢 Publisher:</b> %s\n", publisher))
		}
		if lang != "" {
			sb.WriteString(fmt.Sprintf("<b>🌐 Language:</b> %s\n", lang))
		}
		if series != "" {
			sb.WriteString(fmt.Sprintf("<b>📖 Series:</b> %s\n", series))
		}
		if desc != "" {
			descClean := strings.TrimSpace(desc)
			if len(descClean) > 250 {
				descClean = descClean[:247] + "..."
			}
			sb.WriteString(fmt.Sprintf("<b>📝 Description:</b> %s\n", descClean))
		}

		telegramMsg := map[string]any{
			"text":       sb.String(),
			"parse_mode": "HTML",
		}
		b, _ := jsonx.Marshal(telegramMsg)
		return b, "application/json"

	case "slack":
		bookTitle, _ := rawData["title"].(string)
		author, _ := rawData["author"].(string)
		publisher, _ := rawData["publisher"].(string)
		lang, _ := rawData["language"].(string)

		mrkdwn := fmt.Sprintf("*📚 NovelHub: %s*\n*Book:* %s\n*Author:* %s\n*Publisher:* %s\n*Language:* %s",
			eventType, bookTitle, author, publisher, lang)

		section := map[string]any{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": mrkdwn,
			},
		}

		if coverURL, ok := rawData["cover_url"].(string); ok && strings.TrimSpace(coverURL) != "" {
			section["accessory"] = map[string]string{
				"type":      "image",
				"image_url": coverURL,
				"alt_text":  bookTitle,
			}
		}

		slackMsg := map[string]any{
			"blocks": []map[string]any{section},
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
