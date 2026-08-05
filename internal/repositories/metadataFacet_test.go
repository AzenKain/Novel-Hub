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

// The browse pages used to fetch one page of authors and then apply the A-Z strip and the
// search box in the browser, so on a large library picking "Z" showed nothing even when Z
// authors existed past the first page. Both filters now run in SQL; these cases pin that,
// including the one that is easy to get wrong.
//
// SQLite's UPPER() is ASCII-only, so it will NOT fold Vietnamese d-with-stroke. If the alpha
// bucket were implemented with a plain UPPER() comparison, the lowercase form would silently
// land in the "#" bucket instead of its own, and lowercase-initial names would be
// unreachable from the strip.
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

	for _, tc := range []struct {
		label  string
		filter MetadataFacetFilter
		want   []string
	}{
		{"alpha A", MetadataFacetFilter{Limit: 50, Alpha: "A"}, []string{"Alice"}},
		{"alpha Z", MetadataFacetFilter{Limit: 50, Alpha: "Z"}, []string{"Zoe"}},
		{"alpha B matches lowercase initial", MetadataFacetFilter{Limit: 50, Alpha: "B"}, []string{"bob"}},
		// "Ong" starts with a plain ASCII O, so it belongs to bucket O — not to "#".
		{"alpha O", MetadataFacetFilter{Limit: 50, Alpha: "O"}, []string{"Ong Ba"}},
		// Both cases of d-with-stroke belong to the same bucket, and neither may leak into "#".
		{"alpha D-stroke", MetadataFacetFilter{Limit: 50, Alpha: "Đ"}, []string{"Đặng Thu", "đỏ nam"}},
		{"alpha # is non-letter only", MetadataFacetFilter{Limit: 50, Alpha: "#"}, []string{"9 Lives"}},
		// Plain substring match, same as the client's includes(): "đỏ" does not contain "o".
		{"search is a substring match", MetadataFacetFilter{Limit: 50, Search: "o"}, []string{"bob", "Ong Ba", "Zoe"}},
		{"search and alpha combine", MetadataFacetFilter{Limit: 50, Search: "o", Alpha: "Z"}, []string{"Zoe"}},
		{"no filter returns everything", MetadataFacetFilter{Limit: 50}, names},
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

// Two filters must not share one cache entry, otherwise the second one served from cache
// returns the first one's rows.
func TestMetadataFacetCacheKeyVariesWithFilter(t *testing.T) {
	base := MetadataFacetFilter{Cursor: "c", Limit: 20}
	seen := map[string]string{}
	for label, filter := range map[string]MetadataFacetFilter{
		"plain":      base,
		"search":     {Cursor: "c", Limit: 20, Search: "abc"},
		"alpha":      {Cursor: "c", Limit: 20, Alpha: "A"},
		"both":       {Cursor: "c", Limit: 20, Search: "abc", Alpha: "A"},
		"other page": {Cursor: "c2", Limit: 20},
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
