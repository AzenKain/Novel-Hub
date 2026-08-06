package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func probeDB(tb testing.TB) *sql.DB {
	tb.Helper()
	dsn := filepath.Join(tb.TempDir(), "probe.db") +
		"?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)" +
		"&_pragma=trusted_schema(OFF)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" +
		"&_pragma=temp_store(MEMORY)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatal(err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)
	tb.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	return db
}

// A library scan calls CreateBookWithFile once per book, and each call fires 4 DelByPattern
// invalidations. DelByPattern has no key index -- it walks every entry in the cache -- so the
// per-book cost grows with how warm the cache is. Assert it stays flat.
func TestProbeRealScanInvalidation(t *testing.T) {
	ctx := context.Background()

	for _, warm := range []int{0, 8000, 32000} {
		db := probeDB(t)
		if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`); err != nil {
			t.Fatal(err)
		}

		c := cache.NewTheineCache(512 << 20)
		for i := 0; i < warm; i++ {
			_ = c.Set(ctx, cache.BuildKey("book", "id", fmt.Sprintf("warm-%06d", i)), i, time.Hour)
		}
		repo := NewBookDBRepository(db, c)

		const books = 100
		start := time.Now()
		for i := 0; i < books; i++ {
			bookID := fmt.Sprintf("bk-%06d", i)
			book := &models.BookEntity{ID: bookID, LibraryID: "lib-1", Title: "T", Status: "processing"}
			err := repo.CreateBookWithFile(ctx, book, &sqlc.CreateBookFileParams{
				ID:        fmt.Sprintf("f-%06d", i),
				BookID:    bookID,
				Path:      fmt.Sprintf("/books/%s.epub", bookID),
				Format:    "epub",
				SizeBytes: 1024,
				ModTime:   time.Now(),
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		elapsed := time.Since(start)
		perBook := elapsed / books
		t.Logf("warm=%6d: %d books in %v (%v/book, %v projected for 100k)",
			warm, books, elapsed.Round(time.Millisecond), perBook.Round(time.Microsecond),
			(perBook * 100000).Round(time.Second))
	}
}

// The path column stores the path as imported; entities carry the resolved one
// (models.BookFileEntity.FromSqlc runs localfs.ResolveBookFilePath). A path-keyed cache entry is
// therefore unreachable by an id-keyed Del, so nothing may cache under book_file:path:*.
func TestProbeBookFilePathIsNotCached(t *testing.T) {
	ctx := context.Background()
	db := probeDB(t)
	c := cache.NewTheineCache(512 << 20)

	if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`); err != nil {
		t.Fatal(err)
	}
	repo := NewBookDBRepository(db, c)
	const rawPath = "/originally/imported/from/here/volume.epub"
	book := &models.BookEntity{ID: "bk-1", LibraryID: "lib-1", Title: "T", Status: "active"}
	if err := repo.CreateBookWithFile(ctx, book, &sqlc.CreateBookFileParams{
		ID: "f-1", BookID: "bk-1", Path: rawPath, Format: "epub", SizeBytes: 1024, ModTime: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	entity, err := repo.GetBookFileByPath(ctx, rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetFilesByBookId(ctx, "bk-1"); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{
		cache.BuildKey("book_file", "path", rawPath),
		cache.BuildKey("book_file", "path", entity.Path),
	} {
		if ok, _ := c.Exists(ctx, key); ok {
			t.Errorf("%s is cached; an id-keyed Del cannot reach a path-keyed entry", key)
		}
	}
}
