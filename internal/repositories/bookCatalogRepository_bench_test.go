package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/convert"
	"novelhub/pkg/database"
)

func seedBookCatalogBench(tb testing.TB, booksCount int) *sql.DB {
	tb.Helper()
	dsn := filepath.Join(tb.TempDir(), "bench_catalog.db") +
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

	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}

	insBook, err := tx.Prepare(`INSERT INTO books (id, library_id, title, status, created_at) VALUES (?, ?, ?, 'active', ?)`)
	if err != nil {
		tb.Fatal(err)
	}
	defer insBook.Close()

	insSeries, err := tx.Prepare(`INSERT INTO series (id, name) VALUES (?, ?)`)
	if err != nil {
		tb.Fatal(err)
	}
	defer insSeries.Close()

	insBookSeries, err := tx.Prepare(`INSERT INTO book_series (book_id, series_id, series_index) VALUES (?, ?, ?)`)
	if err != nil {
		tb.Fatal(err)
	}
	defer insBookSeries.Close()

	// Insert 200 series
	seriesCount := 200
	for s := 0; s < seriesCount; s++ {
		sid := fmt.Sprintf("s-%05d", s)
		sname := fmt.Sprintf("Series %05d", s)
		if _, err := insSeries.Exec(sid, sname); err != nil {
			tb.Fatal(err)
		}
	}

	// Insert books
	for i := 0; i < booksCount; i++ {
		bid := fmt.Sprintf("b-%06d", i)
		title := fmt.Sprintf("Book Title %06d", i)
		createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second).Format("2006-01-02 15:04:05")

		if _, err := insBook.Exec(bid, "lib-1", title, createdAt); err != nil {
			tb.Fatal(err)
		}

		// Link 80% of books to series (25 books per series on average)
		if i < int(float64(booksCount)*0.8) {
			sIdx := (i / 20) % seriesCount
			sid := fmt.Sprintf("s-%05d", sIdx)
			seriesIndex := fmt.Sprintf("%d", (i%20)+1)
			if _, err := insBookSeries.Exec(bid, sid, seriesIndex); err != nil {
				tb.Fatal(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}

	return db
}

func BenchmarkSearchBooks(b *testing.B) {
	db := seedBookCatalogBench(b, 5000)
	repo := NewBookDBRepository(db, cache.NewRamCache())
	ctx := context.Background()
	lib := "lib-1"

	// 1. Recently Added
	b.Run("RecentlyAdded_FirstPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "recently_added", "", 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	booksRA, _ := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "recently_added", "", 20)
	cursorRA := ""
	if len(booksRA) > 0 {
		last := booksRA[len(booksRA)-1]
		cursorRA = last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID
	}

	b.Run("RecentlyAdded_NextPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "recently_added", cursorRA, 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// 2. Title A-Z
	b.Run("TitleAZ_FirstPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "title_az", "", 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	booksTitle, _ := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "title_az", "", 20)
	cursorTitle := ""
	if len(booksTitle) > 0 {
		last := booksTitle[len(booksTitle)-1]
		cursorTitle = convert.EncodeCursor(last.Title, last.ID)
	}

	b.Run("TitleAZ_NextPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "title_az", cursorTitle, 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// 3. Series Order
	b.Run("SeriesOrder_FirstPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "series_order", "", 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	booksSeries, _ := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "series_order", "", 20)
	cursorSeries := ""
	if len(booksSeries) > 0 {
		last := booksSeries[len(booksSeries)-1]
		seriesList, _ := repo.GetBookSeries(ctx, last.ID)
		var seriesName string
		var seriesIndex string
		if len(seriesList) > 0 {
			seriesName = seriesList[0].SeriesName
			if seriesList[0].SeriesIndex != nil {
				seriesIndex = *seriesList[0].SeriesIndex
			}
		}
		cursorSeries = convert.EncodeCursor(seriesName+"|"+seriesIndex, last.ID)
	}

	b.Run("SeriesOrder_NextPage", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "series_order", cursorSeries, 20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
