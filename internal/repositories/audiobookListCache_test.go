package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// New chapters change which books have audio, so ListBooksWithAudio's page
// caches must be dropped on every chapter mutation — otherwise the vbook audio
// shelf goes stale after a chapter merge.
func TestAudiobookListInvalidatedOnUpsert(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ram := cache.NewRamCache()
	repo := NewAudiobookRepository(db, ram)

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib1','L')`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"b1", "b2"} {
		if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status, updated_at) VALUES (?, 'lib1', ?, 'active', ?)`,
			id, "Book "+id, time.Now().UTC().Format("2006-01-02 15:04:05")); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES (?, ?, ?, 'mp3', 1, ?)`,
			id+"f", id, "/tmp/"+id+".mp3", time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}

	// Seed one chapter on b1, then read the list twice to populate + hit cache
	if _, err := repo.UpsertChapter(ctx, "c1", "b1", strPtr("b1f"), 0, "Ch1", 0.0, nil); err != nil {
		t.Fatal(err)
	}
	listA, err := repo.ListBooksWithAudio(ctx, nil, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	// Second read is a cache hit; still the same ids
	if _, err := repo.ListBooksWithAudio(ctx, nil, "", 10); err != nil {
		t.Fatal(err)
	}
	if len(listA) != 1 || listA[0] != "b1" {
		t.Fatalf("initial list = %v, want [b1]", listA)
	}

	// Upsert a chapter on b2 — the list cache must refresh to include it.
	if _, err := repo.UpsertChapter(ctx, "c2", "b2", strPtr("b2f"), 0, "Ch1", 0.0, nil); err != nil {
		t.Fatal(err)
	}
	listB, err := repo.ListBooksWithAudio(ctx, nil, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listB) != 2 {
		t.Fatalf("after upsert list = %v, want 2 books", listB)
	}

	// Delete the only chapter of b1 — it must leave the list.
	if err := repo.DeleteChaptersForBook(ctx, "b1"); err != nil {
		t.Fatal(err)
	}
	listC, err := repo.ListBooksWithAudio(ctx, nil, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listC) != 1 || listC[0] != "b2" {
		t.Fatalf("after delete list = %v, want [b2]", listC)
	}
}

func strPtr(s string) *string { return &s }