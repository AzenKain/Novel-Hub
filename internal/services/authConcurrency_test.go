package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// Logout reads the user then writes token_version — a read-then-write transaction.
func TestConcurrentLogoutsDoNotHitBusySnapshot(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "logout.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider, token_version) VALUES ('01920000-0000-7000-8000-0000000000f1','multi@n.h','LOCAL',1)`); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	svc := NewAuthService(
		repositories.NewUserRepository(db, c),
		repositories.NewRoleRepository(db, c),
		database.NewTxManager(db),
		repositories.NewSettingsRepository(db, c),
		nil,
	)

	const devices = 12
	var wg sync.WaitGroup
	errs := make(chan error, devices)
	for range devices {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.Logout(context.Background(), "01920000-0000-7000-8000-0000000000f1"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent logout failed: %v", err)
	}

	var version int64
	if err := db.QueryRow(`SELECT token_version FROM users WHERE id='01920000-0000-7000-8000-0000000000f1'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != devices+1 {
		t.Fatalf("token_version = %d after %d logouts, want %d", version, devices, devices+1)
	}
}

var _ = sql.ErrNoRows
