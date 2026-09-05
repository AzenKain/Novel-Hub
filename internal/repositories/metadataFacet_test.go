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

// The browse pages used to fetch one page of authors and then apply the A-Z strip and the search box in the browser, so on a large library picking "Z" showed nothing even when Z authors existed past the first page.
func TestMetadataFacetFiltersRunInSQL(t *testing.T) {
	repo, db := newFacetTestRepo(t)
	ctx := context.Background()

	names := []string{"Alice", "bob", "Zoe", "9 Lives", "Ong Ba", "Đặng Thu", "đỏ nam"}
	for i, name := range names {
		authorID := fmt.Sprintf("author-%d", i)
		if _, err := db.ExecContext(ctx, `INSERT INTO authors (id, name) VALUES (?, ?)`, authorID, name); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO books (id, library_id, title, author_id) VALUES (?, 'lib-1', ?, ?)`,
			fmt.Sprintf("book-%d", i), name, authorID); err != nil {
			t.Fatal(err)
		}
	}

	scoped := func(f MetadataFacetFilter) MetadataFacetFilter {
		f.LibraryIDs = []string{"lib-1"}
		return f
	}
	for _, tc := range []struct {
		label  string
		filter MetadataFacetFilter
		want   []string
	}{
		{"alpha A", scoped(MetadataFacetFilter{Limit: 50, Alpha: "A"}), []string{"Alice"}},
		{"alpha Z", scoped(MetadataFacetFilter{Limit: 50, Alpha: "Z"}), []string{"Zoe"}},
		{"alpha B matches lowercase initial", scoped(MetadataFacetFilter{Limit: 50, Alpha: "B"}), []string{"bob"}},
		{"alpha O", scoped(MetadataFacetFilter{Limit: 50, Alpha: "O"}), []string{"Ong Ba"}},
		{"alpha D-stroke", scoped(MetadataFacetFilter{Limit: 50, Alpha: "Đ"}), []string{"Đặng Thu", "đỏ nam"}},
		{"alpha # is non-letter only", scoped(MetadataFacetFilter{Limit: 50, Alpha: "#"}), []string{"9 Lives"}},
		{"search is a substring match", scoped(MetadataFacetFilter{Limit: 50, Search: "o"}), []string{"bob", "Ong Ba", "Zoe"}},
		{"search and alpha combine", scoped(MetadataFacetFilter{Limit: 50, Search: "o", Alpha: "Z"}), []string{"Zoe"}},
		{"no filter returns everything in scope", scoped(MetadataFacetFilter{Limit: 50}), names},
		{"empty scope matches nothing", MetadataFacetFilter{Limit: 50}, nil},
		{"unreadable library matches nothing", MetadataFacetFilter{Limit: 50, LibraryIDs: []string{"lib-other"}}, nil},
	} {
		t.Run(tc.label, func(t *testing.T) {
			items, err := repo.ListAuthorsWithCount(ctx, tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(items))
			for i, item := range items {
				got[i] = item.Name
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			wanted := make(map[string]bool, len(tc.want))
			for _, name := range tc.want {
				wanted[name] = true
			}
			for _, name := range got {
				if !wanted[name] {
					t.Fatalf("got %v, want exactly %v", got, tc.want)
				}
			}
		})
	}
}

// Two filters must not share one cache entry, otherwise the second one served from cache returns the first one's rows.
func TestMetadataFacetCacheKeyVariesWithFilter(t *testing.T) {
	base := MetadataFacetFilter{Cursor: "c", Limit: 20}
	seen := map[string]string{}
	for label, filter := range map[string]MetadataFacetFilter{
		"plain":         base,
		"search":        {Cursor: "c", Limit: 20, Search: "abc"},
		"alpha":         {Cursor: "c", Limit: 20, Alpha: "A"},
		"both":          {Cursor: "c", Limit: 20, Search: "abc", Alpha: "A"},
		"other page":    {Cursor: "c2", Limit: 20},
		"one library":   {Cursor: "c", Limit: 20, LibraryIDs: []string{"lib-1"}},
		"two libraries": {Cursor: "c", Limit: 20, LibraryIDs: []string{"lib-1", "lib-2"}},
	} {
		key := filter.cacheKey("authors")
		if prev, ok := seen[key]; ok {
			t.Fatalf("%s and %s collide on cache key %q", label, prev, key)
		}
		seen[key] = label
	}
}

func newFacetTestRepo(t *testing.T) (*bookDBRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "facets.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Lib')`); err != nil {
		t.Fatal(err)
	}
	return NewBookDBRepository(db, nil).(*bookDBRepository), db
}
