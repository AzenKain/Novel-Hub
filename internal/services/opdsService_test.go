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
	"novelhub/pkg/opds"
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
		"../../db/schema/91_rbac_restructure.sql",
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
	libraryRepo := repositories.NewLibraryRepository(db, ramCache)
	libraryService := NewLibraryService(libraryRepo, bookRepo, nil, nil, permissionCache, settingsService, nil)
	metadataService := NewMetadataService(bookRepo, libraryService)
	opdsService := NewOPDSService(bookService, metadataService, settingsService, permissionCache)
	adminClaims := &response.JWTClaims{UId: "1", Roles: []constants.RoleType{constants.RoleTypeAdmin}}

	return opdsService, db, adminClaims
}

func TestOPDS12_RootCatalogNavigationLinks(t *testing.T) {
	opdsSvc, _, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"
	const basePath = "/opds"

	feed, err := opdsSvc.GetRootCatalog(ctx, serverURL, basePath, claims)
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

	if len(feed.Entries) < 5 {
		t.Fatalf("expected at least 5 navigation entries in root feed, got %d", len(feed.Entries))
	}

	var titles []string
	for _, entry := range feed.Entries {
		titles = append(titles, entry.Title)
	}
	joined := strings.Join(titles, ", ")
	if !strings.Contains(joined, "All Books") || !strings.Contains(joined, "Authors") || !strings.Contains(joined, "Series") || !strings.Contains(joined, "Tags") {
		t.Errorf("expected All Books, Authors, Series, Tags in root feed entries, got %s", joined)
	}
}

func TestOPDS12_SearchAndOpenSearch(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"
	const basePath = "/opds"

	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('book-search-1', 'lib-1', 'Overlord Volume 01', 'active')`); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	openSearch := opdsSvc.GetOpenSearchDescription(serverURL, basePath)
	if openSearch.ShortName != "NovelHub" {
		t.Errorf("unexpected short name: %s", openSearch.ShortName)
	}

	feed, err := opdsSvc.SearchBooksOPDS(ctx, serverURL, basePath, "Overlord", request.OPDSPageDto{Limit: 10}, claims)
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
	const basePath = "/opds"

	if _, err := db.Exec(`INSERT INTO authors (id, name) VALUES ('Reki Kawahara', 'Reki Kawahara')`); err != nil {
		t.Fatalf("failed to insert author: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, author_id, status) VALUES ('book-auth-1', 'lib-1', 'Sword Art Online 01', 'Reki Kawahara', 'active')`); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	authorsFeed, err := opdsSvc.GetAuthorsCatalog(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetAuthorsCatalog failed: %v", err)
	}
	if len(authorsFeed.Entries) == 0 {
		t.Fatalf("expected author entry, got 0")
	}

	authorBooksFeed, err := opdsSvc.GetAuthorBooks(ctx, serverURL, basePath, "Reki Kawahara", request.OPDSPageDto{Limit: 10}, claims)
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
	const basePath = "/opds"

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

	seriesFeed, err := opdsSvc.GetSeriesCatalog(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetSeriesCatalog failed: %v", err)
	}
	if len(seriesFeed.Entries) == 0 {
		t.Fatalf("expected series entry, got 0")
	}

	seriesBooksFeed, err := opdsSvc.GetSeriesBooks(ctx, serverURL, basePath, "Slime Isekai", request.OPDSPageDto{Limit: 10}, claims)
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
	const basePath = "/opds"

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

	tagsFeed, err := opdsSvc.GetTagsCatalog(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetTagsCatalog failed: %v", err)
	}
	if len(tagsFeed.Entries) == 0 {
		t.Fatalf("expected tag entry, got 0")
	}

	tagBooksFeed, err := opdsSvc.GetTagBooks(ctx, serverURL, basePath, "Fantasy", request.OPDSPageDto{Limit: 10}, claims)
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
	const basePath = "/opds"

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

	feed, err := opdsSvc.GetRecentBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
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
	const basePath = "/opds"

	metaJSON := `{"series":"Test Series","subject":["Fantasy","Adventure"]}`
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, description, metadata_json) VALUES ('test-book-opds-2', 'lib-1', 'Test OPDS 2.0 Book Title', 'active', 'A sample book description', ?)`, metaJSON); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	catalog, err := opdsSvc.GetOPDS2Catalog(ctx, serverURL, basePath, claims)
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
	if !ok || len(navigation) < 5 {
		t.Fatalf("expected at least 5 navigation feeds in OPDS 2.0 catalog, got %d", len(navigation))
	}
	if navigation[0]["title"] != "All Books" || navigation[1]["title"] != "Recent Additions" {
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

func TestOPDS_AllBooksHotRandomFeeds(t *testing.T) {
	opdsSvc, db, claims := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"
	const basePath = "/opds"

	for i := 1; i <= 3; i++ {
		if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, read_count) VALUES (?, 'lib-1', ?, 'active', 10)`, fmt.Sprintf("book-%d", i), fmt.Sprintf("Catalog Book %d", i)); err != nil {
			t.Fatalf("insert book failed: %v", err)
		}
	}

	// OPDS 1.2 All Books
	allFeed, err := opdsSvc.GetAllBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetAllBooks failed: %v", err)
	}
	if len(allFeed.Entries) != 3 {
		t.Errorf("expected 3 entries in all books, got %d", len(allFeed.Entries))
	}

	// OPDS 1.2 Hot Books
	hotFeed, err := opdsSvc.GetHotBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetHotBooks failed: %v", err)
	}
	if len(hotFeed.Entries) != 3 {
		t.Errorf("expected 3 entries in hot books, got %d", len(hotFeed.Entries))
	}

	// OPDS 1.2 Random Books
	randFeed, err := opdsSvc.GetRandomBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetRandomBooks failed: %v", err)
	}
	if len(randFeed.Entries) != 3 {
		t.Errorf("expected 3 entries in random books, got %d", len(randFeed.Entries))
	}

	// OPDS 2.0 All Books
	allV2, err := opdsSvc.GetOPDS2AllBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetOPDS2AllBooks failed: %v", err)
	}
	if pubs, ok := allV2["publications"].([]map[string]any); !ok || len(pubs) != 3 {
		t.Errorf("expected 3 publications in OPDS 2.0 all books, got %v", allV2["publications"])
	}

	// OPDS 2.0 Hot Books
	hotV2, err := opdsSvc.GetOPDS2HotBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetOPDS2HotBooks failed: %v", err)
	}
	if pubs, ok := hotV2["publications"].([]map[string]any); !ok || len(pubs) != 3 {
		t.Errorf("expected 3 publications in OPDS 2.0 hot books, got %v", hotV2["publications"])
	}

	// OPDS 2.0 Random Books
	randV2, err := opdsSvc.GetOPDS2RandomBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, claims)
	if err != nil {
		t.Fatalf("GetOPDS2RandomBooks failed: %v", err)
	}
	if pubs, ok := randV2["publications"].([]map[string]any); !ok || len(pubs) != 3 {
		t.Errorf("expected 3 publications in OPDS 2.0 random books, got %v", randV2["publications"])
	}
}

func TestOPDS_GuestAcquisitionAndExcludesAudiobooks(t *testing.T) {
	opdsSvc, db, _ := setupOPDSFullTestEnv(t)
	ctx := context.Background()
	const serverURL = "http://192.168.1.19:3434"
	const basePath = "/opds"

	// Insert an ebook with an EPUB file
	metaJSON := `{"date":"2023-05-15","series":"My Light Novel"}`
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, metadata_json) VALUES ('book-ebook-1', 'lib-1', 'EBook Title', 'active', ?)`, metaJSON); err != nil {
		t.Fatalf("insert ebook failed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES ('f-epub-1', 'book-ebook-1', '/data/books/novel.epub', 'epub', 1024, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert epub file failed: %v", err)
	}

	// Insert an audiobook with only an MP3 file
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('book-audio-1', 'lib-1', 'Audiobook Title', 'active')`); err != nil {
		t.Fatalf("insert audiobook failed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES ('f-mp3-1', 'book-audio-1', '/data/books/track.mp3', 'mp3', 2048, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert mp3 file failed: %v", err)
	}

	guestClaims := &response.JWTClaims{UId: "0", Roles: []constants.RoleType{constants.RoleTypeGuest}}

	// 1. Check OPDS 1.2 All Books excludes audiobooks and includes acquisition links for Guest
	feed12, err := opdsSvc.GetAllBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, guestClaims)
	if err != nil {
		t.Fatalf("GetAllBooks failed: %v", err)
	}
	foundAudio := false
	var ebookEntry *opds.Entry
	for i := range feed12.Entries {
		if feed12.Entries[i].ID == "novelhub:book:book-audio-1" {
			foundAudio = true
		}
		if feed12.Entries[i].ID == "novelhub:book:book-ebook-1" {
			ebookEntry = &feed12.Entries[i]
		}
	}
	if foundAudio {
		t.Errorf("pure audiobook should be excluded from standard OPDS ebook feeds, but was found")
	}
	if ebookEntry == nil {
		t.Fatalf("ebook entry not found in OPDS 1.2 feed")
	}

	hasAcquisition := false
	for _, link := range ebookEntry.Links {
		if link.Rel == "http://opds-spec.org/acquisition" {
			hasAcquisition = true
			if !strings.HasPrefix(link.Href, serverURL+basePath+"/v1/books/book-ebook-1/download") {
				t.Errorf("unexpected acquisition href: %s", link.Href)
			}
			if link.Type != "application/epub+zip" {
				t.Errorf("expected MIME application/epub+zip, got %s", link.Type)
			}
		}
	}
	if !hasAcquisition {
		t.Errorf("guest user missing acquisition link in OPDS 1.2 feed")
	}

	// 2. Check OPDS 2.0 All Books excludes audiobooks and includes acquisition links and publication date
	feedV2, err := opdsSvc.GetOPDS2AllBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 10}, guestClaims)
	if err != nil {
		t.Fatalf("GetOPDS2AllBooks failed: %v", err)
	}
	pubs, ok := feedV2["publications"].([]map[string]any)
	if !ok {
		t.Fatalf("invalid publications list in OPDS 2.0")
	}
	var ebookPub map[string]any
	for _, p := range pubs {
		meta := p["metadata"].(map[string]any)
		if meta["identifier"] == "urn:novelhub:book:book-audio-1" {
			t.Errorf("pure audiobook should be excluded from OPDS 2.0 feed")
		}
		if meta["identifier"] == "urn:novelhub:book:book-ebook-1" {
			ebookPub = p
		}
	}
	if ebookPub == nil {
		t.Fatalf("ebook publication not found in OPDS 2.0 feed")
	}

	meta := ebookPub["metadata"].(map[string]any)
	if pubDate, ok := meta["published"].(string); !ok || pubDate != "2023-05-15" {
		t.Errorf("expected published date 2023-05-15, got %v", meta["published"])
	}

	links, ok := ebookPub["links"].([]map[string]any)
	if !ok {
		t.Fatalf("invalid links in publication")
	}
	hasV2Acquisition := false
	for _, l := range links {
		if l["rel"] == "http://opds-spec.org/acquisition" {
			hasV2Acquisition = true
			if l["type"] != "application/epub+zip" {
				t.Errorf("expected type application/epub+zip, got %v", l["type"])
			}
		}
	}
	if !hasV2Acquisition {
		t.Errorf("guest user missing acquisition link in OPDS 2.0 publication")
	}

	// 3. Check guest download succeeds via GetBookFileForDownload
	filePath, downloadName, err := opdsSvc.GetBookFileForDownload(ctx, "book-ebook-1", "f-epub-1", guestClaims)
	if err != nil {
		t.Fatalf("guest GetBookFileForDownload failed: %v", err)
	}
	if !strings.HasSuffix(filePath, "novel.epub") {
		t.Errorf("expected path ending in novel.epub, got %s", filePath)
	}
	if downloadName != "EBook Title.epub" {
		t.Errorf("expected 'EBook Title.epub', got %s", downloadName)
	}
}


