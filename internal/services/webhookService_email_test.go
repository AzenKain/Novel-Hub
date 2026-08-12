package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newEmailWebhookService(t *testing.T) (WebhookService, SettingsService) {
	t.Helper()
	t.Setenv("DB_ENCRYPTION_KEY", "webhook-email-test-key")
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "webhook-email.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	settings := NewSettingsService(repositories.NewSettingsRepository(db, c), database.NewTxManager(db))
	if err := settings.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	return NewWebhookService(repositories.NewWebhookRepository(db, c), nil, settings), settings
}

func enableCaptureSMTP(t *testing.T, settings SettingsService, port int) {
	t.Helper()
	if _, err := settings.UpdateSettings(context.Background(), map[string]any{
		"smtp.enabled":                true,
		"smtp.host":                   "localhost",
		"smtp.port":                   port,
		"smtp.from_email":             "library@example.com",
		"smtp.tls_mode":               "none",
		"smtp.allow_private_networks": true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookEmailTemplateDelivers(t *testing.T) {
	svc, settings := newEmailWebhookService(t)
	ctx := context.Background()
	port, received := captureOTPMail(t)
	enableCaptureSMTP(t, settings, port)

	created, err := svc.Create(ctx, &request.CreateWebhookDto{
		Name:         "Ops mailbox",
		URL:          "mailto:ops@example.com, ops@example.com ,second@example.com",
		TemplateType: "email",
		Events:       []string{"book.created"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ExecuteDispatch(ctx, created.ID, "book.created", []byte(`{"title":"Dune","id":"abc"}`)); err != nil {
		t.Fatalf("email webhook dispatch failed: %v", err)
	}

	seen := make([]string, 0, 2)
	for range 2 {
		select {
		case message := <-received:
			seen = append(seen, message)
		case <-time.After(10 * time.Second):
			t.Fatalf("only %d message(s) delivered, want 2 (duplicate recipient must be collapsed)", len(seen))
		}
	}
	joined := strings.Join(seen, "\n")
	for _, want := range []string{"ops@example.com", "second@example.com", "book.created", "Ops mailbox", "Dune"} {
		if !strings.Contains(joined, want) {
			t.Errorf("delivered mail is missing %q:\n%s", want, joined)
		}
	}

	select {
	case extra := <-received:
		t.Fatalf("a third message was sent, so the duplicate recipient was not deduplicated:\n%s", extra)
	default:
	}
}

func TestWebhookEmailRequiresConfiguredSMTP(t *testing.T) {
	svc, _ := newEmailWebhookService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &request.CreateWebhookDto{
		Name:         "Ops mailbox",
		URL:          "mailto:ops@example.com",
		TemplateType: "email",
		Events:       []string{"book.created"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ExecuteDispatch(ctx, created.ID, "book.created", []byte(`{}`)); err == nil {
		t.Fatal("email webhook reported success while SMTP was disabled")
	}
}

func TestWebhookTargetMustMatchTransport(t *testing.T) {
	svc, _ := newEmailWebhookService(t)
	ctx := context.Background()

	cases := []struct {
		name         string
		templateType string
		url          string
	}{
		{"email with https target", "email", "https://example.com/hook"},
		{"email without recipients", "email", "mailto:"},
		{"email with malformed address", "email", "mailto:not-an-address"},
		{"email with header injection", "email", "mailto:ops@example.com%0ABcc:victim@example.com"},
		{"http webhook with mailto target", "discord", "mailto:ops@example.com"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := svc.Create(ctx, &request.CreateWebhookDto{
				Name:         "Bad target",
				URL:          test.url,
				TemplateType: test.templateType,
				Events:       []string{"book.created"},
			}); err == nil {
				t.Fatalf("%s was accepted", test.name)
			}
		})
	}
}

func TestWebhookEmailNeverUsesHTTPClient(t *testing.T) {
	svc, settings := newEmailWebhookService(t)
	ctx := context.Background()
	port, received := captureOTPMail(t)
	enableCaptureSMTP(t, settings, port)

	created, err := svc.Create(ctx, &request.CreateWebhookDto{
		Name:         "Ops",
		URL:          "mailto:ops@example.com",
		TemplateType: "email",
		Events:       []string{"metadata.updated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ExecuteDispatch(ctx, created.ID, "metadata.updated", []byte(`{"title":"X"}`)); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	select {
	case <-received:
	case <-time.After(10 * time.Second):
		t.Fatal("no mail arrived, so the email webhook did not use SMTP")
	}
}

// HTTP webhooks must keep working without a SettingsService.
func TestWebhookListAllStillWorksWithoutSettings(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	svc := NewWebhookService(repositories.NewWebhookRepository(db, nil), nil)
	ctx := context.Background()
	if _, err := svc.Create(ctx, &request.CreateWebhookDto{
		Name:         "Test Hook",
		URL:          "https://example.com/hook",
		TemplateType: "generic",
		Events:       []string{"book.created"},
	}); err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListAll(ctx, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("ListAll returned %d webhooks, want 1", len(list))
	}
}
