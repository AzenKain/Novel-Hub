package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// GET /books/:id/rating and /books/:id/reviews carried no middleware and no handler check while every sibling route carried two guards.
func TestReviewRoutesAreScopedToTheBook(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "review-scope.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	seed := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("seed %q: %v", query, err)
		}
	}
	seed(`INSERT INTO libraries (id, name) VALUES ('lib-open', 'Open'), ('lib-closed', 'Closed')`)
	seed(`INSERT INTO books (id, library_id, title, status) VALUES
		('bk-open', 'lib-open', 'Public', 'published'),
		('bk-closed', 'lib-closed', 'Secret', 'published')`)
	seed(`INSERT INTO app_settings (key, value_json) VALUES ('guest_access.mode', '"selected_libraries"')
	      ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json`)
	seed(`INSERT INTO app_settings (key, value_json) VALUES ('guest_access.library_ids', '["lib-open"]')
	      ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json`)

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/books/bk-closed/rating", http.StatusForbidden},
		{"/api/v1/books/bk-closed/reviews", http.StatusForbidden},
		{"/api/v1/books/bk-open/rating", http.StatusOK},
		{"/api/v1/books/bk-open/reviews", http.StatusOK},
	} {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := server.App.Test(httptest.NewRequest(http.MethodGet, tc.path, nil))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tc.want {
				t.Fatalf("status = %d, want %d: %s", resp.StatusCode, tc.want, body)
			}
			var envelope struct {
				Status bool `json:"status"`
			}
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatalf("body is not the standard envelope: %v (%s)", err, body)
			}
			if envelope.Status != (tc.want == http.StatusOK) {
				t.Errorf("status field = %v for HTTP %d", envelope.Status, resp.StatusCode)
			}
		})
	}
}
