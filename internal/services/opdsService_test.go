package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func setupOPDSFullTestEnv(t *testing.T) (OPDSService, *sql.DB, *response.JWTClaims) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_opds_full.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schemaFiles := []string{
		"../../db/schema/10_auth.sql",
		"../../db/schema/15_libraries.sql",
		"../../db/schema/20_books.sql",
		"../../db/schema/25_metadata.sql",
		"../../db/schema/30_files_and_jobs.sql",
		"../../db/schema/45_metadata_fts.sql",
		"../../db/schema/50_user_features.sql",
		"../../db/schema/55_reading_activity.sql",
		"../../db/schema/65_permissions_settings.sql",
		"../../db/schema/90_seed_roles.sql",
		"../../db/schema/95_rbac_restructure.sql",
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

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1','Default Library')`); err != nil {
		t.Fatalf("failed to insert default library: %v", err)
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permissionCache := NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatalf("failed to load permissions: %v", err)
	}
	settingsService := NewSettingsService(settingsRepo, database.NewTxManager(db), permissionCache)
	bookService := NewBookService(bookRepo, nil, nil, nil, bookparser.NewRegistry(), database.NewTxManager(db), settingsService, permissionCache, nil, nil)
	opdsService := NewOPDSService(bookService, permissionCache)
	adminClaims := &response.JWTClaims{UId: "1", Roles: []constants.RoleType{constants.RoleTypeAdmin}}

	return opdsService, db, adminClaims
}

func TestOPDS12_RootCatalogNavigationLinks(t *testing.T) {
	opdsSvc, _, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"

	feed, err := opdsSvc.GetRootCatalog(ctx, serverURL, claims)
	if err != nil {
		t.Fatalf("GetRootCatalog failed: %v", err)
	}

	if feed.Title != "NovelHub OPDS Catalog" {
		t.Errorf("unexpected feed title: %s", feed.Title)
	}

	expectedRels := map[string]bool{
		"self":   false,
		"start":  false,
		"search": false,
	}

	for _, link := range feed.Links {
		if _, exists := expectedRels[link.Rel]; exists {
			expectedRels[link.Rel] = true
		}
	}

	for rel, found := range expectedRels {
		if !found {
			t.Errorf("expected link rel '%s' in root feed, but not found", rel)
		}
	}

	if len(feed.Entries) != 4 {
		t.Fatalf("expected 4 navigation entries in root feed, got %d", len(feed.Entries))
	}

	titles := []string{feed.Entries[0].Title, feed.Entries[1].Title, feed.Entries[2].Title, feed.Entries[3].Title}
	joined := strings.Join(titles, ", ")
	if !strings.Contains(joined, "Authors") || !strings.Contains(joined, "Series") || !strings.Contains(joined, "Tags") {
		t.Errorf("expected Authors, Series, Tags in root feed entries, got %s", joined)
	}
}

func TestOPDS12_SearchAndOpenSearch(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"

	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('book-search-1', 'lib-1', 'Overlord Volume 01', 'active')`); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	openSearch := opdsSvc.GetOpenSearchDescription(serverURL)
	if openSearch.ShortName != "NovelHub" {
		t.Errorf("unexpected short name: %s", openSearch.ShortName)
	}

	feed, err := opdsSvc.SearchBooksOPDS(ctx, serverURL, "Overlord", request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("SearchBooksOPDS failed: %v", err)
	}

	if len(feed.Entries) == 0 {
		t.Fatalf("expected search entry for 'Overlord', got none")
	}

	if feed.Entries[0].Title != "Overlord Volume 01" {
		t.Errorf("expected entry title 'Overlord Volume 01', got '%s'", feed.Entries[0].Title)
	}
}

func TestOPDS12_AuthorsCatalogAndFilter(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"

	if _, err := db.Exec(`INSERT INTO authors (id, name) VALUES ('Reki Kawahara', 'Reki Kawahara')`); err != nil {
		t.Fatalf("failed to insert author: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, author_id, status) VALUES ('book-auth-1', 'lib-1', 'Sword Art Online 01', 'Reki Kawahara', 'active')`); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	authorsFeed, err := opdsSvc.GetAuthorsCatalog(ctx, serverURL, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetAuthorsCatalog failed: %v", err)
	}
	if len(authorsFeed.Entries) == 0 {
		t.Fatalf("expected author entry, got 0")
	}

	authorBooksFeed, err := opdsSvc.GetAuthorBooks(ctx, serverURL, "Reki Kawahara", request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetAuthorBooks failed: %v", err)
	}
	if len(authorBooksFeed.Entries) == 0 {
		t.Fatalf("expected books by Reki Kawahara, got 0")
	}
	if authorBooksFeed.Entries[0].Title != "Sword Art Online 01" {
		t.Errorf("unexpected title: %s", authorBooksFeed.Entries[0].Title)
	}
}

func TestOPDS12_SeriesCatalogAndFilter(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"

	metaJSON := `{"series":"Slime Isekai"}`
	if _, err := db.Exec(`INSERT INTO series (id, name) VALUES ('Slime Isekai', 'Slime Isekai')`); err != nil {
		t.Fatalf("failed to insert series: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, metadata_json) VALUES ('book-series-1', 'lib-1', 'Slime Isekai Vol 1', 'active', ?)`, metaJSON); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_series (book_id, series_id) VALUES ('book-series-1', 'Slime Isekai')`); err != nil {
		t.Fatalf("failed to insert book_series: %v", err)
	}

	seriesFeed, err := opdsSvc.GetSeriesCatalog(ctx, serverURL, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetSeriesCatalog failed: %v", err)
	}
	if len(seriesFeed.Entries) == 0 {
		t.Fatalf("expected series entry, got 0")
	}

	seriesBooksFeed, err := opdsSvc.GetSeriesBooks(ctx, serverURL, "Slime Isekai", request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetSeriesBooks failed: %v", err)
	}
	if len(seriesBooksFeed.Entries) == 0 {
		t.Fatalf("expected books in series, got 0")
	}
	if seriesBooksFeed.Entries[0].Title != "Slime Isekai Vol 1" {
		t.Errorf("unexpected title: %s", seriesBooksFeed.Entries[0].Title)
	}
}

func TestOPDS12_TagsCatalogAndFilter(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"

	metaJSON := `{"subject":"Fantasy, Isekai"}`
	if _, err := db.Exec(`INSERT INTO tags (id, name) VALUES ('Fantasy', 'Fantasy')`); err != nil {
		t.Fatalf("failed to insert tag: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, metadata_json) VALUES ('book-tag-1', 'lib-1', 'Fantasy Adventure Book', 'active', ?)`, metaJSON); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_tags (book_id, tag_id) VALUES ('book-tag-1', 'Fantasy')`); err != nil {
		t.Fatalf("failed to insert book_tags: %v", err)
	}

	tagsFeed, err := opdsSvc.GetTagsCatalog(ctx, serverURL, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetTagsCatalog failed: %v", err)
	}
	if len(tagsFeed.Entries) == 0 {
		t.Fatalf("expected tag entry, got 0")
	}

	tagBooksFeed, err := opdsSvc.GetTagBooks(ctx, serverURL, "Fantasy", request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetTagBooks failed: %v", err)
	}
	if len(tagBooksFeed.Entries) == 0 {
		t.Fatalf("expected books tagged Fantasy, got 0")
	}
}

func TestOPDS12_BookEntryMimeTypesAndAcquisitionLinks(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"

	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('book-files-1', 'lib-1', 'Multi Format Book', 'active')`); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	formats := []struct {
		id     string
		format string
		mime   string
	}{
		{"f-1", "epub", "application/epub+zip"},
		{"f-2", "pdf", "application/pdf"},
		{"f-3", "mobi", "application/x-mobipocket-ebook"},
		{"f-4", "cbz", "application/x-cbz"},
		{"f-5", "mp3", "audio/mpeg"},
	}

	for _, f := range formats {
		filePath := fmt.Sprintf("/tmp/test_%s", f.id)
		if _, err := db.Exec(`INSERT INTO book_files (id, book_id, format, size_bytes, path, mod_time) VALUES (?, 'book-files-1', ?, 1024, ?, CURRENT_TIMESTAMP)`, f.id, f.format, filePath); err != nil {
			t.Fatalf("failed to insert book file: %v", err)
		}
	}

	feed, err := opdsSvc.GetRecentBooks(ctx, serverURL, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetRecentBooks failed: %v", err)
	}
	if len(feed.Entries) == 0 {
		t.Fatalf("expected 1 book entry, got 0")
	}

	entry := feed.Entries[0]
	hasThumbnail := false
	hasFullImage := false
	acquisitionMimes := make(map[string]bool)

	for _, link := range entry.Links {
		if link.Rel == "http://opds-spec.org/image/thumbnail" {
			hasThumbnail = true
		}
		if link.Rel == "http://opds-spec.org/image" {
			hasFullImage = true
		}
		if link.Rel == "http://opds-spec.org/acquisition" {
			acquisitionMimes[link.Type] = true
		}
	}

	if !hasThumbnail || !hasFullImage {
		t.Errorf("missing OPDS cover image links: thumbnail=%v, fullImage=%v", hasThumbnail, hasFullImage)
	}

	for _, f := range formats {
		if !acquisitionMimes[f.mime] {
			t.Errorf("expected acquisition link with MIME type '%s' for format '%s', but not found", f.mime, f.format)
		}
	}
}

func TestOPDSService_GetOPDS2Catalog(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:8080"

	metaJSON := `{"series":"Test Series","subject":["Fantasy","Adventure"]}`
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, description, metadata_json) VALUES ('test-book-opds-2', 'lib-1', 'Test OPDS 2.0 Book Title', 'active', 'A sample book description', ?)`, metaJSON); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	catalog, err := opdsSvc.GetOPDS2Catalog(ctx, serverURL, claims)
	if err != nil {
		t.Fatalf("failed to get OPDS 2.0 catalog: %v", err)
	}

	metadata, ok := catalog["metadata"].(map[string]any)
	if !ok || metadata["title"] != "NovelHub OPDS 2.0 Catalog" {
		t.Errorf("unexpected metadata: %v", catalog["metadata"])
	}
	if numItems, ok := metadata["numberOfItems"].(int); !ok || numItems != 1 {
		t.Errorf("expected numberOfItems 1, got %v", metadata["numberOfItems"])
	}

	navigation, ok := catalog["navigation"].([]map[string]any)
	if !ok || len(navigation) != 4 {
		t.Fatalf("expected 4 navigation feeds in OPDS 2.0 catalog, got %d", len(navigation))
	}
	if navigation[0]["title"] != "Recent Additions" || navigation[1]["title"] != "Authors" {
		t.Errorf("unexpected navigation titles: %v, %v", navigation[0]["title"], navigation[1]["title"])
	}

	publications, ok := catalog["publications"].([]map[string]any)
	if !ok || len(publications) == 0 {
		t.Fatalf("expected publications in OPDS 2.0 catalog, got %v", catalog["publications"])
	}

	pubMeta, ok := publications[0]["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("invalid publication metadata")
	}
	if pubMeta["title"] != "Test OPDS 2.0 Book Title" {
		t.Errorf("expected publication title 'Test OPDS 2.0 Book Title', got %v", pubMeta["title"])
	}
	if pubMeta["summary"] != "A sample book description" {
		t.Errorf("expected summary 'A sample book description', got %v", pubMeta["summary"])
	}
	if belongsTo, ok := pubMeta["belongsTo"].(map[string]any); !ok || belongsTo["series"] != "Test Series" {
		t.Errorf("expected series 'Test Series' in belongsTo, got %v", pubMeta["belongsTo"])
	}
}
