package services

import (
	"context"
	"database/sql"
	"testing"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/crypto"
)

func seedTrackerTestUser(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if _, err := sqlc.New(db).UpsertUser(context.Background(), sqlc.UpsertUserParams{
		ID:           id,
		Email:        "tracker-" + id + "@example.com",
		AuthProvider: "local",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGetUserTrackerConnectionsReportsPerProviderState(t *testing.T) {
	t.Setenv("DB_ENCRYPTION_KEY", "tracker-test-encryption-key")
	db := auditDB(t)
	repo := repositories.NewTrackerRepository(db, cache.NewRamCache())
	svc := NewTrackerService(repo)
	seedTrackerTestUser(t, db, "1")

	if err := svc.SaveUserTracker(context.Background(), "1", "readwise", "rw-secret-token"); err != nil {
		t.Fatal(err)
	}

	conns, err := svc.GetUserTrackerConnections(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(conns))
	}
	byProvider := map[string]bool{}
	for _, c := range conns {
		byProvider[c.Provider] = c.Connected
	}
	if !byProvider["readwise"] {
		t.Fatal("readwise should be connected")
	}
	if byProvider["anilist"] || byProvider["hardcover"] {
		t.Fatal("anilist/hardcover should not be connected")
	}
}

func TestGetUserTrackerConnectionsNoRowsIsNotConnected(t *testing.T) {
	db := auditDB(t)
	repo := repositories.NewTrackerRepository(db, cache.NewRamCache())
	svc := NewTrackerService(repo)
	seedTrackerTestUser(t, db, "42")

	conns, err := svc.GetUserTrackerConnections(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range conns {
		if c.Connected {
			t.Fatalf("provider %s should not be connected for a fresh user", c.Provider)
		}
	}
}

func TestTrackerTokenStoredEncrypted(t *testing.T) {
	t.Setenv("DB_ENCRYPTION_KEY", "tracker-test-encryption-key")
	db := auditDB(t)
	repo := repositories.NewTrackerRepository(db, cache.NewRamCache())
	seedTrackerTestUser(t, db, "7")
	const secret = "plaintext-token-must-not-hit-disk"

	if _, err := repo.UpsertUserTracker(context.Background(), "7", "readwise", secret, nil, nil); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.QueryRowContext(context.Background(),
		"SELECT access_token FROM user_trackers WHERE user_id = ?", "7").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == secret {
		t.Fatal("access_token stored in plaintext")
	}
	decrypted, err := crypto.DecryptAES(stored)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != secret {
		t.Fatal("decrypted token mismatch")
	}
}
