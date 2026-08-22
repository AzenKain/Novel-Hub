package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// Komga contract tests.
//
// The Mihon Komga extension is not runnable in CI, so these do not prove the integration works
// on a phone. What they prove is that the JSON on the wire carries every field the two Kotlin
// clients require, since a single missing non-nullable field makes kotlinx.serialization throw
// at decode time and the extension silently shows an empty library.
//
// Expected field names come from the sources, not from memory:
//   - keiyoushi/extensions-source .../komga/dto/{Dto,PageWrapperDto}.kt
//   - mihonapp/mihon .../data/track/komga/KomgaModels.kt
//
// Requests are built the way the extension builds them: it appends /api/v1 to the address the
// user typed, and its OkHttp authenticator only attaches Basic credentials after a 401.

type komgaFixture struct {
	app      *fiber.App
	db       *sql.DB
	cache    cache.Cache
	seriesID string
	bookID   string
	libID    string
	pages    int
}

const komgaTestPassword = "password123"
const komgaTestEmail = "komga-test@example.com"

func setupKomgaFixture(t *testing.T) komgaFixture {
	t.Helper()
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "novelhub-komga-test.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(komgaTestPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, ?, 'Komga User', ?, 'LOCAL', 1)
	`, userID, komgaTestEmail, string(hash)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// USER carries komga.sync by default (pkg/constants/role.go), which the routes gate on.
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'
	`, userID); err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	libID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES (?, 'Manga')`, libID); err != nil {
		t.Fatalf("seed library: %v", err)
	}

	seriesID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO series (id, name) VALUES (?, 'Test Manga')`, seriesID); err != nil {
		t.Fatalf("seed series: %v", err)
	}

	bookID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO books (id, library_id, title, description, status)
		VALUES (?, ?, 'Volume 1', 'First volume.', 'active')
	`, bookID, libID); err != nil {
		t.Fatalf("seed book: %v", err)
	}
	// 2.5 rather than an integer: numberSort must survive as a float, or chapter ordering
	// collapses for half-volumes.
	if _, err := db.Exec(`
		INSERT INTO book_series (book_id, series_id, series_index) VALUES (?, ?, '2.5')
	`, bookID, seriesID); err != nil {
		t.Fatalf("seed book_series: %v", err)
	}

	cbzPath := filepath.Join(t.TempDir(), "volume-1.cbz")
	pages := writeTestCBZ(t, cbzPath, 3)
	if _, err := db.Exec(`
		INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time)
		VALUES (?, ?, ?, 'CBZ', 4096, CURRENT_TIMESTAMP)
	`, uuid.Must(uuid.NewV7()).String(), bookID, cbzPath); err != nil {
		t.Fatalf("seed book file: %v", err)
	}

	ramCache := cache.NewRamCache()
	server := NewHTTPServer()
	server.SetupServer(db, ramCache)

	return komgaFixture{app: server.App, db: db, cache: ramCache, seriesID: seriesID, bookID: bookID, libID: libID, pages: pages}
}

// writeTestCBZ builds a real comic archive: the page endpoints go through the actual zip
// reader, so a stub file would only prove the handler returns something.
func writeTestCBZ(t *testing.T, path string, count int) int {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for i := 1; i <= count; i++ {
		w, err := zw.Create("page" + string(rune('0'+i)) + ".jpg")
		if err != nil {
			t.Fatalf("zip create: %v", err)
		}
		img := image.NewRGBA(image.Rect(0, 0, 4, 4))
		img.Set(0, 0, color.RGBA{R: uint8(i * 40), A: 255})
		if err := jpeg.Encode(w, img, nil); err != nil {
			t.Fatalf("encode jpeg: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write cbz: %v", err)
	}
	return count
}

func (f komgaFixture) request(t *testing.T, method, path string, body []byte, auth bool) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, "/komga"+path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		cred := base64.StdEncoding.EncodeToString([]byte(komgaTestEmail + ":" + komgaTestPassword))
		req.Header.Set("Authorization", "Basic "+cred)
	}
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func (f komgaFixture) get(t *testing.T, path string) *http.Response {
	t.Helper()
	return f.request(t, http.MethodGet, path, nil, true)
}

// The extension's OkHttp authenticator is challenge-driven: it attaches credentials only after
// a 401. A 403 or a redirect here means it never authenticates and the library reads as empty.
func TestKomgaChallengesUnauthenticatedRequestWith401(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.request(t, http.MethodGet, "/api/v1/series", nil, false)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /api/v1/series = %d, want 401", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.HasPrefix(got, "Basic ") {
		t.Errorf("WWW-Authenticate = %q, want a Basic challenge", got)
	}
}

func TestKomgaRejectsWrongPassword(t *testing.T) {
	f := setupKomgaFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/komga/api/v1/series", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(komgaTestEmail+":wrong")))
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong password = %d, want 401", resp.StatusCode)
	}
}

// PageWrapperDto has no defaults in Kotlin, so every one of these keys must be present or the
// series list fails to decode.
func TestKomgaSeriesListCarriesEveryPageWrapperField(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.get(t, "/api/v1/series")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/series = %d, want 200", resp.StatusCode)
	}

	var page map[string]any
	decodeJSON(t, resp, &page)
	for _, key := range []string{
		"content", "empty", "first", "last", "number",
		"numberOfElements", "size", "totalElements", "totalPages",
	} {
		if _, ok := page[key]; !ok {
			t.Errorf("PageWrapperDto.%s missing; kotlinx.serialization throws on absent non-null fields", key)
		}
	}

	content, _ := page["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("content has %d series, want 1", len(content))
	}
	series, _ := content[0].(map[string]any)
	// Union of the extension's SeriesDto and the tracker's: the tracker adds the three counts.
	for _, key := range []string{
		"id", "libraryId", "name", "fileLastModified", "booksCount",
		"booksReadCount", "booksUnreadCount", "booksInProgressCount",
		"metadata", "booksMetadata",
	} {
		if _, ok := series[key]; !ok {
			t.Errorf("SeriesDto.%s missing", key)
		}
	}
	if series["libraryId"] != f.libID {
		t.Errorf("libraryId = %v, want %s", series["libraryId"], f.libID)
	}

	metadata, _ := series["metadata"].(map[string]any)
	for _, key := range []string{
		"status", "title", "titleSort", "summary", "summaryLock",
		"readingDirection", "readingDirectionLock", "publisher", "publisherLock",
		"ageRating", "ageRatingLock", "language", "languageLock",
		"genres", "genresLock", "tags", "tagsLock",
	} {
		if _, ok := metadata[key]; !ok {
			t.Errorf("SeriesMetadataDto.%s missing", key)
		}
	}
	// The extension maps status onto SManga; an unknown value silently becomes UNKNOWN.
	switch metadata["status"] {
	case "ONGOING", "ENDED", "ABANDONED", "HIATUS":
	default:
		t.Errorf("metadata.status = %v, want one of ONGOING/ENDED/ABANDONED/HIATUS", metadata["status"])
	}

	booksMetadata, _ := series["booksMetadata"].(map[string]any)
	for _, key := range []string{"authors", "tags", "summary", "summaryNumber", "created", "lastModified"} {
		if _, ok := booksMetadata[key]; !ok {
			t.Errorf("BookMetadataAggregationDto.%s missing", key)
		}
	}
}

func TestKomgaSeriesBooksCarryMediaAndNumberSort(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.get(t, "/api/v1/series/"+f.seriesID+"/books?unpaged=true&media_status=READY&deleted=false")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET series books = %d, want 200", resp.StatusCode)
	}

	var page map[string]any
	decodeJSON(t, resp, &page)
	content, _ := page["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("content has %d books, want 1", len(content))
	}
	book, _ := content[0].(map[string]any)
	for _, key := range []string{
		"id", "seriesId", "seriesTitle", "name", "number",
		"fileLastModified", "sizeBytes", "size", "media", "metadata",
	} {
		if _, ok := book[key]; !ok {
			t.Errorf("BookDto.%s missing", key)
		}
	}

	media, _ := book["media"].(map[string]any)
	// A profile of EPUB makes the extension skip the book unless epubDivinaCompatible is true.
	if media["mediaProfile"] != "DIVINA" {
		t.Errorf("media.mediaProfile = %v, want DIVINA", media["mediaProfile"])
	}
	if got, want := media["pagesCount"], float64(f.pages); got != want {
		t.Errorf("media.pagesCount = %v, want %v", got, want)
	}

	metadata, _ := book["metadata"].(map[string]any)
	if got := metadata["numberSort"]; got != 2.5 {
		t.Errorf("metadata.numberSort = %v, want 2.5 (fractional volume indexes must survive)", got)
	}
	if got := metadata["number"]; got != "2.5" {
		t.Errorf("metadata.number = %v, want \"2.5\" (it is a string in BookMetadataDto)", got)
	}
}

// Page numbers are 1-based here, unlike the 0-based ?page= on the series list.
func TestKomgaPagesAreOneBasedAndServeImageBytes(t *testing.T) {
	f := setupKomgaFixture(t)

	resp := f.get(t, "/api/v1/books/"+f.bookID+"/pages")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET pages = %d, want 200", resp.StatusCode)
	}
	var pages []map[string]any
	decodeJSON(t, resp, &pages)
	if len(pages) != f.pages {
		t.Fatalf("got %d pages, want %d", len(pages), f.pages)
	}
	if got := pages[0]["number"]; got != float64(1) {
		t.Errorf("first page number = %v, want 1", got)
	}
	for _, key := range []string{"number", "fileName", "mediaType"} {
		if _, ok := pages[0][key]; !ok {
			t.Errorf("PageDto.%s missing", key)
		}
	}

	img := f.get(t, "/api/v1/books/"+f.bookID+"/pages/1")
	if img.StatusCode != http.StatusOK {
		t.Fatalf("GET page 1 = %d, want 200", img.StatusCode)
	}
	if ct := img.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		t.Errorf("page Content-Type = %q, want an image type", ct)
	}

	if out := f.get(t, "/api/v1/books/"+f.bookID+"/pages/0"); out.StatusCode != http.StatusNotFound {
		t.Errorf("page 0 = %d, want 404 (numbering starts at 1)", out.StatusCode)
	}
	if out := f.get(t, "/api/v1/books/"+f.bookID+"/pages/99"); out.StatusCode != http.StatusNotFound {
		t.Errorf("page 99 = %d, want 404", out.StatusCode)
	}
}

// The tracker reads progress from /api/v2 and writes it back with lastBookNumberSortRead,
// then immediately re-reads to confirm. Both halves must line up.
func TestKomgaReadProgressRoundTrips(t *testing.T) {
	f := setupKomgaFixture(t)

	resp := f.get(t, "/api/v2/series/"+f.seriesID+"/read-progress/tachiyomi")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET read-progress = %d, want 200", resp.StatusCode)
	}
	var progress map[string]any
	decodeJSON(t, resp, &progress)
	for _, key := range []string{
		"booksCount", "booksReadCount", "booksUnreadCount",
		"booksInProgressCount", "lastReadContinuousNumberSort", "maxNumberSort",
	} {
		if _, ok := progress[key]; !ok {
			t.Errorf("ReadProgressV2Dto.%s missing", key)
		}
	}
	if got := progress["booksReadCount"]; got != float64(0) {
		t.Fatalf("booksReadCount starts at %v, want 0", got)
	}

	put := f.request(t, http.MethodPut,
		"/api/v2/series/"+f.seriesID+"/read-progress/tachiyomi",
		[]byte(`{"lastBookNumberSortRead": 2.5}`), true)
	if put.StatusCode < 200 || put.StatusCode >= 300 {
		t.Fatalf("PUT read-progress = %d, want 2xx", put.StatusCode)
	}

	after := f.get(t, "/api/v2/series/"+f.seriesID+"/read-progress/tachiyomi")
	var updated map[string]any
	decodeJSON(t, after, &updated)
	if got := updated["booksReadCount"]; got != float64(1) {
		t.Errorf("booksReadCount after PUT = %v, want 1", got)
	}
	if got := updated["booksUnreadCount"]; got != float64(0) {
		t.Errorf("booksUnreadCount after PUT = %v, want 0", got)
	}
}

// A library the user cannot sync must not leak through any endpoint. Revoking the permission
// after startup would be invisible (permissionCache.Reload runs once), so this drops the role
// membership instead, which is read per request.
func TestKomgaHidesLibrariesWithoutPermission(t *testing.T) {
	f := setupKomgaFixture(t)
	if _, err := f.db.Exec(`DELETE FROM user_roles`); err != nil {
		t.Fatalf("revoke roles: %v", err)
	}

	resp := f.get(t, "/api/v1/series")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET series = %d, want 200", resp.StatusCode)
	}
	var page map[string]any
	decodeJSON(t, resp, &page)
	if content, _ := page["content"].([]any); len(content) != 0 {
		t.Errorf("series visible without komga.sync: %v", content)
	}

	if out := f.get(t, "/api/v1/books/"+f.bookID+"/pages"); out.StatusCode == http.StatusOK {
		t.Error("book pages readable without komga.sync")
	}
}

func TestKomgaLibrariesEndpointAnswers(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.get(t, "/api/v1/libraries")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/libraries = %d, want 200", resp.StatusCode)
	}
	var libs []map[string]any
	decodeJSON(t, resp, &libs)
	if len(libs) != 1 || libs[0]["id"] != f.libID {
		t.Errorf("libraries = %v, want the seeded library %s", libs, f.libID)
	}
}

func TestKomgaSingleSeriesAndBookFetch(t *testing.T) {
	f := setupKomgaFixture(t)

	resp := f.get(t, "/api/v1/series/"+f.seriesID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET series = %d, want 200", resp.StatusCode)
	}
	var series map[string]any
	decodeJSON(t, resp, &series)
	if series["id"] != f.seriesID {
		t.Errorf("series id = %v, want %s", series["id"], f.seriesID)
	}

	// The tracker fetches a book directly and needs its series named on the response.
	book := f.get(t, "/api/v1/books/"+f.bookID)
	if book.StatusCode != http.StatusOK {
		t.Fatalf("GET book = %d, want 200", book.StatusCode)
	}
	var dto map[string]any
	decodeJSON(t, book, &dto)
	if dto["seriesId"] != f.seriesID {
		t.Errorf("book.seriesId = %v, want %s", dto["seriesId"], f.seriesID)
	}
	if dto["seriesTitle"] != "Test Manga" {
		t.Errorf("book.seriesTitle = %v, want Test Manga", dto["seriesTitle"])
	}
}

func TestKomgaUnknownIDsReturn404(t *testing.T) {
	f := setupKomgaFixture(t)
	for _, path := range []string{
		"/api/v1/series/does-not-exist",
		"/api/v1/books/does-not-exist",
		"/api/v1/books/does-not-exist/pages",
		"/api/v2/series/does-not-exist/read-progress/tachiyomi",
	} {
		if resp := f.get(t, path); resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, resp.StatusCode)
		}
	}
}

// pageNames caches the archive's page list under the file's mod_time. A replaced file must not
// keep serving the old mapping, or page N points at the wrong image.
func TestKomgaPageNameCacheFollowsModTime(t *testing.T) {
	f := setupKomgaFixture(t)

	first := f.get(t, "/api/v1/books/"+f.bookID+"/pages")
	var before []map[string]any
	decodeJSON(t, first, &before)
	if len(before) != f.pages {
		t.Fatalf("got %d pages, want %d", len(before), f.pages)
	}

	var path string
	if err := f.db.QueryRow(`SELECT path FROM book_files WHERE book_id = ?`, f.bookID).Scan(&path); err != nil {
		t.Fatal(err)
	}
	writeTestCBZ(t, path, f.pages+2)
	// Through the repository, not raw SQL: UpsertBookFile is what a library rescan calls, and it
	// evicts the book_file entity whose mod_time keys the page-name cache.
	repo := repositories.NewBookDBRepository(f.db, f.cache)
	if err := repo.UpsertBookFile(context.Background(), sqlc.UpsertBookFileParams{
		ID:        uuid.Must(uuid.NewV7()).String(),
		BookID:    f.bookID,
		Path:      path,
		Format:    "CBZ",
		SizeBytes: 8192,
		ModTime:   time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	second := f.get(t, "/api/v1/books/"+f.bookID+"/pages")
	var after []map[string]any
	decodeJSON(t, second, &after)
	if len(after) != f.pages+2 {
		t.Errorf("after replacing the file got %d pages, want %d — stale page-name cache", len(after), f.pages+2)
	}
}

func TestKomgaUserMeContract(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.get(t, "/api/v1/users/me")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var me map[string]any
	decodeJSON(t, resp, &me)
	if me["email"] != komgaTestEmail {
		t.Errorf("got email %v, want %s", me["email"], komgaTestEmail)
	}
	roles, ok := me["roles"].([]any)
	if !ok || len(roles) == 0 {
		t.Fatalf("expected non-empty roles, got %v", me["roles"])
	}
}

func TestKomgaReadProgressV1Contract(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.get(t, "/api/v1/series/"+f.seriesID+"/read-progress/tachiyomi")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var progress map[string]any
	decodeJSON(t, resp, &progress)
	if progress["booksCount"] == nil {
		t.Errorf("expected booksCount in v1 progress response")
	}
}

func TestKomgaZeroBasedPagesContract(t *testing.T) {
	f := setupKomgaFixture(t)
	resp := f.get(t, "/api/v1/books/"+f.bookID+"/pages?zero_based=true")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var pages []map[string]any
	decodeJSON(t, resp, &pages)
	if len(pages) == 0 {
		t.Fatalf("expected pages, got 0")
	}
	if num, ok := pages[0]["number"].(float64); !ok || int(num) != 0 {
		t.Errorf("expected first page number to be 0 with zero_based=true, got %v", pages[0]["number"])
	}
}
