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

// The by-IDs readers sort their id list to build a stable singleflight key. When every id is a
// cache miss the "missing" slice aliases the caller's, so sorting it in place reorders the
// caller's page: SearchBooks hands over keyset order (created_at DESC) and gets id-ascending
// back, which makes StreamLibraryZip's cursor walk one book per page instead of one hundred.
func TestGetByIDsPreservesCallerOrder(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "order.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','lib-1')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := db.Exec(`INSERT INTO books (id,library_id,title,status) VALUES (?,'lib-1',?,'ready')`,
			fmt.Sprintf("b-%d", i), fmt.Sprintf("t-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewBookDBRepository(db, nil)

	asked := []string{"b-4", "b-2", "b-0", "b-3", "b-1"}
	input := append([]string(nil), asked...)
	books, err := repo.GetBooksByIDs(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range asked {
		if input[i] != id {
			t.Fatalf("caller slice was mutated: asked %v, now %v", asked, input)
		}
	}
	if len(books) != len(asked) {
		t.Fatalf("got %d books, want %d", len(books), len(asked))
	}
	for i, b := range books {
		if b.ID != asked[i] {
			t.Fatalf("result %d is %s, want %s; the requested order was not preserved", i, b.ID, asked[i])
		}
	}
}
