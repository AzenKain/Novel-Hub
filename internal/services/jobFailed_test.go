package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
)

// A nil queue makes DispatchEvent a silent no-op, so the queue must be real here.
func jobFailedHarness(t *testing.T) (*jobService, <-chan string) {
	t.Helper()
	t.Setenv("DB_ENCRYPTION_KEY", "job-failed-test-key")
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "job-failed.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

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
	queue.RegisterHandler("database_health_check", func(ctx context.Context, jobID string, payload string) error {
		return nil
	})
	queue.Start()
	t.Cleanup(queue.Stop)

	if _, err := webhooks.Create(ctx, &request.CreateWebhookDto{
		Name:         "Ops pager",
		URL:          "mailto:ops@example.com",
		TemplateType: "email",
		Events:       []string{"job.failed"},
	}); err != nil {
		t.Fatal(err)
	}

	service := NewJobService(repositories.NewJobRepository(db, c), queue)
	service.SetWebhookService(webhooks)
	queue.SetLifecycle(service)
	return service, received
}

func TestJobFailedDispatchesWebhookOnce(t *testing.T) {
	service, received := jobFailedHarness(t)
	ctx := context.Background()

	job, err := service.Trigger(ctx, "database_health_check", "")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case message := <-received:
		t.Fatalf("a successful job paged the operator:\n%s", message)
	case <-time.After(400 * time.Millisecond):
	}

	if err := service.Failed(ctx, worker.Job{ID: job.ID, Type: job.Type}, errors.New("disk is on fire")); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-received:
		for _, want := range []string{"job.failed", "database_health_check", "disk is on fire"} {
			if !strings.Contains(message, want) {
				t.Errorf("job.failed payload is missing %q:\n%s", want, message)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("an exhausted job did not dispatch job.failed")
	}
}

// Failed must record the status even with no webhook service injected.
func TestJobFailedStillRecordsWithoutWebhookService(t *testing.T) {
	service, _ := jobFailedHarness(t)
	service.SetWebhookService(nil)
	ctx := context.Background()

	job, err := service.Trigger(ctx, "database_health_check", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Failed(ctx, worker.Job{ID: job.ID, Type: job.Type}, errors.New("boom")); err != nil {
		t.Fatal(err)
	}

	stored, err := service.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status == nil || *stored.Status != "failed" {
		t.Fatalf("status = %v, want failed", stored.Status)
	}
	if stored.ErrorMsg == nil || *stored.ErrorMsg != "boom" {
		t.Fatalf("error_msg = %v, want boom", stored.ErrorMsg)
	}
}
