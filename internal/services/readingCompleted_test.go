package services

import (
	"context"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
)

func progressAt(percent float64) *float64 { return &percent }

func completedWebhookHarness(t *testing.T) (*featureService, <-chan string) {
	t.Helper()
	t.Setenv("DB_ENCRYPTION_KEY", "reading-completed-test-key")
	svc, db := newActivityService(t)
	ctx := context.Background()

	c := cache.NewRamCache()
	settings := NewSettingsService(repositories.NewSettingsRepository(db, c), database.NewTxManager(db))
	if err := settings.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	port, received := captureOTPMail(t)
	enableCaptureSMTP(t, settings, port)

	queue := worker.NewQueue(1)
	webhooks := NewWebhookService(repositories.NewWebhookRepository(db, c), queue, settings)
	queue.RegisterHandler("webhook.dispatch", func(ctx context.Context, jobID string, payload string) error {
		var job struct {
			WebhookID string `json:"webhook_id"`
			EventType string `json:"event_type"`
			Data      string `json:"data"`
		}
		if err := jsonx.UnmarshalString(payload, &job); err != nil {
			return err
		}
		return webhooks.ExecuteDispatch(ctx, job.WebhookID, job.EventType, []byte(job.Data))
	})
	queue.Start()
	t.Cleanup(queue.Stop)

	if _, err := webhooks.Create(ctx, &request.CreateWebhookDto{
		Name:         "Finished shelf",
		URL:          "mailto:ops@example.com",
		TemplateType: "email",
		Events:       []string{"reading.completed"},
	}); err != nil {
		t.Fatal(err)
	}
	svc.SetWebhookService(webhooks)
	return svc, received
}

// reading.completed was advertised in the UI but nothing ever dispatched it.
func TestReadingCompletedFiresOnceWhenFinished(t *testing.T) {
	svc, received := completedWebhookHarness(t)
	ctx := context.Background()

	record := func(percent float64) {
		t.Helper()
		if _, err := svc.RecordReadingActivity(ctx, models.ReadingActivityInput{
			UserID:          "user",
			BookID:          "book",
			ChapterID:       "chapter-1",
			ProgressPercent: progressAt(percent),
			EventType:       "progress",
		}, &response.JWTClaims{UId: "user"}); err != nil {
			t.Fatal(err)
		}
	}

	record(42)
	select {
	case message := <-received:
		t.Fatalf("a half-read book was announced as completed:\n%s", message)
	case <-time.After(400 * time.Millisecond):
	}

	record(100)
	select {
	case message := <-received:
		for _, want := range []string{"reading.completed", "Book", "user"} {
			if !strings.Contains(message, want) {
				t.Errorf("completed event is missing %q:\n%s", want, message)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("finishing the book did not dispatch reading.completed")
	}

	record(100)
	select {
	case message := <-received:
		t.Fatalf("reading.completed fired twice for the same book:\n%s", message)
	case <-time.After(400 * time.Millisecond):
	}
}

func TestReadingCompletedUsesTheCatalogThreshold(t *testing.T) {
	if bookCompletedPercent != 99.5 {
		t.Fatalf("bookCompletedPercent = %v, want 99.5 to match db/query/books.sql", bookCompletedPercent)
	}

	svc, received := completedWebhookHarness(t)
	ctx := context.Background()

	if _, err := svc.RecordReadingActivity(ctx, models.ReadingActivityInput{
		UserID:          "user",
		BookID:          "book",
		ChapterID:       "chapter-1",
		ProgressPercent: progressAt(99.4),
		EventType:       "progress",
	}, &response.JWTClaims{UId: "user"}); err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-received:
		t.Fatalf("99.4%% is still 'reading' in the catalog but was announced completed:\n%s", message)
	case <-time.After(400 * time.Millisecond):
	}
}
