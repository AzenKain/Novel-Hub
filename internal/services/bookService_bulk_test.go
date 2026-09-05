package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func TestBulkUpdateMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "bulk_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	seed := []string{
		`INSERT INTO users (id, email, password_hash) VALUES ('u-admin', 'admin@example.com', 'hash')`,
		`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main Library')`,
		`INSERT INTO authors (id, name) VALUES ('a-old', 'Original Author')`,
		`INSERT INTO books (id, library_id, title, author_id, status) VALUES ('book-1', 'lib-1', '[Old] Book 1', 'a-old', 'ready')`,
		`INSERT INTO books (id, library_id, title, author_id, status) VALUES ('book-2', 'lib-1', '[Old] Book 2', 'a-old', 'ready')`,
		`INSERT INTO tags (id, name) VALUES ('tag-delete', 'To Remove')`,
		`INSERT INTO book_tags (book_id, tag_id) VALUES ('book-1', 'tag-delete')`,
		`INSERT INTO book_tags (book_id, tag_id) VALUES ('book-2', 'tag-delete')`,
	}
	for _, stmt := range seed {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	c := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, c)
	txManager := database.NewTxManager(db)
	settingsService := NewSettingsService(repositories.NewSettingsRepository(db, c), txManager, allowAllPermissions{})
	bookService := NewBookService(bookRepo, nil, nil, nil, bookparser.NewRegistry(), txManager, settingsService, allowAllPermissions{}, nil, nil)

	claims := &response.JWTClaims{
		UId:   "u-admin",
		Roles: []constants.RoleType{constants.RoleTypeAdmin},
	}

	commonAuthor := "Brandon Sanderson"
	commonSeries := "Cosmere"
	commonPublisher := "Tor Books"
	commonLanguage := "English"
	newTitle1 := "Mistborn: The Final Empire"
	newTitle2 := "Mistborn: The Well of Ascension"

	dto := &request.BulkUpdateMetadataDto{
		BookIDs:         []string{"book-1", "book-2"},
		Author:          &commonAuthor,
		Series:          &commonSeries,
		AutoIndexSeries: true,
		Publisher:       &commonPublisher,
		Language:        &commonLanguage,
		AddTags:         []string{"Epic Fantasy", "Magic"},
		RemoveTags:      []string{"To Remove"},
		Items: []request.BulkUpdateMetadataItemDto{
			{
				BookID: "book-1",
				Title:  &newTitle1,
			},
			{
				BookID: "book-2",
				Title:  &newTitle2,
			},
		},
	}

	res, err := bookService.BulkUpdateMetadata(context.Background(), dto, claims)
	if err != nil {
		t.Fatalf("BulkUpdateMetadata failed: %v", err)
	}

	if res.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d (errors: %v)", res.SuccessCount, res.Errors)
	}

	var title1, title2 string
	if err := db.QueryRow(`SELECT title FROM books WHERE id = 'book-1'`).Scan(&title1); err != nil {
		t.Fatal(err)
	}
	if title1 != newTitle1 {
		t.Errorf("book-1 title want %q, got %q", newTitle1, title1)
	}

	if err := db.QueryRow(`SELECT title FROM books WHERE id = 'book-2'`).Scan(&title2); err != nil {
		t.Fatal(err)
	}
	if title2 != newTitle2 {
		t.Errorf("book-2 title want %q, got %q", newTitle2, title2)
	}

	var authorName string
	if err := db.QueryRow(`SELECT a.name FROM books b JOIN authors a ON b.author_id = a.id WHERE b.id = 'book-1'`).Scan(&authorName); err != nil {
		t.Fatal(err)
	}
	if authorName != commonAuthor {
		t.Errorf("author want %q, got %q", commonAuthor, authorName)
	}

	var sIdx1, sIdx2 string
	if err := db.QueryRow(`SELECT bs.series_index FROM book_series bs JOIN series s ON bs.series_id = s.id WHERE bs.book_id = 'book-1'`).Scan(&sIdx1); err != nil {
		t.Fatal(err)
	}
	if sIdx1 != "1" {
		t.Errorf("book-1 series index want '1', got %q", sIdx1)
	}

	if err := db.QueryRow(`SELECT bs.series_index FROM book_series bs JOIN series s ON bs.series_id = s.id WHERE bs.book_id = 'book-2'`).Scan(&sIdx2); err != nil {
		t.Fatal(err)
	}
	if sIdx2 != "2" {
		t.Errorf("book-2 series index want '2', got %q", sIdx2)
	}

	var tagCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM book_tags bt JOIN tags t ON bt.tag_id = t.id WHERE bt.book_id = 'book-1' AND t.name = 'To Remove'`).Scan(&tagCount); err != nil {
		t.Fatal(err)
	}
	if tagCount != 0 {
		t.Errorf("expected 'To Remove' tag to be deleted, count = %d", tagCount)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM book_tags bt JOIN tags t ON bt.tag_id = t.id WHERE bt.book_id = 'book-1' AND t.name = 'Epic Fantasy'`).Scan(&tagCount); err != nil {
		t.Fatal(err)
	}
	if tagCount != 1 {
		t.Errorf("expected 'Epic Fantasy' tag added, count = %d", tagCount)
	}
}

func TestBulkUpdateMetadata_ValidationErrors(t *testing.T) {
	bookService := NewBookService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := bookService.BulkUpdateMetadata(context.Background(), &request.BulkUpdateMetadataDto{
		BookIDs: []string{},
	}, nil)
	if err == nil {
		t.Error("expected error for empty book IDs, got nil")
	}

	_, err = bookService.BulkUpdateMetadata(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error for nil DTO, got nil")
	}
}
