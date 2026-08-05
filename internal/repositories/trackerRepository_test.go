package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// GetByID and GetMappingByID used to fold a real database failure into the same branch as an
// empty result set (err != nil || len(rows) == 0 -> sql.ErrNoRows), so a broken or locked
// database was reported to the caller as "this tracker does not exist" and the mapping was
// silently dropped instead of surfacing as a 500.
func TestTrackerRepositoryDoesNotReportDatabaseFailuresAsNotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "tracker.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db, filepath.Join("..", "..", "db", "schema")); err != nil {
		t.Fatal(err)
	}

	repo := NewTrackerRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "missing-tracker"); !apperrors.IsNotFound(err) {
		t.Fatalf("missing tracker: err = %v, want not-found", err)
	}
	if _, err := repo.GetMappingByID(ctx, "missing-mapping"); !apperrors.IsNotFound(err) {
		t.Fatalf("missing mapping: err = %v, want not-found", err)
	}

	for _, table := range []string{"user_trackers", "book_tracker_mappings"} {
		if _, err := db.Exec(`DROP TABLE ` + table); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}

	if _, err := repo.GetByID(ctx, "tracker-after-drop"); apperrors.IsNotFound(err) {
		t.Errorf("a failing query was reported as not-found: %v", err)
	} else if err == nil {
		t.Error("a failing query returned no error at all")
	}
	if _, err := repo.GetMappingByID(ctx, "mapping-after-drop"); apperrors.IsNotFound(err) {
		t.Errorf("a failing mapping query was reported as not-found: %v", err)
	} else if err == nil {
		t.Error("a failing mapping query returned no error at all")
	}
}
