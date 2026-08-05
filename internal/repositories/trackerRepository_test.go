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
	if err := database.ApplySchema(db); err != nil {
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

// book_tracker_mappings had no user_id and a UNIQUE(book_id, provider), so one row was shared by
// the whole instance. Any account with read access to a book could repoint it, and the victim's
// next sync then pushed progress to the attacker's chosen series using the victim's own OAuth
// token. The cache key was book-scoped too, so scoping only the SQL would still have served one
// reader's mapping to another out of RAM.
func TestBookTrackerMappingIsPerUser(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tracker-scope.db")
	t.Setenv("SQLITE_DB_PATH", dbPath)
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
	seed(`INSERT INTO users (id, email, full_name, auth_provider, token_version) VALUES
		('user-a', 'a@example.com', 'A', 'LOCAL', 1),
		('user-b', 'b@example.com', 'B', 'LOCAL', 1)`)
	seed(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main')`)
	seed(`INSERT INTO books (id, library_id, title, status) VALUES ('book-1', 'lib-1', 'Shared', 'published')`)

	// A live cache, because a book-scoped key would leak across users even with scoped SQL.
	repo := NewTrackerRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := repo.UpsertBookTrackerMapping(ctx, "user-a", "book-1", "anilist", "series-A"); err != nil {
		t.Fatalf("user A upsert: %v", err)
	}
	if _, err := repo.UpsertBookTrackerMapping(ctx, "user-b", "book-1", "anilist", "series-B"); err != nil {
		// With UNIQUE(book_id, provider) this call overwrote A's row instead of adding B's.
		t.Fatalf("user B upsert: %v", err)
	}

	for _, tc := range []struct{ user, want string }{
		{"user-a", "series-A"},
		{"user-b", "series-B"},
	} {
		mapping, err := repo.GetBookTrackerMapping(ctx, tc.user, "book-1", "anilist")
		if err != nil {
			t.Fatalf("%s read back: %v", tc.user, err)
		}
		if mapping.ExternalSeriesID != tc.want {
			t.Errorf("%s maps book-1 to %q, want %q", tc.user, mapping.ExternalSeriesID, tc.want)
		}
		if mapping.UserID != tc.user {
			t.Errorf("mapping.UserID = %q, want %q", mapping.UserID, tc.user)
		}
	}

	// A third user has no mapping at all; a global row would hand them someone else's.
	if _, err := repo.GetBookTrackerMapping(ctx, "user-c", "book-1", "anilist"); !apperrors.IsNotFound(err) {
		t.Errorf("an unmapped user got a mapping: err = %v", err)
	}
}
