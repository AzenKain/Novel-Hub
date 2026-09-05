package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SearchUser emitted a next_cursor whenever the page was non-empty, with no comparison against the limit, so a short page still advertised another one — the admin list rendered an empty page after a complete one.
func TestUserSearchOmitsCursorOnAShortPage(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	adminID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'admin@example.com', 'Admin', ?, 'LOCAL', 1)`, adminID, string(hash)); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'ADMIN'`, adminID); err != nil {
		t.Fatalf("seed role: %v", err)
	}
	for i := range 4 {
		id := uuid.Must(uuid.NewV7()).String()
		if _, err := db.Exec(`
			INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
			VALUES (?, ?, 'User', ?, 'LOCAL', 1)`,
			id, fmt.Sprintf("user%d@example.com", i), string(hash)); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin",
		bytes.NewReader([]byte(`{"email":"admin@example.com","password":"password123"}`)))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin: %v", err)
	}
	var signin struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(signinResp.Body).Decode(&signin); err != nil {
		t.Fatalf("decode signin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=20", nil)
	req.Header.Set("Authorization", "Bearer "+signin.Data.AccessToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("search users: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Data       []json.RawMessage `json:"data"`
		Pagination struct {
			NextCursor string `json:"next_cursor"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if len(payload.Data) != 5 {
		t.Fatalf("returned %d users, want 5: %s", len(payload.Data), body)
	}
	if payload.Pagination.NextCursor != "" {
		t.Errorf("5 users under a limit of 20 still advertised next_cursor %q, so the UI renders an empty page",
			payload.Pagination.NextCursor)
	}
}

// The collection filter bypasses the keyset SQL and pages in Go instead.
func TestCollectionPagingKeepsBooksSharingATimestamp(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'reader@example.com', 'Reader', ?, 'LOCAL', 1)`, userID, string(hash)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'ADMIN'`, userID); err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatalf("seed library: %v", err)
	}
	collectionID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO collections (id, user_id, name) VALUES (?, ?, 'Batch')`,
		collectionID, userID); err != nil {
		t.Fatalf("seed collection: %v", err)
	}

	const stamp = "2026-08-05 10:00:00"
	bookIDs := make([]string, 5)
	for i := range bookIDs {
		bookIDs[i] = uuid.Must(uuid.NewV7()).String()
		if _, err := db.Exec(`
			INSERT INTO books (id, library_id, title, status, created_at, updated_at)
			VALUES (?, 'lib', ?, 'published', ?, ?)`,
			bookIDs[i], fmt.Sprintf("Batch %d", i), stamp, stamp); err != nil {
			t.Fatalf("seed book: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO collection_books (collection_id, book_id) VALUES (?, ?)`,
			collectionID, bookIDs[i]); err != nil {
			t.Fatalf("seed collection book: %v", err)
		}
	}

	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin",
		bytes.NewReader([]byte(`{"email":"reader@example.com","password":"password123"}`)))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin: %v", err)
	}
	var signin struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(signinResp.Body).Decode(&signin); err != nil {
		t.Fatalf("decode signin: %v", err)
	}

	seen := map[string]bool{}
	cursor := ""
	for page := 0; page < len(bookIDs)+2; page++ {
		path := "/api/v1/books?limit=2&collection=" + collectionID
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+signin.Data.AccessToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d = %d: %s", page, resp.StatusCode, body)
		}
		var payload struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			NextCursor string `json:"next_cursor"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("page %d decode: %v (%s)", page, err, body)
		}
		for _, item := range payload.Data {
			seen[item.ID] = true
		}
		if payload.NextCursor == "" || payload.NextCursor == cursor {
			break
		}
		cursor = payload.NextCursor
	}

	for i, id := range bookIDs {
		if !seen[id] {
			t.Errorf("book %d (%s) was never returned; paging saw %d of %d", i, id, len(seen), len(bookIDs))
		}
	}
}
