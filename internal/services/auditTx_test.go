package services

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func auditDB(tb testing.TB) *sql.DB {
	tb.Helper()
	dsn := filepath.Join(tb.TempDir(), "audit.db") +
		"?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)" +
		"&_pragma=trusted_schema(OFF)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatal(err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)
	tb.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	return db
}

func auditBookService(tb testing.TB, db *sql.DB, dataDir string, reg bookparser.Registry) (BookService, repositories.BookDBRepository) {
	return auditBookServiceCache(tb, db, dataDir, reg, cache.NewRamCache())
}

func auditBookServiceCache(tb testing.TB, db *sql.DB, dataDir string, reg bookparser.Registry, ram cache.Cache) (BookService, repositories.BookDBRepository) {
	tb.Helper()
	bookRepo := repositories.NewBookDBRepository(db, ram)
	diskRepo, err := repositories.NewBookFileRepository(dataDir)
	if err != nil {
		tb.Fatal(err)
	}
	settingsRepo := repositories.NewSettingsRepository(db, ram)
	roleRepo := repositories.NewRoleRepository(db, ram)
	perms := NewPermissionCache(roleRepo)
	if err := perms.Reload(context.Background()); err != nil {
		tb.Fatal(err)
	}
	tx := database.NewTxManager(db)
	settings := NewSettingsService(settingsRepo, tx, perms)
	return NewBookService(bookRepo, nil, nil, diskRepo, reg, tx, settings, perms, nil, nil), bookRepo
}

// writeBigEPUB builds a spec-valid EPUB whose spine has `chapters` XHTML documents. ParseSpine
// re-opens the zip and linear-scans r.File once per spine item (getZipFile), so cost grows with
// the entry count -- exactly the shape of a real multi-hundred-chapter light novel.
func writeBigEPUB(tb testing.TB, path string, chapters int) {
	tb.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			tb.Fatal(err)
		}
	}

	add("META-INF/container.xml", `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`)

	var manifest, spine bytes.Buffer
	filler := bytes.Repeat([]byte("Lorem ipsum dolor sit amet consectetur. "), 300)
	for i := 0; i < chapters; i++ {
		id := fmt.Sprintf("ch%d", i)
		href := fmt.Sprintf("ch%d.xhtml", i)
		fmt.Fprintf(&manifest, `<item id="%s" href="%s" media-type="application/xhtml+xml"/>`, id, href)
		fmt.Fprintf(&spine, `<itemref idref="%s"/>`, id)
		add("OEBPS/"+href, fmt.Sprintf(
			`<?xml version="1.0" encoding="utf-8"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter %d</title></head><body><p>%s</p></body></html>`,
			i, filler))
	}

	add("OEBPS/content.opf", `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Audit Long Novel</dc:title><dc:creator>Audit Author</dc:creator><dc:language>en</dc:language><dc:identifier id="id">audit-1</dc:identifier></metadata><manifest>`+
		manifest.String()+`</manifest><spine>`+spine.String()+`</spine></package>`)

	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		tb.Fatal(err)
	}
}

// auditSeedBook copies the archive to a per-book path: book_files.path is UNIQUE.
func auditSeedBook(tb testing.TB, db *sql.DB, bookID, path, format string) {
	tb.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	path = filepath.Join(filepath.Dir(path), bookID+"."+format)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		tb.Fatal(err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO libraries (id, name) VALUES ('lib-audit','Audit')`); err != nil {
		tb.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES (?,?,?,'pending')`,
		bookID, "lib-audit", "Audit Book "+bookID); err != nil {
		tb.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		tb.Fatal(err)
	}

	if _, err := db.Exec(
		`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time, state) VALUES (?,?,?,?,?,?,'managed')`,
		"file-"+bookID, bookID, path, format, info.Size(), info.ModTime()); err != nil {
		tb.Fatal(err)
	}
}

// writeLockProbe repeatedly attempts a real IMMEDIATE write transaction and reports the longest
// stretch during which every attempt was blocked. Because _txlock=immediate takes the write lock
// at BEGIN, a probe that cannot begin is proof another writer holds the lock.
type writeLockProbe struct {
	failed      atomic.Int64
	lastErr     atomic.Value
	blockedFor  atomic.Int64
	attempts    atomic.Int64
	blockedOnce atomic.Int64
}

func (p *writeLockProbe) run(ctx context.Context, db *sql.DB) {
	// Short busy_timeout on the probe only: we want to observe contention, not wait it out.
	var firstBlocked time.Time
	for ctx.Err() == nil {
		p.attempts.Add(1)
		start := time.Now()
		tx, err := db.BeginTx(ctx, nil)
		if err == nil {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO app_settings (key, value_json) VALUES ('audit_probe','1')
				 ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json`)
			if cerr := tx.Commit(); err == nil {
				err = cerr
			}
		}
		waited := time.Since(start)
		if err != nil && ctx.Err() == nil {
			p.failed.Add(1)
			p.lastErr.Store(err.Error())
		}
		if waited > 50*time.Millisecond {
			p.blockedOnce.Add(1)
			if firstBlocked.IsZero() {
				firstBlocked = start
			}
			if cur := int64(waited); cur > p.blockedFor.Load() {
				p.blockedFor.Store(cur)
			}
		} else {
			firstBlocked = time.Time{}
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// ExtractMetadata must not let the cache size leak into how long it holds the write lock. ParseSpine
// runs before BeginTx, and the per-row invalidations are buffered by the deferred cache that WithTx
// installs, then published once by FlushCache after Commit. Regressing either one puts a full-cache
// Range() back inside the transaction, once per chapter.
func TestAuditExtractMetadataHoldsWriteLockDuringArchiveParse(t *testing.T) {
	cases := []struct {
		name         string
		chapters     int
		cacheEntries int
	}{
		{"600_chapters_cold_cache", 600, 0},
		{"600_chapters_warm_cache_20k", 600, 20000},
		{"600_chapters_warm_cache_250k", 600, 250000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := auditDB(t)
			dataDir := t.TempDir()
			bookPath := filepath.Join(dataDir, "audit.epub")
			writeBigEPUB(t, bookPath, tc.chapters)

			// A cache warmed by ordinary browsing traffic. Nothing here relates to the
			// book being imported -- these are other books' cached rows.
			ram := cache.NewRamCache()
			for i := 0; i < tc.cacheEntries; i++ {
				_ = ram.Set(context.Background(), fmt.Sprintf("book:id:other-%d", i),
					[]byte("cached-book-payload-cached-book-payload"), time.Hour)
			}

			reg := bookparser.NewRegistry()
			reg.Register(epub.NewParser(), "epub", "kepub.epub")

			svc, _ := auditBookServiceCache(t, db, dataDir, reg, ram)
			auditSeedBook(t, db, "book-audit", bookPath, "epub")

			// Baseline: what the parse costs with no transaction involved.
			parser, err := reg.Parser("epub", bookPath)
			if err != nil {
				t.Fatal(err)
			}
			parseStart := time.Now()
			spine, err := parser.ParseSpine(bookPath)
			if err != nil {
				t.Fatal(err)
			}
			parseCost := time.Since(parseStart)

			probe := &writeLockProbe{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				probe.run(ctx, db)
				close(done)
			}()
			time.Sleep(100 * time.Millisecond) // let the probe establish a clean baseline

			extractStart := time.Now()
			if err := svc.ExtractMetadata(context.Background(), "book-audit"); err != nil {
				cancel()
				<-done
				t.Fatal(err)
			}
			total := time.Since(extractStart)

			cancel()
			<-done

			blocked := time.Duration(probe.blockedFor.Load())
			if n := probe.failed.Load(); n > 0 {
				le, _ := probe.lastErr.Load().(string)
				t.Errorf("%d/%d concurrent writes FAILED OUTRIGHT (not merely delayed): %s", n, probe.attempts.Load(), le)
			}
			info, _ := os.Stat(bookPath)
			t.Logf("archive %.1f MB, %d spine items, %d unrelated cache entries",
				float64(info.Size())/(1024*1024), len(spine), tc.cacheEntries)
			t.Logf("ParseSpine alone (no tx):             %v", parseCost.Round(time.Millisecond))
			t.Logf("ExtractMetadata total:                %v", total.Round(time.Millisecond))
			t.Logf("LONGEST WRITE-LOCK STALL for any other writer: %v (%d/%d probe attempts stalled)",
				blocked.Round(time.Millisecond), probe.blockedOnce.Load(), probe.attempts.Load())

			if blocked > 500*time.Millisecond {
				t.Errorf("REGRESSION: write-lock stall %v scales with the %d unrelated cache entries; "+
					"invalidation belongs after Commit, not once per row inside the tx", blocked, tc.cacheEntries)
			}
		})
	}
}

// TestAuditExtractMetadataBlocksConcurrentWriters shows the user-visible effect: a second
// request that only needs to write one row waits behind the archive parse.
func TestAuditExtractMetadataBlocksConcurrentWriters(t *testing.T) {
	db := auditDB(t)
	dataDir := t.TempDir()
	bookPath := filepath.Join(dataDir, "audit.epub")
	writeBigEPUB(t, bookPath, 800)

	reg := bookparser.NewRegistry()
	reg.Register(epub.NewParser(), "epub", "kepub.epub")
	svc, _ := auditBookService(t, db, dataDir, reg)
	auditSeedBook(t, db, "book-blocked", bookPath, "epub")
	auditSeedBook(t, db, "book-victim", bookPath, "epub")

	var wg sync.WaitGroup
	var victimWait time.Duration

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Give ExtractMetadata a head start so it is already inside its tx and parsing.
		time.Sleep(150 * time.Millisecond)
		start := time.Now()
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = tx.Exec(`UPDATE books SET title = 'victim' WHERE id = 'book-victim'`)
		if err != nil {
			_ = tx.Rollback()
			t.Error(err)
			return
		}
		if err := tx.Commit(); err != nil {
			t.Error(err)
			return
		}
		victimWait = time.Since(start)
	}()

	if err := svc.ExtractMetadata(context.Background(), "book-blocked"); err != nil {
		t.Fatal(err)
	}
	wg.Wait()

	t.Logf("a one-row UPDATE issued while ExtractMetadata parsed took %v to commit",
		victimWait.Round(time.Millisecond))
	if victimWait > 500*time.Millisecond {
		t.Errorf("REGRESSION: a one-row UPDATE waited %v behind the import", victimWait)
	}
}

// TestAuditExtractMetadataConnectionsReturnToBaseline guards against a leaked connection or a
// leaked *sql.Tx on the same path (a leaked immediate tx would hold the write lock forever).
func TestAuditExtractMetadataConnectionsReturnToBaseline(t *testing.T) {
	db := auditDB(t)
	dataDir := t.TempDir()
	bookPath := filepath.Join(dataDir, "audit.epub")
	writeBigEPUB(t, bookPath, 40)

	reg := bookparser.NewRegistry()
	reg.Register(epub.NewParser(), "epub", "kepub.epub")
	svc, _ := auditBookService(t, db, dataDir, reg)

	for i := 0; i < 30; i++ {
		auditSeedBook(t, db, fmt.Sprintf("leak-%02d", i), bookPath, "epub")
	}

	base := db.Stats().InUse
	for i := 0; i < 30; i++ {
		if err := svc.ExtractMetadata(context.Background(), fmt.Sprintf("leak-%02d", i)); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
	after := db.Stats()
	t.Logf("InUse baseline=%d after=%d open=%d", base, after.InUse, after.OpenConnections)
	if after.InUse > base {
		t.Errorf("connection leak: InUse went from %d to %d", base, after.InUse)
	}

	// A held write lock would make this fail rather than return promptly.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("write lock still held after ExtractMetadata returned: %v", err)
	}
	_ = tx.Rollback()
}

// TestAuditParseSpineDataRace runs the exact ExtractMetadata path concurrently under -race to
// look for shared-state races in the service/repository/cache layers it touches.
func TestAuditParseSpineDataRace(t *testing.T) {
	db := auditDB(t)
	dataDir := t.TempDir()
	bookPath := filepath.Join(dataDir, "audit.epub")
	writeBigEPUB(t, bookPath, 20)

	reg := bookparser.NewRegistry()
	reg.Register(epub.NewParser(), "epub", "kepub.epub")
	svc, repo := auditBookService(t, db, dataDir, reg)

	const n = 12
	for i := 0; i < n; i++ {
		auditSeedBook(t, db, fmt.Sprintf("race-%02d", i), bookPath, "epub")
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("race-%02d", i)
			if err := svc.ExtractMetadata(context.Background(), id); err != nil {
				t.Errorf("%s: %v", id, err)
				return
			}
			if _, err := repo.GetBook(context.Background(), id); err != nil {
				t.Errorf("%s get: %v", id, err)
			}
		}(i)
	}
	wg.Wait()
}
