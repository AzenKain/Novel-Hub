package repositories

import (
	"database/sql"
	"slices"
	"time"
)

// cursorTimeArg binds a pagination cursor timestamp as a **string in the column's own
// format**, "YYYY-MM-DD HH:MM:SS". The cursor predicates compare the bare column so SQLite
// can seek the created_at index; that only works if both sides are the same text format.
// RFC3339 would compare 'T' against ' ' and silently return page 1 forever instead of
// erroring. modernc.org/sqlite also hands time.Time over as a non-text affinity, which
// compares as less than any string. An invalid NullString → SQLite NULL → the
// `sqlc.narg('cursor_created_at') IS NULL` first-page branch.
const sqliteTimeLayout = "2006-01-02 15:04:05"

func cursorTimeArg(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(sqliteTimeLayout), Valid: true}
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
