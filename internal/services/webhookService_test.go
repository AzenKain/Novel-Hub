package services

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/database"
)

func TestWebhookListAll(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	repo := repositories.NewWebhookRepository(db, nil)
	svc := NewWebhookService(repo, nil)

	ctx := context.Background()

	created, err := svc.Create(ctx, &request.CreateWebhookDto{
		Name:         "Test Hook",
		URL:          "https://example.com/hook",
		TemplateType: "generic",
		Events:       []string{"book.created"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	t.Logf("Created webhook ID: %s", created.ID)

	list, err := svc.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	t.Logf("ListAll returned %d webhooks", len(list))
}
