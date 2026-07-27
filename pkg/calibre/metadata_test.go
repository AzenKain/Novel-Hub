package calibre

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReadMetadataDB_FullMetadata(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "metadata.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create test metadata.db: %v", err)
	}

	schema := `
	CREATE TABLE books (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		sort TEXT,
		author_sort TEXT,
		path TEXT NOT NULL,
		pubdate DATETIME,
		timestamp DATETIME,
		has_cover BOOLEAN DEFAULT 0,
		uuid TEXT,
		isbn TEXT,
		lccn TEXT,
		series_index REAL
	);
	CREATE TABLE comments (id INTEGER PRIMARY KEY, book INTEGER, text TEXT);
	CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE books_authors_link (id INTEGER PRIMARY KEY, book INTEGER, author INTEGER);
	CREATE TABLE series (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE books_series_link (id INTEGER PRIMARY KEY, book INTEGER, series INTEGER);
	CREATE TABLE publishers (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE books_publishers_link (id INTEGER PRIMARY KEY, book INTEGER, publisher INTEGER);
	CREATE TABLE languages (id INTEGER PRIMARY KEY, lang_code TEXT);
	CREATE TABLE books_languages_link (id INTEGER PRIMARY KEY, book INTEGER, lang_code INTEGER);
	CREATE TABLE tags (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE books_tags_link (id INTEGER PRIMARY KEY, book INTEGER, tag INTEGER);
	CREATE TABLE ratings (id INTEGER PRIMARY KEY, rating INTEGER);
	CREATE TABLE books_ratings_link (id INTEGER PRIMARY KEY, book INTEGER, rating INTEGER);
	CREATE TABLE identifiers (id INTEGER PRIMARY KEY, book INTEGER, type TEXT, val TEXT);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to execute schema: %v", err)
	}

	_, _ = db.Exec("INSERT INTO books (id, title, sort, author_sort, path, pubdate, has_cover, uuid, isbn, series_index) VALUES (1, 'Dune', 'Dune', 'Herbert, Frank', 'Frank Herbert/Dune (1)', '1965-08-01', 1, 'uuid-1234', '9780441172719', 1.0)")
	_, _ = db.Exec("INSERT INTO comments (book, text) VALUES (1, '<p>Set on the desert planet Arrakis...</p>')")
	_, _ = db.Exec("INSERT INTO authors (id, name) VALUES (10, 'Frank Herbert')")
	_, _ = db.Exec("INSERT INTO books_authors_link (book, author) VALUES (1, 10)")
	_, _ = db.Exec("INSERT INTO series (id, name) VALUES (20, 'Dune Chronicles')")
	_, _ = db.Exec("INSERT INTO books_series_link (book, series) VALUES (1, 20)")
	_, _ = db.Exec("INSERT INTO publishers (id, name) VALUES (30, 'Chilton Books')")
	_, _ = db.Exec("INSERT INTO books_publishers_link (book, publisher) VALUES (1, 30)")
	_, _ = db.Exec("INSERT INTO languages (id, lang_code) VALUES (40, 'eng')")
	_, _ = db.Exec("INSERT INTO books_languages_link (book, lang_code) VALUES (1, 40)")
	_, _ = db.Exec("INSERT INTO tags (id, name) VALUES (50, 'Sci-Fi'), (51, 'Classics')")
	_, _ = db.Exec("INSERT INTO books_tags_link (book, tag) VALUES (1, 50), (1, 51)")
	_, _ = db.Exec("INSERT INTO ratings (id, rating) VALUES (60, 10)")
	_, _ = db.Exec("INSERT INTO books_ratings_link (book, rating) VALUES (1, 60)")
	_, _ = db.Exec("INSERT INTO identifiers (book, type, val) VALUES (1, 'goodreads', '23422'), (1, 'isbn', '9780441172719')")

	db.Close()

	records, err := ReadMetadataDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("ReadMetadataDB failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0]
	if rec.Title != "Dune" {
		t.Errorf("expected title 'Dune', got '%s'", rec.Title)
	}
	if rec.Description == "" {
		t.Errorf("expected non-empty description")
	}
	if len(rec.Authors) != 1 || rec.Authors[0] != "Frank Herbert" {
		t.Errorf("unexpected authors: %v", rec.Authors)
	}
	if rec.Series != "Dune Chronicles" {
		t.Errorf("expected series 'Dune Chronicles', got '%s'", rec.Series)
	}
	if rec.SeriesIndex == nil || *rec.SeriesIndex != "1" {
		t.Errorf("expected series index '1', got %v", rec.SeriesIndex)
	}
	if len(rec.Publishers) != 1 || rec.Publishers[0] != "Chilton Books" {
		t.Errorf("unexpected publishers: %v", rec.Publishers)
	}
	if len(rec.Languages) != 1 || rec.Languages[0] != "eng" {
		t.Errorf("unexpected languages: %v", rec.Languages)
	}
	if len(rec.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(rec.Tags))
	}
	if rec.Rating == nil || *rec.Rating != 5.0 {
		t.Errorf("expected rating 5.0, got %v", rec.Rating)
	}
	if rec.Identifiers["goodreads"] != "23422" {
		t.Errorf("expected goodreads identifier 23422, got %s", rec.Identifiers["goodreads"])
	}
}
