package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

type allowAllPermissions struct{}

func (allowAllPermissions) Reload(context.Context) error { return nil }
func (allowAllPermissions) Can(context.Context, string, string, map[string]any) bool {
	return true
}
func (allowAllPermissions) CanRoles([]string, []constants.RoleType, string, map[string]any) bool {
	return true
}
func (allowAllPermissions) IsAdmin([]string, []constants.RoleType) bool { return true }
func (allowAllPermissions) GetGuestPermissions() []string               { return nil }
func (allowAllPermissions) DescribeRoles([]string) []*models.RoleSimple { return nil }

func newActivityService(t *testing.T) (*featureService, *sql.DB) {
	t.Helper()
	// NewSQLiteDB, not sql.Open: the production DSN sets busy_timeout and WAL, which is
	// exactly what decides whether concurrent writers block or fail with SQLITE_BUSY.
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "activity.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib','Lib')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title) VALUES ('book','lib','Book')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, email) VALUES ('user','u@n.h')`); err != nil {
		t.Fatal(err)
	}
	c := cache.NewRamCache()
	return &featureService{
		repo:        repositories.NewFeatureRepository(db, c),
		bookRepo:    repositories.NewBookDBRepository(db, c),
		permissions: allowAllPermissions{},
		txManager:   database.NewTxManager(db),
	}, db
}

// opened_count used to be read in Go, incremented, and written back absolutely, so a
// process-global mutex was the only thing keeping concurrent readers from clobbering each
// other's count. That mutex serialized every user's reading writes. The increment now
// happens in SQL, which makes the write atomic per row without a shared lock — this test
// is what proves it: with the old read-modify-write it loses increments.
func TestRecordReadingActivityCountsEveryConcurrentOpen(t *testing.T) {
	svc, db := newActivityService(t)
	ctx := context.Background()

	const opens = 20
	var wg sync.WaitGroup
	errs := make(chan error, opens)
	for i := range opens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := svc.RecordReadingActivity(ctx, models.ReadingActivityInput{
				UserID:       "user",
				BookID:       "book",
				ChapterID:    "chapter-1",
				ChapterIndex: int64(i),
				EventType:    "chapter_open",
			}, &response.JWTClaims{UId: "user"})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent RecordReadingActivity failed: %v", err)
	}

	var openedCount int64
	if err := db.QueryRow(`SELECT opened_count FROM reading_progress WHERE user_id='user' AND book_id='book'`).Scan(&openedCount); err != nil {
		t.Fatal(err)
	}
	if openedCount != opens {
		t.Fatalf("opened_count = %d after %d concurrent opens, want %d", openedCount, opens, opens)
	}
}
