package services

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func setupKoboTestEnv(t *testing.T) (KoboService, *sql.DB, cache.Cache, *response.JWTClaims, string) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_kobo.db")
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
		"../../db/schema/57_reading_sessions.sql",
		"../../db/schema/65_permissions_settings.sql",
		"../../db/schema/90_seed_roles.sql",
		"../../db/schema/95_rbac_restructure.sql",
		"../../db/schema/98_kobo_auth.sql",
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

	if _, err := db.Exec(`INSERT INTO users (id, email, password_hash) VALUES ('user-1', 'test@example.com', 'hash')`); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1','Default Library')`); err != nil {
		t.Fatalf("failed to insert library: %v", err)
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	diskRepo, err := repositories.NewBookFileRepository(tempDir)
	if err != nil {
		t.Fatalf("failed to create diskRepo: %v", err)
	}
	koboRepo := repositories.NewKoboRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	featureRepo := repositories.NewFeatureRepository(db, ramCache)

	permissionCache := NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatalf("failed to load permissions: %v", err)
	}

	settingsService := NewSettingsService(settingsRepo, database.NewTxManager(db), permissionCache)
	bookService := NewBookService(bookRepo, featureRepo, nil, diskRepo, bookparser.NewRegistry(), database.NewTxManager(db), settingsService, permissionCache, nil)
	featureService := NewFeatureService(featureRepo, bookRepo, settingsService, permissionCache, database.NewTxManager(db))
	koboService := NewKoboService(bookRepo, diskRepo, koboRepo, bookService, featureService, permissionCache, ramCache)

	adminClaims := &response.JWTClaims{UId: "user-1", RoleIDs: []string{"1"}, Roles: []constants.RoleType{constants.RoleTypeAdmin}}

	return koboService, db, ramCache, adminClaims, tempDir
}

func createSampleEPUBFile(t *testing.T, filePath string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, err := zw.Create("chapter1.xhtml")
	if err != nil {
		t.Fatalf("failed to create chapter entry: %v", err)
	}
	if _, err := w.Write([]byte(`<html><body><p>Hello Kobo reader from NovelHub!</p></body></html>`)); err != nil {
		t.Fatalf("failed to write chapter content: %v", err)
	}

	wImage, err := zw.Create("cover.png")
	if err != nil {
		t.Fatalf("failed to create image entry: %v", err)
	}
	if _, err := wImage.Write([]byte{0x89, 'P', 'N', 'G'}); err != nil {
		t.Fatalf("failed to write image content: %v", err)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test EPUB file: %v", err)
	}
}

func TestKoboService_GetBookKePubStream_CacheAndSingleflight(t *testing.T) {
	koboSvc, db, ramCache, claims, tempDir := setupKoboTestEnv(t)
	ctx := context.Background()

	bookID := "kobo-book-1"
	fileID := "file-epub-1"
	epubPath := filepath.Join(tempDir, "sample.epub")
	createSampleEPUBFile(t, epubPath)

	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES (?, 'lib-1', 'Test Kobo EPUB', 'active')`, bookID); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, format, size_bytes, path, mod_time) VALUES (?, ?, 'epub', 1024, ?, CURRENT_TIMESTAMP)`, fileID, bookID, epubPath); err != nil {
		t.Fatalf("failed to insert book file: %v", err)
	}

	cacheKey := "kepub:converted:" + bookID + ":" + fileID
	exists, _ := ramCache.Exists(ctx, cacheKey)
	if exists {
		t.Fatalf("expected cache miss before first stream request")
	}

	var out1 bytes.Buffer
	if err := koboSvc.GetBookKePubStream(ctx, bookID, claims, &out1); err != nil {
		t.Fatalf("GetBookKePubStream 1st call failed: %v", err)
	}

	zr1, err := zip.NewReader(bytes.NewReader(out1.Bytes()), int64(out1.Len()))
	if err != nil {
		t.Fatalf("failed to parse output 1 as zip: %v", err)
	}

	var chapterHTML string
	for _, f := range zr1.File {
		if f.Name == "chapter1.xhtml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open chapter file in zip: %v", err)
			}
			content, _ := io.ReadAll(rc)
			_ = rc.Close()
			chapterHTML = string(content)
			break
		}
	}

	if !strings.Contains(chapterHTML, `class="koboSpan"`) {
		t.Fatalf("expected converted KePub chapter HTML to contain koboSpan tags, got: %s", chapterHTML)
	}

	exists, _ = ramCache.Exists(ctx, cacheKey)
	if !exists {
		t.Fatalf("expected KePub bytes to be cached after 1st stream request")
	}

	var out2 bytes.Buffer
	if err := koboSvc.GetBookKePubStream(ctx, bookID, claims, &out2); err != nil {
		t.Fatalf("GetBookKePubStream 2nd call failed: %v", err)
	}

	if !bytes.Equal(out1.Bytes(), out2.Bytes()) {
		t.Fatalf("expected 2nd call (cached) to return identical bytes to 1st call")
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var concurrentOut bytes.Buffer
			_ = koboSvc.GetBookKePubStream(ctx, bookID, claims, &concurrentOut)
		}()
	}
	wg.Wait()
}

func TestKoboService_PutAndGetReadingState(t *testing.T) {
	koboSvc, db, _, claims, _ := setupKoboTestEnv(t)
	ctx := context.Background()

	bookID := "kobo-book-2"
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES (?, 'lib-1', 'Kobo Progress Test', 'active')`, bookID); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	progressPct := 65.5
	locVal := "koboSpan-42"
	dto := request.PutKoboStateDto{
		ReadingStates: []request.KoboReadingStateDto{
			{
				CurrentBookmark: &request.KoboBookmarkDto{
					ProgressPercent: &progressPct,
					Location: &request.KoboLocationDto{
						Value: locVal,
						Type:  "KoboSpan",
					},
				},
			},
		},
	}

	res, err := koboSvc.PutReadingState(ctx, claims.UId, bookID, dto, claims)
	if err != nil {
		t.Fatalf("PutReadingState failed: %v", err)
	}

	if res.RequestResult != "Success" && len(res.UpdateResults) == 0 {
		t.Fatalf("unexpected PutReadingState response: %v", res)
	}

	states, err := koboSvc.GetReadingState(ctx, claims.UId, bookID, claims)
	if err != nil {
		t.Fatalf("GetReadingState failed: %v", err)
	}
	if len(states) == 0 {
		t.Fatalf("expected 1 reading state, got 0")
	}

	state := states[0]
	if state.CurrentBookmark.ProgressPercent == nil || *state.CurrentBookmark.ProgressPercent != progressPct {
		t.Errorf("expected ProgressPercent %f, got %v", progressPct, state.CurrentBookmark.ProgressPercent)
	}
	if state.CurrentBookmark.Location == nil || state.CurrentBookmark.Location.Value != locVal {
		t.Errorf("expected Location Value '%s', got %v", locVal, state.CurrentBookmark.Location)
	}
}

func TestKoboService_GetSyncList(t *testing.T) {
	koboSvc, db, _, claims, tempDir := setupKoboTestEnv(t)
	ctx := context.Background()

	epubPath1 := filepath.Join(tempDir, "sync_sample1.epub")
	epubPath2 := filepath.Join(tempDir, "sync_sample2.epub")
	createSampleEPUBFile(t, epubPath1)
	createSampleEPUBFile(t, epubPath2)

	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('kobo-sync-1', 'lib-1', 'Sync Book 1', 'active')`); err != nil {
		t.Fatalf("failed to insert book 1: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, format, size_bytes, path, mod_time) VALUES ('f1', 'kobo-sync-1', 'epub', 1024, ?, CURRENT_TIMESTAMP)`, epubPath1); err != nil {
		t.Fatalf("failed to insert file 1: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('kobo-sync-2', 'lib-1', 'Sync Book 2', 'active')`); err != nil {
		t.Fatalf("failed to insert book 2: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, format, size_bytes, path, mod_time) VALUES ('f2', 'kobo-sync-2', 'epub', 1024, ?, CURRENT_TIMESTAMP)`, epubPath2); err != nil {
		t.Fatalf("failed to insert file 2: %v", err)
	}

	dto := request.KoboSyncDto{UserID: claims.UId, SyncToken: "", EndpointURL: "http://localhost:3434/kobo"}
	resp, err := koboSvc.GetSyncList(ctx, dto, claims)
	if err != nil {
		t.Fatalf("GetSyncList failed: %v", err)
	}

	if len(resp.Items) == 0 {
		t.Fatalf("expected sync items, got 0")
	}
	if resp.SyncToken == "" {
		t.Errorf("expected non-empty sync token")
	}
}

func TestKoboService_GetBookMetadata(t *testing.T) {
	koboSvc, db, _, claims, _ := setupKoboTestEnv(t)
	ctx := context.Background()
	const endpointURL = "http://localhost:3434/kobo"

	bookID := "kobo-meta-1"
	metaJSON := `{"creator":"Kugane Maruyama"}`
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, metadata_json) VALUES (?, 'lib-1', 'Overlord Vol 1', 'active', ?)`, bookID, metaJSON); err != nil {
		t.Fatalf("failed to insert book: %v", err)
	}

	metaList, err := koboSvc.GetBookMetadata(ctx, bookID, endpointURL, claims)
	if err != nil {
		t.Fatalf("GetBookMetadata failed: %v", err)
	}

	if len(metaList) == 0 {
		t.Fatalf("expected metadata for book, got 0")
	}

	meta := metaList[0]
	if meta.Title != "Overlord Vol 1" {
		t.Errorf("unexpected metadata title: %s", meta.Title)
	}
}
