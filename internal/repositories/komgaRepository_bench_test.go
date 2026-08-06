package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func seedKomgaBench(tb testing.TB, seriesCount, booksPerSeries int) *sql.DB {
	tb.Helper()
	// Same pragmas and pool ceiling as pkg/database.NewSQLiteDB: a bare sql.Open lets every
	// goroutine take its own connection with no WAL, so a concurrency test on it measures
	// connection contention that production never has.
	dsn := filepath.Join(tb.TempDir(), "bench.db") +
		"?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)" +
		"&_pragma=trusted_schema(OFF)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" +
		"&_pragma=temp_store(MEMORY)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatal(err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)
	tb.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO libraries (id,name) VALUES ('lib-1','L')`); err != nil {
		tb.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id,email,password_hash) VALUES ('u-1','a@b.c','x')`); err != nil {
		tb.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	stmts := make([]*sql.Stmt, 0, 4)
	prep := func(query string) *sql.Stmt {
		st, err := tx.Prepare(query)
		if err != nil {
			tb.Fatal(err)
		}
		stmts = append(stmts, st)
		return st
	}
	insSeries := prep(`INSERT INTO series (id,name) VALUES (?,?)`)
	insBook := prep(`INSERT INTO books (id,library_id,title,status) VALUES (?,?,?,'active')`)
	insLink := prep(`INSERT INTO book_series (book_id,series_id,series_index) VALUES (?,?,?)`)
	insProgress := prep(`INSERT INTO reading_progress (user_id,book_id,chapter_ref,progress_percent) VALUES ('u-1',?,'c',100)`)

	for s := 0; s < seriesCount; s++ {
		sid := fmt.Sprintf("s-%05d", s)
		if _, err := insSeries.Exec(sid, fmt.Sprintf("Series %05d", s)); err != nil {
			tb.Fatal(err)
		}
		for k := 0; k < booksPerSeries; k++ {
			bid := fmt.Sprintf("b-%05d-%03d", s, k)
			if _, err := insBook.Exec(bid, "lib-1", bid); err != nil {
				tb.Fatal(err)
			}
			if _, err := insLink.Exec(bid, sid, fmt.Sprint(k+1)); err != nil {
				tb.Fatal(err)
			}
			if k%2 == 0 {
				if _, err := insProgress.Exec(bid); err != nil {
					tb.Fatal(err)
				}
			}
		}
	}
	for _, st := range stmts {
		_ = st.Close()
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	return db
}

// One /api/v1/series page as the service builds it: list the page, then one progress query per
// series. The per-series call is what N+1 costs.
func BenchmarkKomgaSeriesPage(b *testing.B) {
	for _, size := range []int64{20, 100} {
		db := seedKomgaBench(b, 500, 10)
		repo := NewKomgaRepository(db, cache.NewRamCache())
		ctx := context.Background()
		libs := []string{"lib-1"}

		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				rows, err := repo.ListSeries(ctx, libs, "", size, 0)
				if err != nil {
					b.Fatal(err)
				}
				ids := make([]string, len(rows))
				for i, row := range rows {
					ids[i] = row.ID
				}
				if _, err := repo.SeriesProgressByIDs(ctx, "u-1", ids, libs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkKomgaSeriesProgressSingle(b *testing.B) {
	db := seedKomgaBench(b, 500, 10)
	repo := NewKomgaRepository(db, cache.NewRamCache())
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := repo.SeriesProgress(ctx, "u-1", "s-00250", []string{"lib-1"}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKomgaListSeriesOnly(b *testing.B) {
	db := seedKomgaBench(b, 500, 10)
	repo := NewKomgaRepository(db, cache.NewRamCache())
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := repo.ListSeries(ctx, []string{"lib-1"}, "", 20, 0); err != nil {
			b.Fatal(err)
		}
	}
}
