package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

type seriesContextBody struct {
	Status bool `json:"status"`
	Data   struct {
		Series []struct {
			SeriesID    string `json:"series_id"`
			SeriesName  string `json:"series_name"`
			SeriesIndex string `json:"series_index"`
		} `json:"series"`
		Next *struct {
			BookID     string `json:"book_id"`
			Title      string `json:"title"`
			SeriesName string `json:"series_name"`
		} `json:"next"`
	} `json:"data"`
}

func seedSeries(t *testing.T, db *sql.DB, libraryID, seriesID, seriesName string, books [][3]string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO series (id, name) VALUES (?, ?)`, seriesID, seriesName); err != nil {
		t.Fatalf("seed series: %v", err)
	}
	for _, book := range books {
		if _, err := db.Exec(`
			INSERT INTO books (id, library_id, title, status) VALUES (?, ?, ?, 'active')
		`, book[0], libraryID, book[1]); err != nil {
			t.Fatalf("seed book %s: %v", book[0], err)
		}
		if _, err := db.Exec(`
			INSERT INTO book_series (book_id, series_id, series_index) VALUES (?, ?, ?)
		`, book[0], seriesID, book[2]); err != nil {
			t.Fatalf("link book %s: %v", book[0], err)
		}
	}
}

func seriesContext(t *testing.T, app *fiber.App, bookID string) seriesContextBody {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/books/"+bookID+"/series", nil))
	if err != nil {
		t.Fatalf("series request: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("series returned %d: %s", resp.StatusCode, raw)
	}
	var decoded seriesContextBody
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode: %v (%s)", err, raw)
	}
	return decoded
}

// The chip needs the series id: the catalog filter matches on facet_id and silently returns
// every book when only a name is supplied, so a name-only response is the bug, not the fix.
func TestBookSeriesReturnsTheSeriesID(t *testing.T) {
	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatal(err)
	}
	seedSeries(t, db, "lib", "s1", "Dune", [][3]string{{"b1", "Dune", "1"}, {"b2", "Dune Messiah", "2"}})

	body := seriesContext(t, app, "b1")
	if len(body.Data.Series) != 1 {
		t.Fatalf("expected 1 series entry, got %d", len(body.Data.Series))
	}
	if body.Data.Series[0].SeriesID != "s1" {
		t.Errorf("series_id = %q, want s1 — the chip cannot filter without it", body.Data.Series[0].SeriesID)
	}
	if body.Data.Series[0].SeriesName != "Dune" {
		t.Errorf("series_name = %q", body.Data.Series[0].SeriesName)
	}
	if body.Data.Series[0].SeriesIndex != "1" {
		t.Errorf("series_index = %q, want 1", body.Data.Series[0].SeriesIndex)
	}
}

func TestNextInSeriesFollowsTheIndexAndStopsAtTheEnd(t *testing.T) {
	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatal(err)
	}
	seedSeries(t, db, "lib", "s1", "Dune", [][3]string{
		{"b1", "Dune", "1"},
		{"b2", "Dune Messiah", "2"},
		{"b3", "Children of Dune", "3"},
	})

	first := seriesContext(t, app, "b1")
	if first.Data.Next == nil {
		t.Fatal("book 1 has no next book")
	}
	if first.Data.Next.BookID != "b2" {
		t.Errorf("next after #1 = %s, want b2", first.Data.Next.BookID)
	}
	if first.Data.Next.SeriesName != "Dune" {
		t.Errorf("next carries no series name: %q", first.Data.Next.SeriesName)
	}

	middle := seriesContext(t, app, "b2")
	if middle.Data.Next == nil || middle.Data.Next.BookID != "b3" {
		t.Errorf("next after #2 is not b3: %+v", middle.Data.Next)
	}

	last := seriesContext(t, app, "b3")
	if last.Data.Next != nil {
		t.Errorf("the last book claims a next book: %+v", last.Data.Next)
	}
}

// An archived book is hidden from the catalog, so offering it as "next" sends the reader to a
// page they cannot open.
func TestNextInSeriesSkipsArchivedBooks(t *testing.T) {
	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatal(err)
	}
	seedSeries(t, db, "lib", "s1", "Dune", [][3]string{
		{"b1", "Dune", "1"},
		{"b2", "Dune Messiah", "2"},
		{"b3", "Children of Dune", "3"},
	})
	if _, err := db.Exec(`UPDATE books SET status = 'archived' WHERE id = 'b2'`); err != nil {
		t.Fatal(err)
	}

	body := seriesContext(t, app, "b1")
	if body.Data.Next == nil {
		t.Fatal("archiving the middle book removed the next book entirely")
	}
	if body.Data.Next.BookID != "b3" {
		t.Errorf("next = %s, want b3 — an archived book was offered", body.Data.Next.BookID)
	}
}

// A guest restricted to one library must not learn the title of the next book in another.
// The settings cache is loaded during SetupServer, so guest access has to be seeded before the
// app is built rather than through setupTestAppWithDB.
func TestNextInSeriesRespectsLibraryVisibility(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "series-visibility.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	for id, name := range map[string]string{"open": "Open", "hidden": "Hidden"} {
		if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES (?, ?)`, id, name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`UPDATE app_settings SET value_json = ? WHERE key = 'guest_access.mode'`, `"selected_libraries"`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE app_settings SET value_json = ? WHERE key = 'guest_access.library_ids'`, `["open"]`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO series (id, name) VALUES ('s1', 'Dune')`); err != nil {
		t.Fatal(err)
	}
	for _, row := range [][3]string{{"b1", "open", "1"}, {"b2", "hidden", "2"}} {
		if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES (?, ?, ?, 'active')`,
			row[0], row[1], "Book "+row[2]); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO book_series (book_id, series_id, series_index) VALUES (?, 's1', ?)`, row[0], row[2]); err != nil {
			t.Fatal(err)
		}
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())
	app := server.App

	probe, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/books/b2", nil))
	if err != nil {
		t.Fatal(err)
	}
	if probe.StatusCode != http.StatusForbidden {
		t.Fatalf("guest can read the hidden book directly (%d); this test cannot detect a leak", probe.StatusCode)
	}

	body := seriesContext(t, app, "b1")
	if body.Data.Next != nil {
		t.Fatalf("a guest was shown the next book from a library they cannot read: %+v", body.Data.Next)
	}
}
