package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/database"
)

// sqlc.Prepare is never called in this codebase, so every generated query runs with
// stmt == nil and lands on the `default:` branch of exec/query/queryRow — which uses
// q.db. That is only correct because WithTx sets db and tx to the same *sql.Tx. This
// test pins that invariant: if WithTx ever stops assigning db, writes would silently
// escape to the connection pool and commit even when the caller rolls back.
//
// BulkDeleteBooks is the case worth pinning: it is a sqlc.slice query (dynamic SQL,
// never preparable) and it is called through WithTx from bookService_bulk.go.
func TestWithTxRoutesSliceQueriesThroughTransaction(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	seed := func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM books`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO libraries (id, name) VALUES ('lib-tx', 'Lib')`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO books (id, library_id, title) VALUES ('b1','lib-tx','B1'),('b2','lib-tx','B2')`); err != nil {
			t.Fatal(err)
		}
	}
	countBooks := func() int {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("commit applies the delete", func(t *testing.T) {
		seed()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := sqlc.New(db).WithTx(tx).BulkDeleteBooks(ctx, []string{"b1", "b2"}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if n := countBooks(); n != 0 {
			t.Errorf("%d books remain after commit, want 0", n)
		}
	})

	t.Run("rollback discards the delete", func(t *testing.T) {
		seed()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := sqlc.New(db).WithTx(tx).BulkDeleteBooks(ctx, []string{"b1", "b2"}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		// A delete that leaked onto the pool would have committed on its own.
		if n := countBooks(); n != 2 {
			t.Errorf("%d books remain after rollback, want 2 — the delete escaped the transaction", n)
		}
	})
}
