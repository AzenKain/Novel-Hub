package services

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestEnrichBooksBatch(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_enrich.db")
	t.Setenv("SQLITE_DB_PATH", tempDB)

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ramCache := cache.NewRamCache()
	repo := repositories.NewBookDBRepository(db, ramCache)
	fileRepo, err := repositories.NewBookFileRepository(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create file repo: %v", err)
	}
	svc := NewBookService(repo, fileRepo, nil, db)

	ctx := context.Background()

	_, err = db.ExecContext(ctx, "INSERT INTO libraries (id, name) VALUES ('lib-test-999', 'Main Library') ON CONFLICT DO NOTHING")
	if err != nil {
		t.Fatalf("failed to insert library: %v", err)
	}

	// Create test author
	authorID := "author-test-100"
	if err := repo.CreateAuthor(ctx, &models.AuthorEntity{ID: authorID, Name: "Test Author"}); err != nil {
		t.Fatalf("failed to create author: %v", err)
	}

	// Create 10 books
	books := make([]*models.BookEntity, 10)
	for i := 0; i < 10; i++ {
		b := &models.BookEntity{
			ID:        fmt.Sprintf("book-%d", i),
			LibraryID: "lib-test-999",
			Title:     fmt.Sprintf("Book %d", i),
			AuthorID:  &authorID,
			Status:    "active",
		}
		if err := repo.CreateBook(ctx, b); err != nil {
			t.Fatalf("failed to create book: %v", err)
		}
		_ = repo.CreateBookFile(ctx, repositories.BookFileRecordParams{
			ID:        fmt.Sprintf("file-%d", i),
			BookID:    b.ID,
			Path:      fmt.Sprintf("/tmp/book_%d.epub", i),
			Format:    "epub",
			SizeBytes: 1024,
			ModTime:   time.Now(),
		})
		books[i] = b
	}

	// Test enrichBooks batch loading
	svc.(*bookService).enrichBooks(ctx, books)

	for i, b := range books {
		if b.AuthorName == nil || *b.AuthorName != "Test Author" {
			t.Errorf("book %d author name mismatch, expected 'Test Author', got %v", i, b.AuthorName)
		}
		if len(b.Files) != 1 {
			t.Errorf("book %d files count mismatch, expected 1, got %d", i, len(b.Files))
		}
	}
}
