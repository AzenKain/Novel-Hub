package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// The facet endpoints carried OptionalJwtAccess but no authorization at all, and the queries were
// unscoped. A facet list is an index of the catalog — every author, tag, publisher and language in
// every library, with counts — so an unauthenticated visitor read the shape of a closed library
// even though /books and /libraries both refused it.
//
// Scoped inside the SQL via ReadableLibraryIDs rather than filtered after the fetch: filtering
// afterwards would let LIMIT page over rows the caller cannot see, so a page of 20 could arrive
// with 3 entries and the cursor would still advance past the other 17.
func TestMetadataFacetsAreScopedToReadableLibraries(t *testing.T) {
	seedFacetFixture := func(t *testing.T, extra ...string) *sql.DB {
		t.Helper()
		t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "facet-scope.db"))
		db, err := database.NewSQLiteDB()
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if err := database.ApplySchema(db); err != nil {
			t.Fatalf("apply schema: %v", err)
		}
		queries := append([]string{
			`INSERT INTO libraries (id, name) VALUES ('lib-open','Open'),('lib-closed','Closed')`,
			`INSERT INTO authors (id, name) VALUES ('a1','Open Author'),('a2','Secret Author')`,
			`INSERT INTO books (id, library_id, title, status, author_id) VALUES
				('bk1','lib-open','Public','published','a1'),
				('bk2','lib-closed','Secret','published','a2')`,
			`INSERT INTO tags (id,name) VALUES ('t1','opentag'),('t2','secrettag')`,
			`INSERT INTO book_tags (book_id, tag_id) VALUES ('bk1','t1'),('bk2','t2')`,
			`INSERT INTO publishers (id,name) VALUES ('p1','OpenPub'),('p2','SecretPub')`,
			`INSERT INTO book_publishers (book_id, publisher_id) VALUES ('bk1','p1'),('bk2','p2')`,
			`INSERT INTO languages (id,name) VALUES ('l1','en'),('l2','ja')`,
			`INSERT INTO book_languages (book_id, language_id) VALUES ('bk1','l1'),('bk2','l2')`,
			`INSERT INTO series (id,name) VALUES ('s1','Open Series'),('s2','Secret Series')`,
			`INSERT INTO book_series (book_id, series_id) VALUES ('bk1','s1'),('bk2','s2')`,
			`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time, hash) VALUES
				('f1','bk1','/open.epub','epub',1,CURRENT_TIMESTAMP,'h1'),
				('f2','bk2','/secret.pdf','pdf',1,CURRENT_TIMESTAMP,'h2')`,
		}, extra...)
		for _, query := range queries {
			if _, err := db.Exec(query); err != nil {
				t.Fatalf("seed %q: %v", query, err)
			}
		}
		return db
	}

	names := func(t *testing.T, server *FiberServer, path string) []string {
		t.Helper()
		resp, err := server.App.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s -> %d: %s", path, resp.StatusCode, body)
		}
		var envelope struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatalf("%s body is not the standard envelope: %v (%s)", path, err, body)
		}
		out := make([]string, 0, len(envelope.Data))
		for _, item := range envelope.Data {
			out = append(out, item.Name)
		}
		return out
	}

	facets := map[string]struct{ open, closed string }{
		"/api/v1/metadata/authors":    {"Open Author", "Secret Author"},
		"/api/v1/metadata/tags":       {"opentag", "secrettag"},
		"/api/v1/metadata/publishers": {"OpenPub", "SecretPub"},
		"/api/v1/metadata/languages":  {"en", "ja"},
		"/api/v1/metadata/series":     {"Open Series", "Secret Series"},
		"/api/v1/metadata/formats":    {"EPUB", "PDF"},
	}

	t.Run("guest_access=selected_libraries hides the closed library", func(t *testing.T) {
		db := seedFacetFixture(t,
			`INSERT INTO app_settings (key,value_json) VALUES ('guest_access.mode','"selected_libraries"')
			 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json`,
			`INSERT INTO app_settings (key,value_json) VALUES ('guest_access.library_ids','["lib-open"]')
			 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json`)
		server := NewHTTPServer()
		server.SetupServer(db, cache.NewRamCache())

		for path, want := range facets {
			got := names(t, server, path)
			// Both assertions matter: a guard that returned nothing would satisfy the first.
			if contains(got, want.closed) {
				t.Errorf("%s leaked %q from the closed library: %v", path, want.closed, got)
			}
			if !contains(got, want.open) {
				t.Errorf("%s dropped %q from the open library: %v", path, want.open, got)
			}
		}
	})

	t.Run("login_required returns nothing to a visitor", func(t *testing.T) {
		db := seedFacetFixture(t,
			`INSERT INTO app_settings (key,value_json) VALUES ('auth.login_required','true')
			 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json`)
		server := NewHTTPServer()
		server.SetupServer(db, cache.NewRamCache())

		for path := range facets {
			if got := names(t, server, path); len(got) != 0 {
				t.Errorf("%s served %v while login is required", path, got)
			}
		}
	})

	t.Run("default guest_access still sees everything", func(t *testing.T) {
		db := seedFacetFixture(t)
		server := NewHTTPServer()
		server.SetupServer(db, cache.NewRamCache())

		for path, want := range facets {
			got := names(t, server, path)
			if !contains(got, want.open) || !contains(got, want.closed) {
				t.Errorf("%s should list both libraries when guest access is open: %v", path, got)
			}
		}
	})
}

// A library the caller cannot read must not be named either — /libraries checked library.read but
// not the guest_access setting that GUEST is actually gated by, so a closed library's name leaked.
func TestLibraryListHidesLibrariesClosedToGuests(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "library-scope.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	for _, query := range []string{
		`INSERT INTO libraries (id, name) VALUES ('lib-open','Open'),('lib-closed','Closed')`,
		`INSERT INTO app_settings (key,value_json) VALUES ('guest_access.mode','"selected_libraries"')
		 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json`,
		`INSERT INTO app_settings (key,value_json) VALUES ('guest_access.library_ids','["lib-open"]')
		 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("seed %q: %v", query, err)
		}
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())
	resp, err := server.App.Test(httptest.NewRequest(http.MethodGet, "/api/v1/libraries", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	var envelope struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("body is not the standard envelope: %v (%s)", err, body)
	}
	got := make([]string, 0, len(envelope.Data))
	for _, item := range envelope.Data {
		got = append(got, item.Name)
	}
	if contains(got, "Closed") {
		t.Errorf("the closed library was named to a visitor: %v", got)
	}
	if !contains(got, "Open") {
		t.Errorf("the open library disappeared: %v", got)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
