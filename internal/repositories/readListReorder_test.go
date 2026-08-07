package repositories

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"novelhub/pkg/cache"
)

// Swapping two neighbours is the one case a UNIQUE(read_list_id, position) index would break: the
// first UPDATE has to park a row on a number the other row still holds. Do it through the real
// repository so the schema, not a comment, is what proves the constraint was left off on purpose.
func TestReorderSwapsAdjacentEntries(t *testing.T) {
	repo, _, ctx := readListProbe(t, cache.NewRamCache())
	if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Monogatari", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if err := repo.AppendBookToReadList(ctx, "rl-1", fmt.Sprintf("bk-%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.ReplaceReadListOrder(ctx, "rl-1", []string{"bk-1", "bk-0", "bk-2", "bk-3"}); err != nil {
		t.Fatalf("swapping two adjacent entries failed: %v", err)
	}
	ids, err := repo.GetReadListBookIDs(ctx, "rl-1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"bk-1", "bk-0", "bk-2", "bk-3"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("after the swap order = %v, want %v", ids, want)
	}

	if err := repo.ReplaceReadListOrder(ctx, "rl-1", []string{"bk-0", "bk-2", "bk-3", "bk-1"}); err != nil {
		t.Fatal(err)
	}
	ids, err = repo.GetReadListBookIDs(ctx, "rl-1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"bk-0", "bk-2", "bk-3", "bk-1"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("after moving the head to the tail order = %v, want %v", ids, want)
	}
}

// A drag that reverses the whole list writes every position at once. Doing it in one pass through
// SetReadListBookPosition means no intermediate state is ever readable as a valid order.
func TestReorderHandlesFullReversal(t *testing.T) {
	repo, _, ctx := readListProbe(t, nil)
	if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Reverse", ""); err != nil {
		t.Fatal(err)
	}
	forward := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("bk-%d", i)
		if err := repo.AppendBookToReadList(ctx, "rl-1", id); err != nil {
			t.Fatal(err)
		}
		forward = append(forward, id)
	}
	reversed := make([]string, len(forward))
	for i, id := range forward {
		reversed[len(forward)-1-i] = id
	}
	if err := repo.ReplaceReadListOrder(ctx, "rl-1", reversed); err != nil {
		t.Fatal(err)
	}
	ids, err := repo.GetReadListBookIDs(ctx, "rl-1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, reversed) {
		t.Errorf("order = %v, want %v", ids, reversed)
	}
}

// SQLite CURRENT_TIMESTAMP has one-second resolution, so read lists created in the same batch share
// a created_at. A cursor carrying only that timestamp cannot break the tie and the walk stalls after
// one page — the reason the API emits "<created_at>|<id>" and the frontend must echo it back rather
// than rebuilding a cursor from the last row's created_at.
func TestReadListCursorNeedsTheIDHalf(t *testing.T) {
	repo, _, ctx := readListProbe(t, nil)
	const total = 12
	for i := 0; i < total; i++ {
		if _, err := repo.CreateReadList(ctx, fmt.Sprintf("rl-%02d", i), "u-1", fmt.Sprintf("L%02d", i), ""); err != nil {
			t.Fatal(err)
		}
	}

	walk := func(keepID bool) []string {
		var cursor *time.Time
		cursorID := ""
		seen := make([]string, 0, total)
		for page := 0; page < total; page++ {
			got, err := repo.GetUserReadLists(ctx, "u-1", cursor, cursorID, 5)
			if err != nil {
				t.Fatal(err)
			}
			for _, list := range got {
				seen = append(seen, list.ID)
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
		t.Errorf("the server cursor walked %d of %d read lists: %v", len(full), total, full)
	}
	if partial := walk(false); len(partial) == total {
		t.Errorf("a timestamp-only cursor walked all %d read lists, so this guard no longer proves the id half is load-bearing", total)
	}
}

// The first page is cached under a key that carries the limit. Deleting a bare "read_list:user:<id>"
// would miss it — the exact defect the collections repository still has — so a list created after
// the page was cached has to show up anyway.
func TestUserReadListsFirstPageIsNotStaleAfterWrite(t *testing.T) {
	repo, _, ctx := readListProbe(t, cache.NewRamCache())
	for i := 0; i < 3; i++ {
		if _, err := repo.CreateReadList(ctx, fmt.Sprintf("rl-%02d", i), "u-1", fmt.Sprintf("L%02d", i), ""); err != nil {
			t.Fatal(err)
		}
	}
	if lists, err := repo.GetUserReadLists(ctx, "u-1", nil, "", 20); err != nil {
		t.Fatal(err)
	} else if len(lists) != 3 {
		t.Fatalf("got %d lists, want 3", len(lists))
	}

	if _, err := repo.CreateReadList(ctx, "rl-99", "u-1", "Later", ""); err != nil {
		t.Fatal(err)
	}
	lists, err := repo.GetUserReadLists(ctx, "u-1", nil, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 4 {
		t.Fatalf("got %d lists, want 4 — the cached first page outlived the create", len(lists))
	}

	if err := repo.DeleteReadList(ctx, "rl-99", "u-1"); err != nil {
		t.Fatal(err)
	}
	lists, err = repo.GetUserReadLists(ctx, "u-1", nil, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 3 {
		t.Errorf("got %d lists after the delete, want 3", len(lists))
	}
}

// Ownership is cached per (user, list) and gates every read in the service. A list deleted while its
// "yes" answer sits in the cache must not stay reachable for the rest of the TTL.
func TestOwnershipCacheDropsWithTheList(t *testing.T) {
	repo, _, ctx := readListProbe(t, cache.NewRamCache())
	if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Owned", ""); err != nil {
		t.Fatal(err)
	}
	owned, err := repo.ReadListOwnedByUser(ctx, "rl-1", "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if !owned {
		t.Fatal("the creator does not own the list")
	}
	if err := repo.DeleteReadList(ctx, "rl-1", "u-1"); err != nil {
		t.Fatal(err)
	}
	owned, err = repo.ReadListOwnedByUser(ctx, "rl-1", "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if owned {
		t.Error("a deleted list still reports as owned — the cached ownership answer survived it")
	}
}
