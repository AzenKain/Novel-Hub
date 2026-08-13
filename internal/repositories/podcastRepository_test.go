package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newPodcastTestRepo(t *testing.T) (PodcastRepository, *sql.DB, cache.Cache) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	c := cache.NewRamCache()
	return NewPodcastRepository(db, c), db, c
}

func TestPodcastEpisodeDeleteBookTrigger(t *testing.T) {
	repo, db, c := newPodcastTestRepo(t)
	ctx := context.Background()

	// 1. Create a library
	if _, err := db.ExecContext(ctx, `INSERT INTO libraries (id, name) VALUES ('lib-1', 'Test Library')`); err != nil {
		t.Fatal(err)
	}

	// 2. Create a book
	if _, err := db.ExecContext(ctx, `INSERT INTO books (id, library_id, title) VALUES ('book-1', 'lib-1', 'Test Episode Book')`); err != nil {
		t.Fatal(err)
	}

	// 3. Create a podcast
	author := "Test Author"
	feedURL := "https://example.com/podcast.xml"
	_, err := repo.CreatePodcast(ctx, "pod-1", "lib-1", feedURL, "Test Podcast", nil, nil, &author)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Create a podcast episode
	ep, err := repo.UpsertEpisode(ctx, "ep-1", "pod-1", "guid-1", "Test Episode", nil, "https://example.com/audio.mp3", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Downloaded {
		t.Fatal("expected episode to be not downloaded initially")
	}

	// 5. Mark episode as downloaded
	err = repo.MarkEpisodeDownloaded(ctx, "ep-1", "book-1")
	if err != nil {
		t.Fatal(err)
	}

	// 6. Verify it is downloaded
	ep, err = repo.GetEpisode(ctx, "ep-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ep.Downloaded {
		t.Fatal("expected episode to be downloaded")
	}
	if ep.BookID == nil || *ep.BookID != "book-1" {
		t.Fatalf("expected book_id to be 'book-1', got %v", ep.BookID)
	}

	// 7. Delete the book via repository to trigger trigger + cache invalidation
	bookRepo := NewBookDBRepository(db, c)
	if err := bookRepo.DeleteBook(ctx, "book-1"); err != nil {
		t.Fatal(err)
	}

	// 8. Verify trigger set downloaded = false and book_id = NULL
	ep, err = repo.GetEpisode(ctx, "ep-1")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Downloaded {
		t.Fatal("expected episode downloaded state to be reset to false after book deletion")
	}
	if ep.BookID != nil {
		t.Fatalf("expected book_id to be NULL, got %v", *ep.BookID)
	}
}
