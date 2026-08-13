package services

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/netx"
)

func TestCleanEnrichQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Solo Leveling (Light Novel) [ENG]", "Solo Leveling"},
		{"Sword Art Online - Vol. 01 (Lục Địa Bay)", "Sword Art Online"},
		{"Ta Có Một Thần Điện (Quyển 2 - Tập 3)", "Ta Có Một Thần Điện"},
		{"Overlord vol 14: The Witch of the Ruined Kingdom", "Overlord"},
		{"Thần Cấp Hệ Thống: Tập 10 - Chương 100", "Thần Cấp Hệ Thống"},
	}

	for _, tt := range tests {
		actual := cleanEnrichQuery(tt.input)
		if actual != tt.expected {
			t.Errorf("cleanEnrichQuery(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestAutoEnrichBook_Success(t *testing.T) {
	// Enable private IPs for this test
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	// Set up local mock API server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
case "/anilist":
			w.Write([]byte(`{
				"data": {
					"Page": {
						"media": [{
							"id": 12345,
							"title": {
								"romaji": "Solo Leveling Romaji",
								"english": "Solo Leveling English",
								"native": "나 혼자만 레벨업"
							},
							"description": "<p>This is a great story about hunters.</p>",
							"coverImage": {
								"large": "https://example.com/cover.jpg"
							},
							"countryOfOrigin": "KR",
							"genres": ["Action", "Adventure", "Fantasy"],
							"staff": {
								"edges": [
									{
										"role": "Story",
										"node": {
											"name": {
												"full": "Chugong"
											}
										}
									}
								]
							}
						}]
					}
				}
			}`))
		case "/openlibrary":
			w.WriteHeader(http.StatusNotFound)
		case "/googlebooks":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Redirect service URLs to test server
	oldAniList := aniListURL
	oldOpenLibrary := openLibraryURL
	oldGoogleBooks := googleBooksURL
	defer func() {
		aniListURL = oldAniList
		openLibraryURL = oldOpenLibrary
		googleBooksURL = oldGoogleBooks
	}()
	aniListURL = ts.URL + "/anilist"
	openLibraryURL = ts.URL + "/openlibrary"
	googleBooksURL = ts.URL + "/googlebooks"

	// Setup db and repository
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_enrich.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// Apply schema migrations (must include 99b_metadata_settings.sql)
	schemaFiles := []string{
		"../../db/schema/10_auth.sql",
		"../../db/schema/15_libraries.sql",
		"../../db/schema/20_books.sql",
		"../../db/schema/25_metadata.sql",
		"../../db/schema/30_files_and_jobs.sql",
		"../../db/schema/65_permissions_settings.sql",
		"../../db/schema/85_operations.sql",
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
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permissionCache := NewPermissionCache(roleRepo)
	settingsService := NewSettingsService(settingsRepo, database.NewTxManager(db), permissionCache)

	// Create test book
	testBook := &models.BookEntity{
		ID:        "book-enrich-1",
		LibraryID: "lib-1",
		Title:     "Solo Leveling (Light Novel)",
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1','Test Library')`); err != nil {
		t.Fatalf("failed to setup test library: %v", err)
	}
	if err := bookRepo.CreateBook(context.Background(), testBook); err != nil {
		t.Fatalf("failed to create test book: %v", err)
	}

	bookService := NewBookService(bookRepo, nil, nil, nil, bookparser.NewRegistry(), database.NewTxManager(db), settingsService, permissionCache, nil, nil)

	// Run AutoEnrichBook
	ctx := context.Background()
	err = bookService.AutoEnrichBook(ctx, "book-enrich-1")
	if err != nil {
		t.Fatalf("AutoEnrichBook failed: %v", err)
	}

	// Verify enriched attributes
	enriched, err := bookRepo.GetBook(ctx, "book-enrich-1")
	if err != nil {
		t.Fatalf("failed to retrieve enriched book: %v", err)
	}

	if enriched.Description == nil || *enriched.Description != "This is a great story about hunters." {
		t.Errorf("unexpected description: %v", enriched.Description)
	}
	if enriched.AnilistID == nil || *enriched.AnilistID != "12345" {
		t.Errorf("unexpected anilist ID: %v", enriched.AnilistID)
	}
	if enriched.AuthorID == nil || *enriched.AuthorID == "" {
		t.Errorf("expected author to be resolved, got nil/empty")
	} else {
		author, err := bookRepo.GetAuthorByID(ctx, *enriched.AuthorID)
		if err != nil || author.Name != "Chugong" {
			t.Errorf("unexpected author resolved: name=%s, err=%v", author.Name, err)
		}
	}
}
