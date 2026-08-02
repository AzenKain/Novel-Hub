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

func autoSQLiteCacheKB(maxOpen int) int {
	totalKB := config.GetIntConfigWithDefault("SQLITE_CACHE_SIZE_KB", 0)
	if totalKB <= 0 {
		totalKB = int(systemMemoryBytes() / 64 / 1024)
		if totalKB < 64*1024 {
			totalKB = 64 * 1024
		}
		if totalKB > 512*1024 {
			totalKB = 512 * 1024
		}
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	perConnection := totalKB / maxOpen
	if perConnection < 4*1024 {
		perConnection = 4 * 1024
	}
	return perConnection
}

func autoSQLiteMmapSize() int64 {
	if configured := config.GetIntConfigWithDefault("SQLITE_MMAP_SIZE_BYTES", 0); configured > 0 {
		return int64(configured)
	}
	mmap := systemMemoryBytes() / 8
	if mmap < 256<<20 {
		mmap = 256 << 20
	}
	if mmap > 2<<30 {
		mmap = 2 << 30
	}
	return mmap
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
	// Derived from DATA_DIR so the database lands beside books, logs and backups
	// by default. A hardcoded "./data" would write outside the mounted volume in
	// Docker, where DATA_DIR is /data — losing the database on every restart.
	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	dbPath := config.GetConfigWithDefault("SQLITE_DB_PATH", filepath.Join(dataDir, "novelhub.db"))
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); err == nil {
		_ = os.Chmod(dbPath, 0600)
	}

	maxOpen := config.GetIntConfigWithDefault("SQLITE_MAX_OPEN_CONNS", defaultMaxOpenConns())
	if maxOpen < 1 {
		maxOpen = defaultMaxOpenConns()
	}
	cacheKB := autoSQLiteCacheKB(maxOpen)
	mmapBytes := autoSQLiteMmapSize()
	db, err := sql.Open("sqlite", sqliteDSN(dbPath, cacheKB, mmapBytes))
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(config.GetIntConfigWithDefault("SQLITE_MAX_IDLE_CONNS", maxOpen))

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func sqliteDSN(dbPath string, cacheKB int, mmapBytes int64) string {
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
