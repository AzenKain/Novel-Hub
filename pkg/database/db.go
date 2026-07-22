package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"novelhub/pkg/config"
)

func defaultMaxOpenConns() int {
	conns := runtime.NumCPU() * 2
	if conns < 4 {
		return 4
	}
	if conns > 16 {
		return 16
	}
	return conns
}

func ApplySchema(db *sql.DB, schemaDir string) error {
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	for _, file := range files {
		var count int
		err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, file).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		path := filepath.Join(schemaDir, file)
		schema, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if _, err := db.ExecContext(context.Background(), string(schema)); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				log.Printf("Warning: duplicate column in %s, marking as applied", file)
			} else if strings.Contains(err.Error(), "already exists") {
				log.Printf("Warning: object already exists in %s, marking as applied", file)
			} else {
				return fmt.Errorf("apply schema %s: %w", path, err)
			}
		}

		if _, err := db.ExecContext(context.Background(), `INSERT INTO schema_migrations (version) VALUES (?)`, file); err != nil {
			return fmt.Errorf("record migration %s: %w", file, err)
		}
	}
	return nil
}



func autoSQLiteCacheKB() int {
	if configured := config.GetIntConfigWithDefault("SQLITE_CACHE_SIZE_KB", 0); configured > 0 {
		return configured
	}
	memBytes := systemMemoryBytes()
	if memBytes >= 32<<30 { // >= 32GB RAM
		return 1048576 // 1GB SQLite cache per connection
	}
	if memBytes >= 16<<30 { // >= 16GB RAM
		return 512000 // 512MB SQLite cache
	}
	if memBytes >= 8<<30 { // >= 8GB RAM
		return 256000 // 256MB SQLite cache
	}
	return 128000 // 128MB default
}

func autoSQLiteMmapSize() int64 {
	if configured := config.GetIntConfigWithDefault("SQLITE_MMAP_SIZE_BYTES", 0); configured > 0 {
		return int64(configured)
	}
	memBytes := systemMemoryBytes()
	if memBytes >= 32<<30 {
		return 16106127360 // 15GB Memory Mapped DB
	}
	if memBytes >= 16<<30 {
		return 8589934592 // 8GB Memory Mapped DB
	}
	if memBytes >= 8<<30 {
		return 4294967296 // 4GB Memory Mapped DB
	}
	return 1073741824 // 1GB default
}

func systemMemoryBytes() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "MemTotal:" {
			continue
		}
		kib, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return kib * 1024
	}
	return 0
}

func NewSQLiteDB() (*sql.DB, error) {
	dbPath := config.GetConfigWithDefault("SQLITE_DB_PATH", "./data/novelhub.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); err == nil {
		_ = os.Chmod(dbPath, 0600)
	}

	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	if err != nil {
		return nil, err
	}

	cacheKB := autoSQLiteCacheKB()
	mmapBytes := autoSQLiteMmapSize()

	pragmas := fmt.Sprintf("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA trusted_schema=OFF; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=10000; PRAGMA temp_store=MEMORY; PRAGMA cache_size=-%d; PRAGMA mmap_size=%d;", cacheKB, mmapBytes)
	if _, err := db.Exec(pragmas); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to apply SQLite pragmas: %w", err)
	}

	maxOpen := config.GetIntConfigWithDefault("SQLITE_MAX_OPEN_CONNS", defaultMaxOpenConns())
	if maxOpen < 1 {
		maxOpen = defaultMaxOpenConns()
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(config.GetIntConfigWithDefault("SQLITE_MAX_IDLE_CONNS", maxOpen))

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func sqliteDSN(dbPath string) string {
	cacheKB := autoSQLiteCacheKB()
	mmapBytes := autoSQLiteMmapSize()

	values := url.Values{}
	values.Add("_pragma", "busy_timeout=10000")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "trusted_schema(OFF)")
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "synchronous(NORMAL)")
	values.Add("_pragma", "temp_store(MEMORY)")
	values.Add("_pragma", fmt.Sprintf("cache_size(-%d)", cacheKB))
	values.Add("_pragma", fmt.Sprintf("mmap_size(%d)", mmapBytes))

	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}
	return dbPath + separator + values.Encode()
}

var globalWriteMutex sync.Mutex

func BeginImmediateTx(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	globalWriteMutex.Lock()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		globalWriteMutex.Unlock()
		return nil, err
	}
	return tx, nil
}

func ReleaseWriteLock() {
	globalWriteMutex.Unlock()
}
