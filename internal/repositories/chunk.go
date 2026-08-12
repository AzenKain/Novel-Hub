package repositories

import (
	"database/sql"
	"slices"
	"time"
)

const sqliteTimeLayout = "2006-01-02 15:04:05"

func cursorTimeArg(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(sqliteTimeLayout), Valid: true}
}

const sqliteMaxSliceArgs = 8000

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
