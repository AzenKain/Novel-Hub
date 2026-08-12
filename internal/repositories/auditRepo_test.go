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

// TestAuditSmartFilterLeaksCrossUserReadingStatus proves task T2.1: the
// SearchSmartFilterBookIDs query (db/query/books.sql:151-153) filters on
// reading_progress without any user_id scoping, and SearchSmartFilterBooks
// takes no userID at all. One user's read state therefore leaks into every
// other user's smart-filter results.
//
// Setup: u1 has read b1, u2 has read b2. Nothing else differs.
// u1's "status = read" filter must return ONLY b1. PASSING the current
// behaviour returns b1 AND b2 — proof that u2's reading state leaked to u1.
func TestAuditSmartFilterLeaksCrossUserReadingStatus(t *testing.T) {
	db := newAuditRepoTestDB(t)
	ctx := context.Background()
	seedAuditLibraryAndBooks(t, db, "b1", "b2")

	if _, err := db.Exec(`INSERT INTO users (id, email) VALUES ('u1', 'u1@example.com'), ('u2', 'u2@example.com')`); err != nil {
		t.Fatal(err)
	}
	// u1 has read b1 to completion; u2 has read b2 to completion.
	for _, row := range []struct{ uid, bid string }{{"u1", "b1"}, {"u2", "b2"}} {
		if _, err := db.Exec(`INSERT INTO reading_progress (user_id, book_id, chapter_ref, progress_percent) VALUES (?, ?, 'c1', 100)`, row.uid, row.bid); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewBookDBRepository(db, cache.NewRamCache())
	rule := request.SmartFilterRuleItemDto{Field: "status", Value: "read"}
	books, err := repo.SearchSmartFilterBooks(ctx, nil, []request.SmartFilterRuleItemDto{rule}, nil, "", 10)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(books))
	for _, b := range books {
		got = append(got, b.ID)
	}

	leaked := slices.Contains(got, "b2")
	// BUG PROOF: u1's "read" filter surfaced b2, which only u2 has read.
	if !leaked {
		t.Fatalf("no leak to prove: u1's read filter returned %v; b2 (u2's book) was correctly excluded", got)
	}
	if !slices.Contains(got, "b1") {
		t.Fatalf("setup broken: u1's own read book b1 missing from %v", got)
	}
}

// TestAuditSpineResyncOrphansHighlights proves task T2.2: spine re-sync
// (maintenanceService.IndexBook) deletes and recreates every chapter with a new
// UUID, but highlights carry chapter_id with no FK to chapters and are never
// cleaned up — so they end up pointing at chapters that no longer exist.
//
// PASSING = orphan proved: after the chapter is deleted, the highlight row
// still exists and references the dead chapter id.
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

	// maintenanceService.IndexBook does exactly this when the spine changes.
	if err := bookRepo.DeleteChaptersByBook(ctx, "b1"); err != nil {
		t.Fatal(err)
	}

	// The chapter is gone...
	if _, err := bookRepo.GetChapter(ctx, "ch-old"); err == nil {
		t.Fatal("setup broken: chapter still exists after DeleteChaptersByBook")
	}

	// ...but the highlight survives, pointing at nothing.
	var orphanCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM highlights WHERE book_id = 'b1' AND chapter_id = 'ch-old'`).Scan(&orphanCount); err != nil {
		t.Fatal(err)
	}
	// BUG PROOF: a highlight references a deleted chapter.
	if orphanCount == 0 {
		t.Fatalf("no orphan to prove: highlights were cleaned up with their chapters")
	}
}
