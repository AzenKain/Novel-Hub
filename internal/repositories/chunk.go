package repositories

import (
	"slices"
	"time"
)

// cursorTimeArg binds a pagination cursor timestamp as an RFC3339 **string**, not a
// sql.NullTime. modernc.org/sqlite hands sql.NullTime/time.Time to SQLite as a non-text
// affinity value, so `datetime(?)` returns NULL — the cursor predicate then never matches
// and every page after the first comes back empty. A text timestamp is what datetime(text)
// expects, so the `<`/`=` comparisons work. nil interface → SQLite NULL → the
// `sqlc.narg('cursor_created_at') IS NULL` first-page branch.
func cursorTimeArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// sqliteMaxSliceArgs bounds how many ids go into one `id IN (?,?,...)` query.
//
// sqlc expands sqlc.slice() into one bound parameter per element, and SQLite rejects a
// statement past SQLITE_MAX_VARIABLE_NUMBER with "too many SQL variables" — so a
// whole-library read (libraryService.StreamLibraryZip asks for a million) failed outright
// instead of returning rows. modernc.org/sqlite refuses at 32767 bound parameters;
// 8000 leaves room for the non-slice params these queries also bind while keeping the
// number of round trips low.
const sqliteMaxSliceArgs = 8000

// queryInChunks runs fn over ids in sqliteMaxSliceArgs-sized batches and concatenates the
// rows. Callers index results by id, so batch order does not matter.
//
// ponytail: caller-side batching, not a query rewrite — the ORDER BY inside these queries
// is per-batch, so only use this where the caller re-orders by the id list it passed in.
func queryInChunks[T any](ids []string, fn func([]string) ([]T, error)) ([]T, error) {
	if len(ids) <= sqliteMaxSliceArgs {
		return fn(ids)
	}
	out := make([]T, 0, len(ids))
	for chunk := range slices.Chunk(ids, sqliteMaxSliceArgs) {
		rows, err := fn(chunk)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	return out, nil
}
