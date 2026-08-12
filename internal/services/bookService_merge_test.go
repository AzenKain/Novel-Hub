package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestMergeSimilarity(t *testing.T) {
	if !sameAuthor("J.K. Rowling", "J K Rowling") {
		t.Error("sameAuthor should match punctuation-differing names")
	}
	if !sameAuthor("", "") {
		t.Error("sameAuthor should match two empty authors")
	}
	if sameAuthor("George Orwell", "J.K. Rowling") {
		t.Error("sameAuthor should reject unrelated names")
	}
	if sameAuthor("J.K. Rowling", "Agatha Christie") {
		t.Error("sameAuthor should reject different names")
	}

	cases := []struct {
		a, b string
		want bool
	}{
		{"Harry Potter and the Philosopher's Stone", "Harry Potter and the Philosophers Stone", true},
		{"Solo Leveling", "Solo Leveling", true},
		{"The Great Gatsby", "Harry Potter", false},
		{"Dune", "Dune Messiah", false},
	}
	for _, c := range cases {
		got := titleMatches(normalizeTitle(c.a), normalizeTitle(c.b))
		if got != c.want {
			t.Errorf("titleMatches(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestMergeBooks(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "merge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	seed := []string{
		`INSERT INTO users (id, email, password_hash) VALUES ('u-1', 'u@e.com', 'x')`,
		`INSERT INTO libraries (id, name) VALUES ('lib-1', 'L')`,
		`INSERT INTO authors (id, name) VALUES ('a-1', 'J.K. Rowling')`,
		`INSERT INTO books (id, library_id, title, author_id, status) VALUES ('src-1', 'lib-1', 'Harry Potter and the Philosophers Stone', 'a-1', 'ready')`,
		`INSERT INTO books (id, library_id, title, author_id, status) VALUES ('tgt-1', 'lib-1', 'Harry Potter and the Philosopher''s Stone', 'a-1', 'ready')`,
		`INSERT INTO chapters (id, book_id, title, chapter_index) VALUES ('ch-src-1', 'src-1', 'Ch1', 1)`,
		`INSERT INTO tags (id, name) VALUES ('tag-1', 'Fantasy')`,
		`INSERT INTO book_tags (book_id, tag_id) VALUES ('src-1', 'tag-1')`,
		`INSERT INTO highlights (id, user_id, book_id, chapter_id, text_content, start_index, end_index) VALUES ('hl-1', 'u-1', 'src-1', 'ch-src-1', 'quote', 0, 5)`,
		`INSERT INTO reading_sessions (id, user_id, book_id, duration_seconds, words_read, session_date) VALUES ('rs-1', 'u-1', 'src-1', 100, 500, '2026-08-01')`,
		`INSERT INTO book_read_stats (book_id, total_open_count, qualified_read_count) VALUES ('src-1', 3, 2)`,
		`INSERT INTO book_read_stats (book_id, total_open_count, qualified_read_count) VALUES ('tgt-1', 1, 1)`,
	}
	for _, stmt := range seed {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	// fts_chapters has no trigger; the app populates it via InsertFTSChapter.
	// Mirror that so MergeFTSChapters is exercised.
	if _, err := db.Exec(`INSERT INTO fts_chapters (rowid, book_id, chapter_id, title, content)
		SELECT c.rowid, c.book_id, c.id, c.title, '' FROM chapters c`); err != nil {
		t.Fatalf("seed fts_chapters: %v", err)
	}

	booksDir := t.TempDir()
	fileRepo, err := repositories.NewBookFileRepository(booksDir)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := fileRepo.SaveBook(context.Background(), "src-1", "source.epub", strings.NewReader("epub-content"))
	if err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, c)
	bookService := NewBookService(bookRepo, nil, nil, fileRepo, bookparser.NewRegistry(), database.NewTxManager(db), nil, nil, nil, nil)

	if err := bookRepo.CreateBookFile(context.Background(), sqlc.CreateBookFileParams{
		ID: "bf-1", BookID: "src-1", Path: saved.Path, Format: "epub",
		SizeBytes: saved.SizeBytes, ModTime: saved.ModTime,
		Hash: convertStrToNull("hash-1"), State: convertStrToNull("managed"),
	}); err != nil {
		t.Fatal(err)
	}

	if err := bookService.MergeBooks(context.Background(), "src-1", "tgt-1"); err != nil {
		t.Fatalf("MergeBooks: %v", err)
	}

	assertRow := func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatalf("query %q: %v", q, err)
		}
		return n
	}

	if n := assertRow(`SELECT COUNT(*) FROM books WHERE id = 'src-1'`); n != 0 {
		t.Errorf("source book still exists: %d", n)
	}
	if n := assertRow(`SELECT COUNT(*) FROM chapters WHERE book_id = 'tgt-1'`); n != 1 {
		t.Errorf("chapter not moved to target: %d", n)
	}
	if n := assertRow(`SELECT COUNT(*) FROM book_files WHERE book_id = 'tgt-1'`); n != 1 {
		t.Errorf("file row not moved to target: %d", n)
	}
	if n := assertRow(`SELECT COUNT(*) FROM book_tags WHERE book_id = 'tgt-1' AND tag_id = 'tag-1'`); n != 1 {
		t.Errorf("tag not merged to target: %d", n)
	}
	if n := assertRow(`SELECT COUNT(*) FROM highlights WHERE book_id = 'tgt-1'`); n != 1 {
		t.Errorf("highlight not moved to target: %d", n)
	}
	if n := assertRow(`SELECT COUNT(*) FROM fts_chapters WHERE book_id = 'tgt-1'`); n != 1 {
		t.Errorf("fts chapter not re-pointed to target: %d", n)
	}
	if n := assertRow(`SELECT COUNT(*) FROM reading_sessions WHERE book_id = 'tgt-1' AND duration_seconds = 100`); n != 1 {
		t.Errorf("reading session not moved to target: %d", n)
	}
	if n := assertRow(`SELECT total_open_count FROM book_read_stats WHERE book_id = 'tgt-1'`); n != 4 {
		t.Errorf("read stats not folded: %d", n)
	}

	if _, err := os.Stat(filepath.Join(booksDir, "tgt-1", "source.epub")); err != nil {
		t.Errorf("physical file not moved to target dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(booksDir, "src-1")); !os.IsNotExist(err) {
		t.Errorf("source dir not removed: %v", err)
	}
}

func TestMergeBooksSelfMergeRejected(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "merge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	c := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, c)
	fileRepo, err := repositories.NewBookFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bookService := NewBookService(bookRepo, nil, nil, fileRepo, bookparser.NewRegistry(), database.NewTxManager(db), nil, nil, nil, nil)
	if err := bookService.MergeBooks(context.Background(), "same", "same"); err == nil {
		t.Error("self-merge should be rejected")
	}
}

func convertStrToNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}