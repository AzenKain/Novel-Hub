package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestSearchBooksUserReadingFilters(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "search_reading.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, email) VALUES ('u1', 'u1@test.com'), ('u2', 'u2@test.com')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO books (id, library_id, title) VALUES ('b1', 'lib-1', 'Book 1'), ('b2', 'lib-1', 'Book 2'), ('b3', 'lib-1', 'Book 3')`); err != nil {
		t.Fatal(err)
	}

	// u1 has reading progress for b1 (50%) and b2 (100%).
	// u2 has reading progress for b3 (100%).
	if _, err := db.ExecContext(ctx, `
		INSERT INTO reading_progress (user_id, book_id, chapter_ref, progress_percent) VALUES
		('u1', 'b1', 'c1', 50.0),
		('u1', 'b2', 'c2', 100.0),
		('u2', 'b3', 'c1', 100.0)
	`); err != nil {
		t.Fatal(err)
	}

	repo := NewBookDBRepository(db, cache.NewRamCache())

	// Test 1: u1 queries "read" -> should return b1 & b2 (progress > 0), NOT b3
	readBooksU1, err := repo.SearchBooks(ctx, nil, nil, "read", "", "", "", "", "", "", 20, "u1")
	if err != nil {
		t.Fatalf("SearchBooks read u1 failed: %v", err)
	}
	if len(readBooksU1) != 2 {
		t.Fatalf("Expected 2 read books for u1, got %d", len(readBooksU1))
	}

	// Test 2: u2 queries "read" -> should return ONLY b3
	readBooksU2, err := repo.SearchBooks(ctx, nil, nil, "read", "", "", "", "", "", "", 20, "u2")
	if err != nil {
		t.Fatalf("SearchBooks read u2 failed: %v", err)
	}
	if len(readBooksU2) != 1 || readBooksU2[0].ID != "b3" {
		t.Fatalf("Expected 1 read book (b3) for u2, got %v", readBooksU2)
	}

	// Test 3: Guest (userID = "") queries "read" -> should return 0 books
	readBooksGuest, err := repo.SearchBooks(ctx, nil, nil, "read", "", "", "", "", "", "", 20, "")
	if err != nil {
		t.Fatalf("SearchBooks read guest failed: %v", err)
	}
	if len(readBooksGuest) != 0 {
		t.Fatalf("Expected 0 read books for guest, got %d", len(readBooksGuest))
	}

	// Test 4: u1 queries "unread" -> should return b3 only
	unreadBooksU1, err := repo.SearchBooks(ctx, nil, nil, "unread", "", "", "", "", "", "", 20, "u1")
	if err != nil {
		t.Fatalf("SearchBooks unread u1 failed: %v", err)
	}
	if len(unreadBooksU1) != 1 || unreadBooksU1[0].ID != "b3" {
		t.Fatalf("Expected 1 unread book (b3) for u1, got %v", unreadBooksU1)
	}

	// Test 5: u1 queries "reading" -> should return b1 only (50%)
	readingBooksU1, err := repo.SearchBooks(ctx, nil, nil, "reading", "", "", "", "", "", "", 20, "u1")
	if err != nil {
		t.Fatalf("SearchBooks reading u1 failed: %v", err)
	}
	if len(readingBooksU1) != 1 || readingBooksU1[0].ID != "b1" {
		t.Fatalf("Expected 1 reading book (b1) for u1, got %v", readingBooksU1)
	}
}

func TestRatingsFacetAndSearch(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "search_ratings.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO books (id, library_id, title, average_rating, rating_count) VALUES
		('b1', 'lib-1', '5 Star Book 1', 5.0, 10),
		('b2', 'lib-1', '5 Star Book 2', 4.8, 5),
		('b3', 'lib-1', '4 Star Book', 4.0, 3),
		('b4', 'lib-1', 'Unrated Book', 0.0, 0)
	`); err != nil {
		t.Fatal(err)
	}

	repo := NewBookDBRepository(db, cache.NewRamCache())

	// Test 1: ListRatingsWithCount -> should return 5 Stars (2 books) and 4 Stars (1 book)
	ratings, err := repo.ListRatingsWithCount(ctx, MetadataFacetFilter{
		LibraryIDs: []string{"lib-1"},
		Limit:      20,
	})
	if err != nil {
		t.Fatalf("ListRatingsWithCount failed: %v", err)
	}
	if len(ratings) != 2 {
		t.Fatalf("Expected 2 rating facets (5 and 4 stars), got %d: %v", len(ratings), ratings)
	}
	if ratings[0].ID != "5" || ratings[0].BookCount != 2 {
		t.Errorf("Expected 5 Stars with 2 books, got %v", ratings[0])
	}
	if ratings[1].ID != "4" || ratings[1].BookCount != 1 {
		t.Errorf("Expected 4 Stars with 1 book, got %v", ratings[1])
	}

	// Test 2: SearchBooks with facet="rating" and facetID="5" -> should return b1 and b2
	star5Books, err := repo.SearchBooks(ctx, nil, nil, "ratings", "", "", "rating", "5", "", "", 20, "")
	if err != nil {
		t.Fatalf("SearchBooks 5 stars failed: %v", err)
	}
	if len(star5Books) != 2 {
		t.Fatalf("Expected 2 books with 5 stars, got %d: %v", len(star5Books), star5Books)
	}

	// Test 3: SearchBooks with facet="rating" and facetID="4" -> should return b3
	star4Books, err := repo.SearchBooks(ctx, nil, nil, "ratings", "", "", "rating", "4", "", "", 20, "")
	if err != nil {
		t.Fatalf("SearchBooks 4 stars failed: %v", err)
	}
	if len(star4Books) != 1 || star4Books[0].ID != "b3" {
		t.Fatalf("Expected 1 book (b3) with 4 stars, got %v", star4Books)
	}
}
