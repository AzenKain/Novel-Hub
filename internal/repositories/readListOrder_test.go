package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"novelhub/pkg/cache"
)

func readListProbe(tb testing.TB, c cache.Cache) (ReadListRepository, *sql.DB, context.Context) {
	tb.Helper()
	ctx := context.Background()
	db := probeDB(tb)
	stmts := []string{
		`INSERT INTO users (id,email,password_hash,full_name) VALUES ('u-1','u@e.com','x','U')`,
		`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`,
	}
	for i := 0; i < 5; i++ {
		stmts = append(stmts, fmt.Sprintf(`INSERT INTO books (id,library_id,title,status) VALUES ('bk-%d','lib-1','B%d','active')`, i, i))
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			tb.Fatal(err)
		}
	}
	return NewReadListRepository(db, c), db, ctx
}

// Every mutation has to be visible through the cache exactly as it is through a cold read: an
// append that lands in SQLite but leaves a stale items key makes the reader walk a list the user
// cannot see any more. Running the identical script twice — once with no cache, once with one —
// and comparing turns "the cache went stale" from a support ticket into a failing test.
func TestReadListOrderSurvivesTheCache(t *testing.T) {
	script := func(repo ReadListRepository, ctx context.Context) [][]string {
		var snapshots [][]string
		record := func() {
			ids, err := repo.GetReadListBookIDs(ctx, "rl-1")
			if err != nil {
				t.Fatal(err)
			}
			snapshots = append(snapshots, ids)
		}

		if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Civil War", ""); err != nil {
			t.Fatal(err)
		}
		record()
		for i := 0; i < 4; i++ {
			if err := repo.AppendBookToReadList(ctx, "rl-1", fmt.Sprintf("bk-%d", i)); err != nil {
				t.Fatal(err)
			}
			record()
		}
		if err := repo.RemoveBookFromReadList(ctx, "rl-1", "bk-1"); err != nil {
			t.Fatal(err)
		}
		record()
		if err := repo.ReplaceReadListOrder(ctx, "rl-1", []string{"bk-3", "bk-0", "bk-2"}); err != nil {
			t.Fatal(err)
		}
		record()
		if err := repo.AppendBookToReadList(ctx, "rl-1", "bk-4"); err != nil {
			t.Fatal(err)
		}
		record()
		return snapshots
	}

	coldRepo, _, coldCtx := readListProbe(t, nil)
	cold := script(coldRepo, coldCtx)
	warmRepo, _, warmCtx := readListProbe(t, cache.NewRamCache())
	warm := script(warmRepo, warmCtx)

	if !reflect.DeepEqual(cold, warm) {
		t.Fatalf("the cached path diverged from the cold path:\ncold %v\nwarm %v", cold, warm)
	}
	final := cold[len(cold)-1]
	want := []string{"bk-3", "bk-0", "bk-2", "bk-4"}
	if !reflect.DeepEqual(final, want) {
		t.Errorf("final order = %v, want %v", final, want)
	}
}

// Append puts a book at MAX(position)+1 rather than at COUNT(*), so removing the middle of the list
// and appending again cannot land on a position that is still occupied.
func TestAppendAfterRemovalDoesNotCollide(t *testing.T) {
	repo, _, ctx := readListProbe(t, nil)
	if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Order", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := repo.AppendBookToReadList(ctx, "rl-1", fmt.Sprintf("bk-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.RemoveBookFromReadList(ctx, "rl-1", "bk-2"); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendBookToReadList(ctx, "rl-1", "bk-3"); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendBookToReadList(ctx, "rl-1", "bk-4"); err != nil {
		t.Fatal(err)
	}
	ids, err := repo.GetReadListBookIDs(ctx, "rl-1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bk-0", "bk-1", "bk-3", "bk-4"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("order = %v, want %v", ids, want)
	}
}

// Appending the same book twice is a double click, not an error, and it must not create a second
// row at a new position — the ON CONFLICT DO NOTHING is what keeps the list a set.
func TestAppendIsIdempotent(t *testing.T) {
	repo, _, ctx := readListProbe(t, cache.NewRamCache())
	if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Order", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := repo.AppendBookToReadList(ctx, "rl-1", "bk-0"); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := repo.GetReadListBookIDs(ctx, "rl-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("got %v, want one entry", ids)
	}
	counts, err := repo.CountBooksInReadLists(ctx, []string{"rl-1"})
	if err != nil {
		t.Fatal(err)
	}
	if counts["rl-1"] != 1 {
		t.Errorf("count = %d, want 1", counts["rl-1"])
	}
}

// The grouped count is keyed by the joined page of ids, so nothing can evict it by list name — the
// whole counts namespace has to go on every membership change or the list cards freeze at old sizes.
func TestCountBooksInReadListsFollowsMembership(t *testing.T) {
	repo, _, ctx := readListProbe(t, cache.NewRamCache())
	for _, id := range []string{"rl-1", "rl-2"} {
		if _, err := repo.CreateReadList(ctx, id, "u-1", id, ""); err != nil {
			t.Fatal(err)
		}
	}
	page := []string{"rl-1", "rl-2"}

	counts, err := repo.CountBooksInReadLists(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if counts["rl-1"] != 0 || counts["rl-2"] != 0 {
		t.Fatalf("empty lists counted %v, want zeroes for both", counts)
	}

	if err := repo.AppendBookToReadList(ctx, "rl-1", "bk-0"); err != nil {
		t.Fatal(err)
	}
	counts, err = repo.CountBooksInReadLists(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if counts["rl-1"] != 1 {
		t.Errorf("after append rl-1 = %d, want 1 — the counts cache outlived the write", counts["rl-1"])
	}

	if err := repo.RemoveBookFromReadList(ctx, "rl-1", "bk-0"); err != nil {
		t.Fatal(err)
	}
	counts, err = repo.CountBooksInReadLists(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if counts["rl-1"] != 0 {
		t.Errorf("after remove rl-1 = %d, want 0", counts["rl-1"])
	}
}

// GetReadListsByIDs answers in the order it was asked, not the order SQLite returns rows, and it has
// to do that on a partial cache hit too — the position-ordered list view depends on it.
func TestGetReadListsByIDsKeepsRequestedOrder(t *testing.T) {
	repo, _, ctx := readListProbe(t, cache.NewRamCache())
	for i := 0; i < 4; i++ {
		if _, err := repo.CreateReadList(ctx, fmt.Sprintf("rl-%d", i), "u-1", fmt.Sprintf("L%d", i), ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.GetReadListsByIDs(ctx, []string{"rl-2"}); err != nil {
		t.Fatal(err)
	}

	asked := []string{"rl-3", "rl-0", "rl-2", "rl-1", "rl-missing"}
	got, err := repo.GetReadListsByIDs(ctx, asked)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(got))
	for _, entity := range got {
		ids = append(ids, entity.ID)
	}
	want := []string{"rl-3", "rl-0", "rl-2", "rl-1"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("order = %v, want %v", ids, want)
	}
}

// Next walks the position column, skips archived books, and reports end-of-list as ErrNoRows so the
// reader can hide the button instead of showing an error.
func TestGetNextInReadListSkipsArchivedAndEnds(t *testing.T) {
	repo, db, ctx := readListProbe(t, nil)
	if _, err := repo.CreateReadList(ctx, "rl-1", "u-1", "Civil War", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if err := repo.AppendBookToReadList(ctx, "rl-1", fmt.Sprintf("bk-%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	first, err := repo.GetFirstInReadList(ctx, "rl-1")
	if err != nil {
		t.Fatal(err)
	}
	if first != "bk-0" {
		t.Errorf("first = %q, want bk-0", first)
	}

	next, err := repo.GetNextInReadList(ctx, "rl-1", "bk-0")
	if err != nil {
		t.Fatal(err)
	}
	if next != "bk-1" {
		t.Errorf("next after bk-0 = %q, want bk-1", next)
	}

	if _, err := db.Exec(`UPDATE books SET status='archived' WHERE id='bk-2'`); err != nil {
		t.Fatal(err)
	}
	next, err = repo.GetNextInReadList(ctx, "rl-1", "bk-1")
	if err != nil {
		t.Fatal(err)
	}
	if next != "bk-3" {
		t.Errorf("next after bk-1 = %q, want bk-3 — an archived book must not be handed to the reader", next)
	}

	if _, err := repo.GetNextInReadList(ctx, "rl-1", "bk-3"); err == nil {
		t.Error("walking past the last entry returned a book, want ErrNoRows")
	}
}
