package repositories

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func progressAt(chapter int64) *models.ReadingProgressEntity {
	return &models.ReadingProgressEntity{
		UserID:       "u1",
		BookID:       "bk",
		ChapterID:    "ch",
		ChapterTitle: "Chapter",
		ChapterIndex: chapter,
	}
}

// Reads inside a transaction used to populate the shared cache, so a rollback left the cache serving a row the database no longer had.
func TestRolledBackTransactionLeavesNothingInCache(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "cache-tx.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	seed := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("seed %q: %v", query, err)
		}
	}
	seed(`INSERT INTO users (id, email, auth_provider, token_version) VALUES ('u1', 'u@example.com', 'LOCAL', 1)`)
	seed(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`)
	seed(`INSERT INTO books (id, library_id, title, status) VALUES ('bk', 'lib', 'Book', 'published')`)

	ramCache := cache.NewRamCache()
	repo := NewFeatureRepository(db, ramCache)
	ctx := context.Background()

	if _, err := repo.UpsertReadingProgress(ctx, progressAt(5)); err != nil {
		t.Fatalf("baseline write: %v", err)
	}
	if _, err := repo.GetReadingProgress(ctx, "u1", "bk"); err != nil {
		t.Fatalf("baseline read: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txRepo := repo.WithTx(tx)
	if _, err := txRepo.UpsertReadingProgress(ctx, progressAt(105)); err != nil {
		t.Fatalf("write in tx: %v", err)
	}
	if _, err := txRepo.GetReadingProgress(ctx, "u1", "bk"); err != nil {
		t.Fatalf("read in tx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var stored int64
	if err := db.QueryRow(`SELECT chapter_index FROM reading_progress WHERE user_id='u1' AND book_id='bk'`).Scan(&stored); err != nil {
		t.Fatalf("read committed row: %v", err)
	}
	if stored != 5 {
		t.Fatalf("database has chapter_index %d after rollback, want 5", stored)
	}

	served, err := repo.GetReadingProgress(ctx, "u1", "bk")
	if err != nil {
		t.Fatalf("read after rollback: %v", err)
	}
	if served.ChapterIndex != stored {
		t.Errorf("cache serves chapter_index %d but the database has %d — the rollback was not honoured",
			served.ChapterIndex, stored)
	}
}

// The shared singleflight group is the second, independent path to the same leak: a plain reader joining a call already in flight inside the transaction receives that call's result directly, without the cache being involved at all.
func TestConcurrentReadersNeverJoinATransactionsFlight(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "cache-tx-flight.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	for _, query := range []string{
		`INSERT INTO users (id, email, auth_provider, token_version) VALUES ('u1', 'u@example.com', 'LOCAL', 1)`,
		`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`,
		`INSERT INTO books (id, library_id, title, status) VALUES ('bk', 'lib', 'Book', 'published')`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	ramCache := cache.NewRamCache()
	repo := NewFeatureRepository(db, ramCache)
	ctx := context.Background()
	if _, err := repo.UpsertReadingProgress(ctx, progressAt(5)); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	const (
		rounds  = 200
		readers = 64
	)
	key := cache.BuildKey("feature", "reading_progress", "user", "u1", "book", "bk")
	leaked := 0

	for range rounds {
		_ = ramCache.Del(ctx, key)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		txRepo := repo.WithTx(tx)
		if _, err := txRepo.UpsertReadingProgress(ctx, progressAt(105)); err != nil {
			_ = tx.Rollback()
			t.Fatalf("write in tx: %v", err)
		}

		var start, done sync.WaitGroup
		start.Add(1)
		seen := make([]int64, readers)

		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			_, _ = txRepo.GetReadingProgress(ctx, "u1", "bk")
		}()
		for i := range readers {
			done.Add(1)
			go func(i int) {
				defer done.Done()
				start.Wait()
				if progress, err := repo.GetReadingProgress(ctx, "u1", "bk"); err == nil && progress != nil {
					seen[i] = progress.ChapterIndex
				}
			}(i)
		}
		start.Done()
		done.Wait()
		if err := tx.Rollback(); err != nil {
			t.Fatalf("rollback: %v", err)
		}

		for _, value := range seen {
			if value == 105 {
				leaked++
			}
		}
	}

	if leaked > 0 {
		t.Errorf("%d reads across %d rounds returned the uncommitted chapter_index 105", leaked, rounds)
	}
}
