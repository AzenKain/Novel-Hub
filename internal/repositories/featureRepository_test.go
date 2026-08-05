package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/database"
)

const featureCursorTimestamp = "2026-08-03 14:59:21"

func newFeatureCursorTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "feature-cursor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('library-1', 'Library')`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"book-1", "book-2", "book-3"} {
		if _, err := db.Exec(`INSERT INTO books (id, library_id, title) VALUES (?, 'library-1', ?)`, id, id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"user-1", "user-2", "user-3"} {
		if _, err := db.Exec(`INSERT INTO users (id, email) VALUES (?, ?)`, id, id+"@example.test"); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func featureCursorTime(t *testing.T) time.Time {
	t.Helper()
	value, err := time.Parse("2006-01-02 15:04:05", featureCursorTimestamp)
	if err != nil {
		t.Fatal(err)
	}
	return value.UTC()
}

func TestFeatureCursorTiebreakers(t *testing.T) {
	db := newFeatureCursorTestDB(t)
	ctx := context.Background()
	repo := NewFeatureRepository(db, nil)

	for _, id := range []string{"collection-1", "collection-2", "collection-3"} {
		if _, err := db.Exec(`INSERT INTO collections (id, user_id, name, created_at) VALUES (?, 'user-1', ?, ?)`, id, id, featureCursorTimestamp); err != nil {
			t.Fatal(err)
		}
	}
	for _, bookID := range []string{"book-1", "book-2", "book-3"} {
		if _, err := db.Exec(`INSERT INTO reading_progress (user_id, book_id, chapter_ref, updated_at) VALUES ('user-1', ?, 'chapter', ?)`, bookID, featureCursorTimestamp); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO bookmarks (user_id, book_id, created_at) VALUES ('user-1', ?, ?)`, bookID, featureCursorTimestamp); err != nil {
			t.Fatal(err)
		}
	}
	for _, userID := range []string{"user-1", "user-2", "user-3"} {
		if _, err := db.Exec(`INSERT INTO book_reviews (user_id, book_id, rating, updated_at) VALUES (?, 'book-1', 5, ?)`, userID, featureCursorTimestamp); err != nil {
			t.Fatal(err)
		}
	}

	cursorTime := featureCursorTime(t)

	collections, err := repo.GetUserCollections(ctx, "user-1", nil, "", 2)
	if err != nil || len(collections) != 2 {
		t.Fatalf("collections page 1: %v, len=%d", err, len(collections))
	}
	collections2, err := repo.GetUserCollections(ctx, "user-1", &cursorTime, collections[1].ID, 2)
	if err != nil || len(collections2) != 1 || collections2[0].ID == collections[0].ID || collections2[0].ID == collections[1].ID {
		t.Fatalf("collections page 2: %#v, %v", collections2, err)
	}

	history, err := repo.GetRecentReadingHistory(ctx, "user-1", nil, "", 2)
	if err != nil || len(history) != 2 {
		t.Fatalf("history page 1: %v, len=%d", err, len(history))
	}
	history2, err := repo.GetRecentReadingHistory(ctx, "user-1", &cursorTime, history[1].BookID, 2)
	if err != nil || len(history2) != 1 || history2[0].BookID == history[0].BookID || history2[0].BookID == history[1].BookID {
		t.Fatalf("history page 2: %#v, %v", history2, err)
	}

	bookmarks, err := repo.GetBookmarkedBooks(ctx, "user-1", nil, "", 2)
	if err != nil || len(bookmarks) != 2 {
		t.Fatalf("bookmarks page 1: %v, len=%d", err, len(bookmarks))
	}
	bookmarks2, err := repo.GetBookmarkedBooks(ctx, "user-1", &cursorTime, bookmarks[1].BookID, 2)
	if err != nil || len(bookmarks2) != 1 || bookmarks2[0].BookID == bookmarks[0].BookID || bookmarks2[0].BookID == bookmarks[1].BookID {
		t.Fatalf("bookmarks page 2: %#v, %v", bookmarks2, err)
	}

	reviews, err := repo.ListBookReviews(ctx, "book-1", nil, "", 2)
	if err != nil || len(reviews) != 2 {
		t.Fatalf("reviews page 1: %v, len=%d", err, len(reviews))
	}
	reviews2, err := repo.ListBookReviews(ctx, "book-1", &cursorTime, reviews[1].UserID, 2)
	if err != nil || len(reviews2) != 1 || reviews2[0].UserID == reviews[0].UserID || reviews2[0].UserID == reviews[1].UserID {
		t.Fatalf("reviews page 2: %#v, %v", reviews2, err)
	}
}
