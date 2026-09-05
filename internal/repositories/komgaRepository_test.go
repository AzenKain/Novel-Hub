package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestKomgaJSONEachAcceptsStringArg(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO libraries (id, name) VALUES ('lib-1','L')`,
		`INSERT INTO books (id, library_id, title) VALUES ('b-1','lib-1','B1'),('b-2','lib-1','B2')`,
		`INSERT INTO series (id, name) VALUES ('s-1','Series One')`,
		`INSERT INTO book_series (book_id, series_id, series_index) VALUES ('b-1','s-1','1'),('b-2','s-1','2.5')`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewKomgaRepository(db, nil)
	got, err := repo.ListSeries(ctx, []string{"lib-1"}, "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].BookCount != 2 {
		t.Fatalf("ListSeries = %+v, want 1 series with 2 books", got)
	}
	if got[0].LastModified == "" {
		t.Error("LastModified empty; fileLastModified is non-nullable in the Kotlin DTO")
	}

	books, err := repo.ListSeriesBooks(ctx, "s-1", []string{"lib-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 || books[1].NumberSort != 2.5 {
		t.Fatalf("books numberSort = %+v, want second = 2.5 (fraction must survive)", books)
	}

	if _, err := repo.ListSeries(ctx, []string{"lib-nope"}, "", 20, 0); err != nil {
		t.Fatal(err)
	}
	if other, _ := repo.ListSeries(ctx, []string{"lib-nope"}, "", 20, 0); len(other) != 0 {
		t.Errorf("library filter leaked: %+v", other)
	}

	hit, err := repo.ListSeries(ctx, []string{"lib-1"}, "One", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hit) != 1 {
		t.Errorf("search=One returned %d rows, want 1", len(hit))
	}
	if m, _ := repo.ListSeries(ctx, []string{"lib-1"}, "zzz", 20, 0); len(m) != 0 {
		t.Errorf("search=zzz returned %+v, want none", m)
	}
}

// Cache-by-ids: the list key holds only ids, and each series is cached per id, so a second listing that overlaps reuses the entity rows.
func TestKomgaSeriesCacheIsScopedByLibrary(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO libraries (id, name) VALUES ('lib-1','L1'),('lib-2','L2')`,
		`INSERT INTO books (id, library_id, title) VALUES ('b-1','lib-1','B1'),('b-2','lib-2','B2')`,
		`INSERT INTO series (id, name) VALUES ('s-1','Shared')`,
		`INSERT INTO book_series (book_id, series_id, series_index) VALUES ('b-1','s-1','1'),('b-2','s-1','2')`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewKomgaRepository(db, cache.NewRamCache())

	narrow, err := repo.ListSeries(ctx, []string{"lib-1"}, "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(narrow) != 1 || narrow[0].BookCount != 1 {
		t.Fatalf("lib-1 scope = %+v, want 1 series with 1 book", narrow[0])
	}
	if narrow[0].LibraryID != "lib-1" {
		t.Errorf("libraryId = %q, want lib-1", narrow[0].LibraryID)
	}

	wide, err := repo.ListSeries(ctx, []string{"lib-1", "lib-2"}, "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(wide) != 1 || wide[0].BookCount != 2 {
		t.Fatalf("two-library scope = %+v, want 1 series with 2 books", wide[0])
	}

	again, err := repo.ListSeries(ctx, []string{"lib-1"}, "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 || again[0].BookCount != 1 {
		t.Errorf("cached lib-1 scope = %+v, want 1 book", again[0])
	}
}

func TestKomgaGetSeriesByIDsPreservesOrderAndSkipsUnknown(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO libraries (id, name) VALUES ('lib-1','L')`,
		`INSERT INTO books (id, library_id, title) VALUES ('b-1','lib-1','B1'),('b-2','lib-1','B2')`,
		`INSERT INTO series (id, name) VALUES ('s-1','Alpha'),('s-2','Beta')`,
		`INSERT INTO book_series (book_id, series_id, series_index) VALUES ('b-1','s-1','1'),('b-2','s-2','1')`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewKomgaRepository(db, cache.NewRamCache())
	got, err := repo.GetSeriesByIDs(ctx, []string{"s-2", "missing", "s-1"}, []string{"lib-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "s-2" || got[1].ID != "s-1" {
		t.Fatalf("GetSeriesByIDs = %v, want [s-2 s-1] in requested order", fileIDsOf(got))
	}
}

func fileIDsOf(items []*models.KomgaSeriesEntity) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}
