package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newAuditServiceForTest(t *testing.T) (AuditService, func(id string, createdAt string)) {
	t.Helper()
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "audit.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db, filepath.Join("..", "..", "db", "schema")); err != nil {
		t.Fatal(err)
	}

	repo := repositories.NewAuditRepository(db, cache.NewRamCache())
	insert := func(id string, createdAt string) {
		t.Helper()
		if _, err := db.Exec(`
			INSERT INTO audit_logs (id, actor_email, action, target_type, ip, created_at)
			VALUES (?, 'seed@example.com', 'user.update', 'user', '127.0.0.1', `+createdAt+`)
		`, id); err != nil {
			t.Fatal(err)
		}
	}
	return NewAuditService(repo, nil), insert
}

func TestAuditPruneKeepsRowsInsideRetentionWindow(t *testing.T) {
	service, insert := newAuditServiceForTest(t)
	ctx := context.Background()

	recent := uuid.Must(uuid.NewV7()).String()
	edge := uuid.Must(uuid.NewV7()).String()
	old := uuid.Must(uuid.NewV7()).String()
	insert(recent, "datetime('now', '-1 days')")
	insert(edge, "datetime('now', '-89 days')")
	insert(old, "datetime('now', '-91 days')")

	deleted, err := service.Prune(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("pruned %d rows, want exactly the one older than %d days", deleted, auditRetentionDays)
	}

	page, err := service.List(ctx, &request.ListAuditLogsDto{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Pagination.TotalRecords != 2 {
		t.Fatalf("total after prune = %d, want 2; prune ate rows inside the window", page.Pagination.TotalRecords)
	}
}

func TestAuditRecordWithoutActorStillWritesTheRow(t *testing.T) {
	service, _ := newAuditServiceForTest(t)
	ctx := context.Background()

	service.Record(ctx, AuditActionSettingsUpdate, "settings", "", "smtp.host")

	page, err := service.List(ctx, &request.ListAuditLogsDto{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Pagination.TotalRecords != 1 {
		t.Fatalf("rows = %d, want 1; an actor-less mutation must still leave a trace", page.Pagination.TotalRecords)
	}
}

func TestAuditListFiltersByAction(t *testing.T) {
	service, _ := newAuditServiceForTest(t)
	ctx := WithAuditActor(context.Background(), AuditActor{Email: "admin@example.com", IP: "10.0.0.1"})

	service.Record(ctx, AuditActionUserDelete, "user", uuid.Must(uuid.NewV7()).String(), "gone@example.com")
	service.Record(ctx, AuditActionRoleCreate, "role", uuid.Must(uuid.NewV7()).String(), "EDITOR")

	page, err := service.List(ctx, &request.ListAuditLogsDto{Action: AuditActionRoleCreate})
	if err != nil {
		t.Fatal(err)
	}
	if page.Pagination.TotalRecords != 1 {
		t.Fatalf("filtered total = %d, want 1", page.Pagination.TotalRecords)
	}

	actions, err := service.ListActions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 {
		t.Fatalf("distinct actions = %v, want the two recorded ones", actions)
	}
}
