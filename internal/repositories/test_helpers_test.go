package repositories

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/database"
)

func probeDB(tb testing.TB) *sql.DB {
	tb.Helper()
	dsn := filepath.Join(tb.TempDir(), "probe.db") +
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
	return db
}

func facetTime(tb testing.TB, runs int, fn func() error) time.Duration {
	tb.Helper()
	if err := fn(); err != nil {
		tb.Fatal(err)
	}
	start := time.Now()
	for i := 0; i < runs; i++ {
		if err := fn(); err != nil {
			tb.Fatal(err)
		}
	}
	return time.Since(start) / time.Duration(runs)
}
