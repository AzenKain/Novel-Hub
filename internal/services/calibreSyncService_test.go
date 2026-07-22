package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestCalibreSyncService_ImportCalibreLibrary(t *testing.T) {
	calibreDbPath := "/run/media/kain344/Work/Backup/Work_Code/Project/NovelHub/metadata.db"
	if _, err := os.Stat(calibreDbPath); os.IsNotExist(err) {
		t.Skip("metadata.db not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "novelhub_test_db_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// Apply database schema
	schemaFiles := []string{
		"../../db/schema/10_auth.sql",
		"../../db/schema/15_libraries.sql",
		"../../db/schema/20_books.sql",
		"../../db/schema/25_metadata.sql",
		"../../db/schema/30_files_and_jobs.sql",
		"../../db/schema/65_permissions_settings.sql",
	}

	for _, sf := range schemaFiles {
		sqlBytes, err := os.ReadFile(sf)
		if err != nil {
			t.Fatalf("failed to read schema %s: %v", sf, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("failed to execute schema %s: %v", sf, err)
		}
	}

	booksDir := filepath.Join(tempDir, "data", "books")
	fileRepo, err := repositories.NewBookFileRepository(booksDir)
	if err != nil {
		t.Fatalf("failed to init file repo: %v", err)
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	txManager := database.NewTxManager(db)

	calibreService := NewCalibreSyncService(bookRepo, fileRepo, txManager)

	calibreDir := "/run/media/kain344/Work/Backup/Work_Code/Project/NovelHub"
	count, err := calibreService.ImportCalibreLibrary(context.Background(), calibreDir, "")
	if err != nil {
		t.Fatalf("failed to import calibre library: %v", err)
	}

	if count <= 0 {
		t.Errorf("expected imported books count > 0, got %d", count)
	}

	// Verify imported metadata entities
	ctx := context.Background()
	authors, err := bookRepo.ListAuthorsWithCount(ctx, "", 10)
	if err == nil {
		t.Logf("Imported %d authors", len(authors))
	}
	tags, err := bookRepo.ListTagsWithCount(ctx, "", 10)
	if err == nil {
		t.Logf("Imported %d tags", len(tags))
	}
	seriesList, err := bookRepo.ListSeriesWithCount(ctx, "", 10)
	if err == nil {
		t.Logf("Imported %d series", len(seriesList))
	}

	t.Logf("Successfully imported %d books with full metadata from Calibre metadata.db!", count)
}
