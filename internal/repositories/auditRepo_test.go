package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newAuditRepoTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "audit_repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedAuditLibraryAndBooks(t *testing.T, db *sql.DB, bookIDs ...string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Audit Lib')`); err != nil {
		t.Fatal(err)
	}
	for _, id := range bookIDs {
		if _, err := db.Exec(`INSERT INTO books (id, library_id, title) VALUES (?, 'lib-1', 'Book ' || ?)`, id, id); err != nil {
			t.Fatal(err)
		}
	}
}

// TestAuditSmartFilterLeaksCrossUserReadingStatus proves task T2.1: the SearchSmartFilterBookIDs query (db/query/books.sql:151-153) filters on reading_progress without any user_id scoping, and SearchSmartFilterBooks takes no userID at all.
func TestAuditSmartFilterLeaksCrossUserReadingStatus(t *testing.T) {
	db := newAuditRepoTestDB(t)
	ctx := context.Background()
	seedAuditLibraryAndBooks(t, db, "b1", "b2")

	if _, err := db.Exec(`INSERT INTO users (id, email) VALUES ('u1', 'u1@example.com'), ('u2', 'u2@example.com')`); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ uid, bid string }{{"u1", "b1"}, {"u2", "b2"}} {
		if _, err := db.Exec(`INSERT INTO reading_progress (user_id, book_id, chapter_ref, progress_percent) VALUES (?, ?, 'c1', 100)`, row.uid, row.bid); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewBookDBRepository(db, cache.NewRamCache())
	rule := request.SmartFilterRuleItemDto{Field: "status", Value: "read"}
	books, err := repo.SearchSmartFilterBooks(ctx, nil, []request.SmartFilterRuleItemDto{rule}, nil, "", 10, "u1")
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(books))
	for _, b := range books {
		got = append(got, b.ID)
	}

	if slices.Contains(got, "b2") {
		t.Fatalf("u1's read filter leaked b2: %v", got)
	}
	if !slices.Contains(got, "b1") {
		t.Fatalf("setup broken: u1's own read book b1 missing from %v", got)
	}
}

// TestAuditSpineResyncOrphansHighlights proves task T2.2: spine re-sync (maintenanceService.IndexBook) deletes and recreates every chapter with a new UUID, but highlights carry chapter_id with no FK to chapters and are never cleaned up — so they end up pointing at chapters that no longer exist.
func TestAuditSpineResyncOrphansHighlights(t *testing.T) {
	db := newAuditRepoTestDB(t)
	ctx := context.Background()
	seedAuditLibraryAndBooks(t, db, "b1")

	if _, err := db.Exec(`INSERT INTO users (id, email) VALUES ('u1', 'u1@example.com')`); err != nil {
		t.Fatal(err)
	}

	bookRepo := NewBookDBRepository(db, cache.NewRamCache())
	chapter := &models.ChapterEntity{ID: "ch-old", BookID: "b1", Title: "Chapter 1", ChapterIndex: 0}
	if err := bookRepo.CreateChapter(ctx, chapter); err != nil {
		t.Fatal(err)
	}

	hlRepo := NewHighlightRepository(db, cache.NewRamCache())
	if _, err := hlRepo.Create(ctx, sqlc.CreateHighlightParams{
		ID:          "hl-1",
		UserID:      "u1",
		BookID:      "b1",
		ChapterID:   "ch-old",
		TextContent: "orphaned text",
		StartIndex:  0,
		EndIndex:    5,
		Color:       "yellow",
	}); err != nil {
		t.Fatal(err)
	}

	if err := bookRepo.DeleteChaptersByBook(ctx, "b1"); err != nil {
		t.Fatal(err)
	}

	if _, err := bookRepo.GetChapter(ctx, "ch-old"); err == nil {
		t.Fatal("setup broken: chapter still exists after DeleteChaptersByBook")
	}

	var orphanCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM highlights WHERE book_id = 'b1' AND chapter_id = 'ch-old'`).Scan(&orphanCount); err != nil {
		t.Fatal(err)
	}
	if orphanCount == 0 {
		t.Fatalf("no orphan to prove: highlights were cleaned up with their chapters")
	}
}
