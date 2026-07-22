package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestBookService_SearchInBook_EPUB(t *testing.T) {
	epubPath := "/run/media/kain344/Work/Backup/Work_Code/Project/NovelHub/data/books/019f7f16-38a0-724a-9d8c-1ed7cde8e0d4/Tap 09 - After Story - Omiya Yuu.epub"
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("EPUB test file not found, skipping TestBookService_SearchInBook_EPUB")
	}

	tempDir, err := os.MkdirTemp("", "novelhub_search_test_*")
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

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	booksDir := filepath.Join(tempDir, "data", "books")
	fileRepo, err := repositories.NewBookFileRepository(booksDir)
	if err != nil {
		t.Fatalf("failed to init file repo: %v", err)
	}

	registry := bookparser.NewRegistry()
	registry.Register(epub.NewParser(), "epub")

	txManager := database.NewTxManager(db)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	settingsService := NewSettingsService(settingsRepo)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permCache := NewPermissionCache(roleRepo)

	bookService := NewBookService(bookRepo, fileRepo, registry, txManager, settingsService, permCache, nil)

	bookID := "019f7f16-38a0-724a-9d8c-1ed7cde8e0d4"
	bookEntity := &models.BookEntity{
		ID:     bookID,
		Title:  "Tap 09 - After Story - Omiya Yuu",
		Status: "active",
	}
	if err := bookRepo.CreateBook(context.Background(), bookEntity); err != nil {
		t.Fatalf("failed to create book entity: %v", err)
	}

	// Copy EPUB file into books directory
	srcFile, err := os.Open(epubPath)
	if err != nil {
		t.Fatalf("failed to open source epub: %v", err)
	}
	defer srcFile.Close()

	_, err = fileRepo.SaveBook(context.Background(), bookID, "Tap 09 - After Story - Omiya Yuu.epub", srcFile)
	if err != nil {
		t.Fatalf("failed to save book file: %v", err)
	}

	// Parse EPUB spine & chapters
	parser := epub.NewParser()
	chaptersData, err := parser.ParseSpine(epubPath)
	if err != nil {
		t.Fatalf("failed to parse epub spine: %v", err)
	}

	for _, cd := range chaptersData {
		ch := &models.ChapterEntity{
			ID:           fmt.Sprintf("%s-ch-%d", bookID, cd.Index),
			BookID:       bookID,
			Title:        cd.Title,
			ContentPath:  &cd.ContentPath,
			ChapterIndex: int64(cd.Index),
		}
		if err := bookRepo.CreateChapter(context.Background(), ch); err != nil {
			t.Fatalf("failed to create chapter record: %v", err)
		}
	}

	// Run In-Book Search
	results, err := bookService.SearchInBook(context.Background(), bookID, "Story")
	if err != nil {
		t.Fatalf("In-Book Search failed: %v", err)
	}

	t.Logf("Search for 'Story' in EPUB returned %d snippets", len(results))
}
