package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/database"
)

// Keeps the newest N finished rows, never touching rows a worker still owns. Now that
// it runs on a schedule (not just at startup) the pending/running exemption matters.
func TestPruneFinishedJobsKeepsNewestAndSparesUnfinished(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db, filepath.Join("..", "..", "db", "schema")); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// 5 finished at distinct ages, plus one pending and one running.
	for i, status := range []string{"completed", "failed", "completed", "failed", "completed"} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO jobs (id, type, status, updated_at) VALUES (?, 'scan', ?, datetime('now', ?))`,
			fmt.Sprintf("finished-%d", i), status, fmt.Sprintf("-%d hours", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO jobs (id, type, status, updated_at) VALUES ('pending-1','scan','pending', datetime('now','-9 hours')),
		 ('running-1','scan','running', datetime('now','-9 hours'))`); err != nil {
		t.Fatal(err)
	}

	repo := NewJobRepository(db, nil)
	if err := repo.PruneFinishedJobs(ctx, 2); err != nil {
		t.Fatal(err)
	}

	var finished, unfinished int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jobs WHERE status IN ('completed','failed')`).Scan(&finished); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jobs WHERE status IN ('pending','running')`).Scan(&unfinished); err != nil {
		t.Fatal(err)
	}
	if finished != 2 {
		t.Errorf("keep=2 should leave 2 finished rows, got %d", finished)
	}
	if unfinished != 2 {
		t.Errorf("prune must never delete pending/running jobs, got %d of 2", unfinished)
	}
}
