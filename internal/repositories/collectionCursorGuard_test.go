package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// SQLite CURRENT_TIMESTAMP has one-second resolution, so collections created in the same batch share
// a created_at. A cursor carrying only that timestamp cannot break the tie and the walk stalls after
// one page — the reason GetCollections emits "<created_at>|<id>" and the reason the frontend must
// send back the server's cursor rather than rebuilding one from the last row's created_at.
func TestCollectionCursorNeedsTheIDHalf(t *testing.T) {
	ctx := context.Background()
	db := probeDB(t)
	if _, err := db.Exec(`INSERT INTO users (id,email,password_hash,full_name) VALUES ('u-1','u@e.com','x','U')`); err != nil {
		t.Fatal(err)
	}
	repo := NewFeatureRepository(db, nil)
	const total = 12
	for i := 0; i < total; i++ {
		if _, err := repo.CreateCollection(ctx, fmt.Sprintf("c-%02d", i), fmt.Sprintf("C%02d", i), "u-1"); err != nil {
			t.Fatal(err)
		}
	}

	walk := func(keepID bool) []string {
		var cursor *time.Time
		cursorID := ""
		seen := make([]string, 0, total)
		for page := 0; page < total; page++ {
			got, err := repo.GetUserCollections(ctx, "u-1", cursor, cursorID, 5)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range got {
				seen = append(seen, c.ID)
			}
			if len(got) < 5 {
				break
			}
			last := got[len(got)-1]
			cursor = &last.CreatedAt
			cursorID = ""
			if keepID {
				cursorID = last.ID
			}
		}
		return seen
	}

	if full := walk(true); len(full) != total {
		t.Errorf("the server cursor walked %d of %d collections: %v", len(full), total, full)
	}
	if partial := walk(false); len(partial) == total {
		t.Errorf("a timestamp-only cursor walked all %d collections, so this guard no longer proves the id half is load-bearing", total)
	}
}
