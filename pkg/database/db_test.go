package database

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestSQLiteConcurrency(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_concurrency.db")
	t.Setenv("SQLITE_DB_PATH", tempDB)

	db, err := NewSQLiteDB()
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	if err := ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	var wg sync.WaitGroup
	workers := 20
	iterations := 10

	for i := range workers {
		workerID := i
		wg.Go(func() {
			ctx := context.Background()
			for j := range iterations {
				tx, err := BeginImmediateTx(ctx, db)
				if err != nil {
					t.Errorf("worker %d failed to begin tx: %v", workerID, err)
					return
				}
				_, err = tx.ExecContext(ctx, "INSERT INTO tags (id, name) VALUES (?, ?) ON CONFLICT DO NOTHING",
					fmt.Sprintf("tag-%d-%d", workerID, j),
					fmt.Sprintf("Tag %d %d", workerID, j),
				)
				if err != nil {
					_ = tx.Rollback()
					ReleaseWriteLock()
					t.Errorf("worker %d failed to insert: %v", workerID, err)
					return
				}
				if err := tx.Commit(); err != nil {
					ReleaseWriteLock()
					t.Errorf("worker %d failed to commit: %v", workerID, err)
					return
				}
				ReleaseWriteLock()

				rows, err := db.QueryContext(ctx, "SELECT id FROM tags LIMIT 5")
				if err != nil {
					t.Errorf("worker %d failed to query: %v", workerID, err)
					return
				}
				rows.Close()
			}
		})
	}

	wg.Wait()
}
