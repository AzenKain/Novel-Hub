package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/cache"
	"novelhub/pkg/calibre"
	"novelhub/pkg/database"
)

type calibreFixture struct {
	app      *fiber.App
	db       *sql.DB
	cache    cache.Cache
	bookID   string
	libID    string
	authorID string
}

const calibreTestPassword = "secretcalibrepass"
const calibreTestEmail = "calibre-tester@example.com"

func setupCalibreFixture(t *testing.T) calibreFixture {
	t.Helper()

	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("SQLITE_DB_PATH", filepath.Join(dataDir, "novelhub-calibre-test.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value_json) VALUES ('auth.login_required', 'true')
		ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
	`); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(calibreTestPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, ?, 'Calibre User', ?, 'LOCAL', 1)
	`, userID, calibreTestEmail, string(hash)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'ADMIN'
	`, userID); err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	bannedUserID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'banned@example.com', 'Banned User', ?, 'LOCAL', 1)
	`, bannedUserID, string(hash)); err != nil {
		t.Fatalf("seed banned user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'BANNED'
	`, bannedUserID); err != nil {
		t.Fatalf("seed banned role: %v", err)
	}

	libID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES (?, 'Fiction')`, libID); err != nil {
		t.Fatalf("seed library: %v", err)
	}

	authorID := uuid.Must(uuid.NewV7()).String()
	authorName := "Arthur Conan Doyle"
	if _, err := db.Exec(`INSERT INTO authors (id, name) VALUES (?, ?)`, authorID, authorName); err != nil {
		t.Fatalf("seed author: %v", err)
	}

	seriesID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO series (id, name) VALUES (?, 'Sherlock Holmes')`, seriesID); err != nil {
		t.Fatalf("seed series: %v", err)
	}

	bookID := uuid.Must(uuid.NewV7()).String()
	desc := "Classic detective novel"
	metaJSON := `{"creator":"Arthur Conan Doyle","series":"Sherlock Holmes","seriesIndex":1,"subject":["Detective","Mystery"]}`
	coverURL := "/covers/" + bookID + "/cover.jpg"

	if _, err := db.Exec(`
		INSERT INTO books (id, library_id, title, author_id, description, cover_url, metadata_json, status)
		VALUES (?, ?, 'A Study in Scarlet', ?, ?, ?, ?, 'active')
	`, bookID, libID, authorID, desc, coverURL, metaJSON); err != nil {
		t.Fatalf("seed book: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO book_series (book_id, series_id, series_index) VALUES (?, ?, '1.0')
	`, bookID, seriesID); err != nil {
		t.Fatalf("seed book_series: %v", err)
	}

	booksDir := filepath.Join(dataDir, "books")
	bookFolder := filepath.Join(booksDir, bookID)
	if err := os.MkdirAll(bookFolder, 0755); err != nil {
		t.Fatalf("mkdir book folder: %v", err)
	}

	coverPath := filepath.Join(bookFolder, "cover.jpg")
	coverF, err := os.Create(coverPath)
	if err != nil {
		t.Fatalf("create cover file: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.RGBA{R: 200, G: 50, B: 50, A: 255})
	_ = jpeg.Encode(coverF, img, nil)
	_ = coverF.Close()

	epubFullPath := filepath.Join(bookFolder, "book.epub")
	if err := os.WriteFile(epubFullPath, []byte("fake epub data stream"), 0644); err != nil {
		t.Fatalf("write epub: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time)
		VALUES (?, ?, ?, 'EPUB', 1024, CURRENT_TIMESTAMP)
	`, uuid.Must(uuid.NewV7()).String(), bookID, epubFullPath); err != nil {
		t.Fatalf("seed book file: %v", err)
	}

	ramCache := cache.NewRamCache()
	server := NewHTTPServer()
	server.SetupServer(db, ramCache)

	return calibreFixture{
		app:      server.App,
		db:       db,
		cache:    ramCache,
		bookID:   bookID,
		libID:    libID,
		authorID: authorID,
	}
}

func basicAuthHeader(email, password string) string {
	raw := email + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
}

func TestCalibreServer_AuthRequirements(t *testing.T) {
	fix := setupCalibreFixture(t)

	req1 := httptest.NewRequest(http.MethodGet, "/calibre/ajax/library-info", nil)
	resp1, err := fix.app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp1.StatusCode)
	}
	if resp1.Header.Get("WWW-Authenticate") != `Basic realm="NovelHub Calibre Server"` {
		t.Errorf("expected WWW-Authenticate header, got %q", resp1.Header.Get("WWW-Authenticate"))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/calibre/ajax/library-info", nil)
	req2.Header.Set("Authorization", basicAuthHeader(calibreTestEmail, "wrongpassword"))
	resp2, err := fix.app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp2.StatusCode)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/calibre/ajax/library-info", nil)
	req3.Header.Set("Authorization", basicAuthHeader(calibreTestEmail, calibreTestPassword))
	resp3, err := fix.app.Test(req3)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp3.StatusCode)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/calibre/ajax/library-info", nil)
	req4.Header.Set("Authorization", basicAuthHeader("banned@example.com", calibreTestPassword))
	resp4, err := fix.app.Test(req4)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp4.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for banned user, got %d", resp4.StatusCode)
	}
}

func TestCalibreServer_CategoriesAndBooks(t *testing.T) {
	fix := setupCalibreFixture(t)
	auth := basicAuthHeader(calibreTestEmail, calibreTestPassword)

	reqCat := httptest.NewRequest(http.MethodGet, "/calibre/ajax/categories", nil)
	reqCat.Header.Set("Authorization", auth)
	respCat, err := fix.app.Test(reqCat)
	if err != nil || respCat.StatusCode != http.StatusOK {
		t.Fatalf("categories failed: status %d, err %v", respCat.StatusCode, err)
	}
	var catMap map[string]response.CalibreCategorySummary
	if err := json.NewDecoder(respCat.Body).Decode(&catMap); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	if _, ok := catMap["authors"]; !ok {
		t.Fatalf("expected 'authors' in category map, got %v", catMap)
	}

	encAuthors := calibre.EncodeName("authors")
	reqAuth := httptest.NewRequest(http.MethodGet, "/calibre/ajax/category/"+encAuthors, nil)
	reqAuth.Header.Set("Authorization", auth)
	respAuth, err := fix.app.Test(reqAuth)
	if err != nil || respAuth.StatusCode != http.StatusOK {
		t.Fatalf("category authors failed: status %d, err %v", respAuth.StatusCode, err)
	}
	var catDetail response.CalibreCategoryDetailResponse
	if err := json.NewDecoder(respAuth.Body).Decode(&catDetail); err != nil {
		t.Fatalf("decode category detail: %v", err)
	}
	if len(catDetail.Items) == 0 {
		t.Fatalf("expected at least 1 author item, got 0")
	}

	encAuthorItem := calibre.EncodeName("Arthur Conan Doyle")
	reqBooksIn := httptest.NewRequest(http.MethodGet, "/calibre/ajax/books_in/"+encAuthors+"/"+encAuthorItem, nil)
	reqBooksIn.Header.Set("Authorization", auth)
	respBooksIn, err := fix.app.Test(reqBooksIn)
	if err != nil || respBooksIn.StatusCode != http.StatusOK {
		t.Fatalf("books_in failed: status %d, err %v", respBooksIn.StatusCode, err)
	}
	var booksIn response.CalibreBooksInResponse
	if err := json.NewDecoder(respBooksIn.Body).Decode(&booksIn); err != nil {
		t.Fatalf("decode books_in: %v", err)
	}
	if len(booksIn.BookIDs) == 0 || booksIn.BookIDs[0] != fix.bookID {
		t.Fatalf("expected book ID %q, got %v", fix.bookID, booksIn.BookIDs)
	}

	reqBooks := httptest.NewRequest(http.MethodGet, "/calibre/ajax/books?ids="+fix.bookID, nil)
	reqBooks.Header.Set("Authorization", auth)
	respBooks, err := fix.app.Test(reqBooks)
	if err != nil || respBooks.StatusCode != http.StatusOK {
		t.Fatalf("books metadata failed: status %d, err %v", respBooks.StatusCode, err)
	}
	var booksMeta map[string]response.CalibreBookMetadataResponse
	if err := json.NewDecoder(respBooks.Body).Decode(&booksMeta); err != nil {
		t.Fatalf("decode books metadata: %v", err)
	}
	book, ok := booksMeta[fix.bookID]
	if !ok || book.Title != "A Study in Scarlet" {
		t.Fatalf("expected title 'A Study in Scarlet', got %v", book)
	}

	reqSingle := httptest.NewRequest(http.MethodGet, "/calibre/ajax/book/"+fix.bookID, nil)
	reqSingle.Header.Set("Authorization", auth)
	respSingle, err := fix.app.Test(reqSingle)
	if err != nil || respSingle.StatusCode != http.StatusOK {
		t.Fatalf("single book failed: status %d, err %v", respSingle.StatusCode, err)
	}

	reqSearch := httptest.NewRequest(http.MethodGet, "/calibre/ajax/search?query=Scarlet", nil)
	reqSearch.Header.Set("Authorization", auth)
	respSearch, err := fix.app.Test(reqSearch)
	if err != nil || respSearch.StatusCode != http.StatusOK {
		t.Fatalf("search failed: status %d, err %v", respSearch.StatusCode, err)
	}
	var searchRes response.CalibreSearchResponse
	if err := json.NewDecoder(respSearch.Body).Decode(&searchRes); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchRes.BookIDs) == 0 || searchRes.BookIDs[0] != fix.bookID {
		t.Fatalf("expected search match %q, got %v", fix.bookID, searchRes.BookIDs)
	}
}

func TestCalibreServer_GetContentFiles(t *testing.T) {
	fix := setupCalibreFixture(t)
	auth := basicAuthHeader(calibreTestEmail, calibreTestPassword)

	reqCover := httptest.NewRequest(http.MethodGet, "/calibre/get/cover/"+fix.bookID, nil)
	reqCover.Header.Set("Authorization", auth)
	respCover, err := fix.app.Test(reqCover)
	if err != nil || respCover.StatusCode != http.StatusOK {
		t.Fatalf("cover failed: status %d, err %v", respCover.StatusCode, err)
	}

	reqEpub := httptest.NewRequest(http.MethodGet, "/calibre/get/epub/"+fix.bookID, nil)
	reqEpub.Header.Set("Authorization", auth)
	respEpub, err := fix.app.Test(reqEpub)
	if err != nil || respEpub.StatusCode != http.StatusOK {
		t.Fatalf("epub download failed: status %d, err %v", respEpub.StatusCode, err)
	}
	if ct := respEpub.Header.Get("Content-Type"); ct != "application/epub+zip" {
		t.Errorf("expected application/epub+zip, got %q", ct)
	}

	reqMobi := httptest.NewRequest(http.MethodGet, "/calibre/get/mobi/"+fix.bookID, nil)
	reqMobi.Header.Set("Authorization", auth)
	respMobi, err := fix.app.Test(reqMobi)
	if err != nil || respMobi.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing mobi, got %d", respMobi.StatusCode)
	}
}
