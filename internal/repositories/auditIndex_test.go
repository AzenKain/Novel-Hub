package repositories

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/database"
)

func auditIndexDB(tb testing.TB) *sql.DB {
	tb.Helper()
	dsn := filepath.Join(tb.TempDir(), "audit.db") +
		"?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)" +
		"&_pragma=trusted_schema(OFF)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" +
		"&_pragma=temp_store(MEMORY)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	tb.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	return db
}

func auditIndexPlan(tb testing.TB, db *sql.DB, query string, args ...any) string {
	tb.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		tb.Fatalf("EQP failed for %q: %v", query, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			tb.Fatal(err)
		}
		out = append(out, detail)
	}
	if err := rows.Err(); err != nil {
		tb.Fatal(err)
	}
	return strings.Join(out, "\n")
}

func auditIndexExec(tb testing.TB, db *sql.DB, q string, args ...any) {
	tb.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		tb.Fatalf("%s: %v", q, err)
	}
}

// ---------------------------------------------------------------------------
// 1. Redundant / duplicate index inventory, read straight out of sqlite_master.
// ---------------------------------------------------------------------------

type auditIndexInfo struct {
	name    string
	table   string
	cols    string
	partial bool
	unique  bool
	origin  string
}

func auditIndexInventory(tb testing.TB, db *sql.DB) []auditIndexInfo {
	tb.Helper()
	tables := []string{}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'fts_%' ORDER BY name`)
	if err != nil {
		tb.Fatal(err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			tb.Fatal(err)
		}
		tables = append(tables, n)
	}
	rows.Close()

	var all []auditIndexInfo
	for _, t := range tables {
		il, err := db.Query(fmt.Sprintf("PRAGMA index_list(%q)", t))
		if err != nil {
			continue
		}
		type li struct {
			name    string
			unique  int
			origin  string
			partial int
		}
		var lis []li
		for il.Next() {
			var seq int
			var x li
			if err := il.Scan(&seq, &x.name, &x.unique, &x.origin, &x.partial); err != nil {
				tb.Fatal(err)
			}
			lis = append(lis, x)
		}
		il.Close()
		for _, x := range lis {
			ic, err := db.Query(fmt.Sprintf("PRAGMA index_xinfo(%q)", x.name))
			if err != nil {
				continue
			}
			var cols []string
			for ic.Next() {
				var seqno int
				var cid int
				var name sql.NullString
				var desc, coll, key int
				var collName sql.NullString
				if err := ic.Scan(&seqno, &cid, &name, &desc, &collName, &key); err != nil {
					ic.Close()
					tb.Fatal(err)
				}
				_ = coll
				if key == 1 && name.Valid {
					d := "ASC"
					if desc == 1 {
						d = "DESC"
					}
					cols = append(cols, name.String+" "+d)
				}
			}
			ic.Close()
			all = append(all, auditIndexInfo{
				name: x.name, table: t, cols: strings.Join(cols, ", "),
				partial: x.partial == 1, unique: x.unique == 1, origin: x.origin,
			})
		}
	}
	return all
}

func TestAuditIndexRedundant(t *testing.T) {
	db := auditIndexDB(t)
	inv := auditIndexInventory(t, db)

	// Exact duplicates: same table, identical key column list, both non-partial.
	byKey := map[string][]auditIndexInfo{}
	for _, ix := range inv {
		if ix.partial {
			continue
		}
		byKey[ix.table+" | "+ix.cols] = append(byKey[ix.table+" | "+ix.cols], ix)
	}
	var dupKeys []string
	for k, v := range byKey {
		if len(v) > 1 {
			dupKeys = append(dupKeys, k)
		}
	}
	sort.Strings(dupKeys)
	t.Logf("=== EXACT DUPLICATE INDEXES (identical key columns, same table) ===")
	for _, k := range dupKeys {
		var names []string
		for _, ix := range byKey[k] {
			tag := ix.name
			if ix.origin != "c" {
				tag += " [" + ix.origin + "]"
			}
			names = append(names, tag)
		}
		sort.Strings(names)
		t.Logf("  %s  ->  %s", k, strings.Join(names, " + "))
	}

	// Prefix redundancy: index A is redundant if another index B on the same table has A's
	// key columns as a strict leading prefix.
	t.Logf("=== PREFIX-REDUNDANT INDEXES (A's columns are a leading prefix of B's) ===")
	for _, a := range inv {
		if a.partial || a.unique {
			continue
		}
		ac := strings.Split(a.cols, ", ")
		for _, b := range inv {
			if b.name == a.name || b.table != a.table || b.partial {
				continue
			}
			bc := strings.Split(b.cols, ", ")
			if len(bc) <= len(ac) {
				continue
			}
			match := true
			for i := range ac {
				// direction does not matter for prefix usability in SQLite
				if strings.Fields(ac[i])[0] != strings.Fields(bc[i])[0] {
					match = false
					break
				}
			}
			if match {
				tag := b.name
				if b.origin != "c" {
					tag += " [" + b.origin + "]"
				}
				t.Logf("  %s(%s) redundant -- covered by %s(%s)", a.name, a.cols, tag, b.cols)
			}
		}
	}

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index'`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	t.Logf("total indexes present after ApplySchema: %d", total)
}

// ---------------------------------------------------------------------------
// 2. FK child columns with no usable index -> DELETE FROM books scans them.
// ---------------------------------------------------------------------------

// auditIndexSeedCascade builds a library whose child tables are large but where the book being deleted
// owns *zero* child rows. Any time spent is therefore pure "find the children" scan cost.
//
// Two details decide whether this measures reality or an artifact:
//   - users: the child tables are keyed (user_id, book_id). With ONE user that prefix is
//     degenerate and a scan of it looks like a seek. Real instances have many readers.
//   - analyze: production never runs ANALYZE (no call anywhere in pkg/database or db/schema),
//     and SQLite's skip-scan optimisation only activates with sqlite_stat1 present. Seeding
//     with ANALYZE therefore grants the planner a rescue production does not have.
func auditIndexSeedCascade(tb testing.TB, childRows, users int, analyze bool) *sql.DB {
	tb.Helper()
	db := auditIndexDB(tb)
	auditIndexExec(tb, db, `INSERT INTO libraries (id,name) VALUES ('lib-1','L')`)

	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	prep := func(q string) *sql.Stmt {
		st, err := tx.Prepare(q)
		if err != nil {
			tb.Fatalf("%s: %v", q, err)
		}
		return st
	}
	insUser := prep(`INSERT INTO users (id,email,password_hash) VALUES (?,?,'x')`)
	for u := 0; u < users; u++ {
		if _, err := insUser.Exec(fmt.Sprintf("u-%05d", u), fmt.Sprintf("u%05d@b.c", u)); err != nil {
			tb.Fatal(err)
		}
	}
	insBook := prep(`INSERT INTO books (id,library_id,title,status) VALUES (?,?,?,'active')`)
	insHl := prep(`INSERT INTO highlights (id,user_id,book_id,chapter_id,text_content,start_index,end_index) VALUES (?,?,?,?,'t',0,1)`)
	insSess := prep(`INSERT INTO reading_sessions (id,user_id,book_id,session_date) VALUES (?,?,?,date('now',?))`)
	insKobo := prep(`INSERT INTO kobo_synced_books (user_id,book_id) VALUES (?,?)`)
	insColl := prep(`INSERT INTO collection_books (collection_id,book_id) VALUES ('c-1',?)`)
	insRP := prep(`INSERT INTO reading_progress (user_id,book_id,file_id,chapter_ref) VALUES (?,?,?,'c')`)

	// books that OWN the child rows
	for i := 0; i < childRows; i++ {
		bid := fmt.Sprintf("owner-%07d", i)
		uid := fmt.Sprintf("u-%05d", i%users)
		if _, err := insBook.Exec(bid, "lib-1", bid); err != nil {
			tb.Fatal(err)
		}
		if _, err := insHl.Exec(fmt.Sprintf("h-%07d", i), uid, bid, fmt.Sprintf("file-%07d:ch1", i)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insSess.Exec(fmt.Sprintf("s-%07d", i), uid, bid, fmt.Sprintf("-%d days", i%3000)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insKobo.Exec(uid, bid); err != nil {
			tb.Fatal(err)
		}
		if _, err := insRP.Exec(uid, bid, fmt.Sprintf("file-%07d", i)); err != nil {
			tb.Fatal(err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO collections (id,user_id,name) VALUES ('c-1','u-00000','C')`); err != nil {
		tb.Fatal(err)
	}
	for i := 0; i < childRows; i++ {
		if _, err := insColl.Exec(fmt.Sprintf("owner-%07d", i)); err != nil {
			tb.Fatal(err)
		}
	}
	// 200 childless victims to delete and time
	for i := 0; i < 200; i++ {
		bid := fmt.Sprintf("victim-%04d", i)
		if _, err := insBook.Exec(bid, "lib-1", bid); err != nil {
			tb.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	if analyze {
		auditIndexExec(tb, db, `ANALYZE`)
	}
	return db
}

// Deletes run inside ONE transaction: a per-statement commit costs a WAL fsync that dwarfs
// the child-lookup CPU and hides the scan entirely. We want the scan cost, not the fsync.
func auditIndexTimeCascadeDeletes(tb testing.TB, db *sql.DB, n int) time.Duration {
	tb.Helper()
	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	st, err := tx.Prepare(`DELETE FROM books WHERE id = ?`)
	if err != nil {
		tb.Fatal(err)
	}
	start := time.Now()
	for i := 0; i < n; i++ {
		if _, err := st.Exec(fmt.Sprintf("victim-%04d", i)); err != nil {
			tb.Fatal(err)
		}
	}
	d := time.Since(start)
	_ = st.Close()
	return d
}

// The four FK child columns that EQP reports as INDEX-SCAN. Each is a book_id that is not the
// leading column of any index, so SQLite's FK enforcement walks the whole child index per delete.
var auditIndexMissingFKIndexes = []string{
	`CREATE INDEX ix_fix_highlights_book ON highlights(book_id)`,
	`CREATE INDEX ix_fix_sessions_book ON reading_sessions(book_id)`,
	`CREATE INDEX ix_fix_kobo_book ON kobo_synced_books(book_id)`,
	`CREATE INDEX ix_fix_btm_book ON book_tracker_mappings(book_id)`,
}

func TestAuditIndexBookDeleteCascadeScan(t *testing.T) {
	const victims = 200
	const users = 200
	for _, analyze := range []bool{false, true} {
		label := "no ANALYZE (production: sqlite_stat1 absent)"
		if analyze {
			label = "with ANALYZE (skip-scan available)"
		}
		t.Logf("################ %s ################", label)
		var first, last [2]time.Duration
		for i, n := range []int{10000, 40000} {
			db := auditIndexSeedCascade(t, n, users, analyze)
			before := auditIndexTimeCascadeDeletes(t, db, victims)

			for _, ddl := range auditIndexMissingFKIndexes {
				auditIndexExec(t, db, ddl)
			}
			if analyze {
				auditIndexExec(t, db, `ANALYZE`)
			}
			after := auditIndexTimeCascadeDeletes(t, db, victims)

			t.Logf("child rows=%-6d  DELETE FROM books x%d (one tx, %d users)", n, victims, users)
			t.Logf("    as shipped     : %-14v (%v per childless book)", before, before/victims)
			t.Logf("    + 4 FK indexes : %-14v (%v per childless book)  -> %.1fx faster",
				after, after/victims, float64(before)/float64(after))
			if i == 0 {
				first = [2]time.Duration{before, after}
			} else {
				last = [2]time.Duration{before, after}
			}
		}
		t.Logf("  SCALING 10k -> 40k child rows:  as shipped %.2fx   |   + FK indexes %.2fx",
			float64(last[0])/float64(first[0]), float64(last[1])/float64(first[1]))
	}
}

// Same effect, isolated to a single child probe with no delete machinery around it.
func TestAuditIndexFKProbeCost(t *testing.T) {
	probes := []struct{ table, col string }{
		{"highlights", "book_id"},
		{"reading_sessions", "book_id"},
		{"kobo_synced_books", "book_id"},
		{"reading_progress", "file_id"},
	}
	for _, analyze := range []bool{false, true} {
		label := "no ANALYZE (production)"
		if analyze {
			label = "with ANALYZE"
		}
		t.Logf("################ %s ################", label)
		for _, n := range []int{10000, 40000} {
			db := auditIndexSeedCascade(t, n, 200, analyze)
			t.Logf("--- child rows=%d ---", n)
			for _, p := range probes {
				q := fmt.Sprintf("SELECT 1 FROM %s WHERE %s = ? LIMIT 1", p.table, p.col)
				st, err := db.Prepare(q)
				if err != nil {
					t.Fatal(err)
				}
				start := time.Now()
				for i := 0; i < 500; i++ {
					var x int
					_ = st.QueryRow(fmt.Sprintf("no-such-%d", i)).Scan(&x)
				}
				d := time.Since(start)
				_ = st.Close()
				t.Logf("  %-18s %-9s 500 probes = %-12v (%v each)  %s",
					p.table, p.col, d, d/500,
					strings.ReplaceAll(auditIndexPlan(t, db, q, "x"), "\n", " | "))
			}
		}
	}
}

// Which of the child tables can the planner actually seek on book_id?
func TestAuditIndexFKChildCoverage(t *testing.T) {
	db := auditIndexDB(t)
	// Every table with a REFERENCES books(id) / users(id) column, probed the way SQLite's
	// FK enforcement probes it: WHERE <child_col> = ?
	cases := []struct{ table, col string }{
		{"highlights", "book_id"},
		{"highlights", "chapter_id"},
		{"reading_sessions", "book_id"},
		{"kobo_synced_books", "book_id"},
		{"book_tracker_mappings", "book_id"},
		{"reading_progress", "book_id"},
		{"reading_progress", "file_id"},
		{"collection_books", "book_id"},
		{"book_tags", "book_id"},
		{"book_series", "book_id"},
		{"book_files", "book_id"},
		{"chapters", "book_id"},
		{"bookmarks", "book_id"},
		{"book_reviews", "book_id"},
		{"book_share_events", "book_id"},
		{"audit_logs", "actor_id"},
		{"user_devices", "user_id"},
		{"user_trackers", "user_id"},
	}
	// SEARCH ...(col=?) is a true O(log n) seek. "SCAN t USING COVERING INDEX ix" is NOT a
	// seek -- it is a full traversal of ix, which happens whenever col is not the LEADING
	// column of any index. Bare "SCAN t" is a full table scan. Both scan variants are O(n).
	t.Logf("=== FK child-column seekability ===")
	t.Logf("    SEEK          = SEARCH ...(col=?), O(log n)")
	t.Logf("    INDEX-SCAN    = col not leading in any index -> full index traversal, O(n)")
	t.Logf("    TABLE-SCAN    = no index at all, O(n) over the row data")
	for _, c := range cases {
		p := auditIndexPlan(t, db, fmt.Sprintf("SELECT 1 FROM %s WHERE %s = ?", c.table, c.col), "x")
		var verdict string
		switch {
		case strings.Contains(p, "SEARCH") && strings.Contains(p, c.col+"=?"):
			verdict = "SEEK"
		case strings.Contains(p, "SCAN") && strings.Contains(p, "INDEX"):
			verdict = "*** INDEX-SCAN ***"
		default:
			verdict = "*** TABLE-SCAN ***"
		}
		t.Logf("  %-22s %-12s %-19s %s", c.table, c.col, verdict, strings.ReplaceAll(p, "\n", " | "))
		if verdict != "SEEK" && !auditIndexKnownUnindexedFK[c.table+"."+c.col] {
			t.Errorf("%s.%s is a CASCADE FK child column but cannot be seeked (%s). "+
				"DELETE FROM the parent will scan this table once per deleted row. "+
				"Add a leading index on %s, or allowlist it in auditIndexKnownUnindexedFK.",
				c.table, c.col, verdict, c.col)
		}
	}
}

// The gaps this audit measured. Listing them keeps the test green on today's schema while
// still failing if a NEW unindexed CASCADE FK child column is introduced -- that is the
// regression this file exists to catch. Delete an entry when its index is added.
var auditIndexKnownUnindexedFK = map[string]bool{
	"highlights.book_id":            true,
	"reading_sessions.book_id":      true,
	"kobo_synced_books.book_id":     true,
	"book_tracker_mappings.book_id": true,
	"reading_progress.file_id":      true,
}

// ---------------------------------------------------------------------------
// 3. reading_progress.file_id + highlights.chapter_id LIKE, hit by the duplicate
//    file merge path (bookFileRecordRepository.RepointFileUserData).
// ---------------------------------------------------------------------------

func TestAuditIndexRepointFileUserData(t *testing.T) {
	var prev time.Duration
	for _, n := range []int{10000, 40000} {
		db := auditIndexSeedCascade(t, n, 200, false)

		t.Logf("--- rows=%d ---", n)
		t.Logf("RepointReadingProgressFile auditIndexPlan:\n%s",
			auditIndexPlan(t, db, `UPDATE reading_progress SET file_id = ?, updated_at = CURRENT_TIMESTAMP WHERE file_id = ?`, "new", "old"))
		t.Logf("RepointHighlightChapters auditIndexPlan:\n%s",
			auditIndexPlan(t, db, `UPDATE highlights SET chapter_id = CAST(? AS TEXT) || substr(chapter_id, length(CAST(? AS TEXT)) + 1), updated_at = CURRENT_TIMESTAMP WHERE chapter_id LIKE CAST(? AS TEXT) || ':%'`, "new", "old", "old"))

		start := time.Now()
		for i := 0; i < 20; i++ {
			old := fmt.Sprintf("file-%07d", i)
			if _, err := db.Exec(`UPDATE reading_progress SET file_id = ?, updated_at = CURRENT_TIMESTAMP WHERE file_id = ?`, "n-"+old, old); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`UPDATE highlights SET chapter_id = CAST(? AS TEXT) || substr(chapter_id, length(CAST(? AS TEXT)) + 1), updated_at = CURRENT_TIMESTAMP WHERE chapter_id LIKE CAST(? AS TEXT) || ':%'`, "n-"+old, old, old); err != nil {
				t.Fatal(err)
			}
		}
		d := time.Since(start)
		t.Logf("20x RepointFileUserData = %v (%v per merged file)", d, d/20)
		if prev > 0 {
			t.Logf("  -> 4x rows changed cost by %.2fx", float64(d)/float64(prev))
		}
		prev = d
	}
}

// ---------------------------------------------------------------------------
// 4. ORDER BY / GROUP BY that cannot ride an index.
// ---------------------------------------------------------------------------

func auditIndexSeedBooks(tb testing.TB, n int) *sql.DB {
	tb.Helper()
	db := auditIndexDB(tb)
	auditIndexExec(tb, db, `INSERT INTO libraries (id,name) VALUES ('lib-1','L')`)
	auditIndexExec(tb, db, `INSERT INTO users (id,email,password_hash) VALUES ('u-1','a@b.c','x')`)
	auditIndexExec(tb, db, `INSERT INTO collections (id,user_id,name) VALUES ('c-1','u-1','C')`)
	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	insBook, _ := tx.Prepare(`INSERT INTO books (id,library_id,title,status,created_at) VALUES (?,?,?,'active',datetime('now', ?))`)
	insFile, _ := tx.Prepare(`INSERT INTO book_files (id,book_id,path,format,size_bytes,mod_time,hash) VALUES (?,?,?,?,1,CURRENT_TIMESTAMP,?)`)
	insColl, _ := tx.Prepare(`INSERT INTO collection_books (collection_id,book_id) VALUES ('c-1',?)`)
	insRev, _ := tx.Prepare(`INSERT INTO book_reviews (user_id,book_id,rating) VALUES ('u-1',?,3)`)
	formats := []string{"epub", "pdf", "cbz", "mobi", "azw3"}
	for i := 0; i < n; i++ {
		bid := fmt.Sprintf("b-%07d", i)
		if _, err := insBook.Exec(bid, "lib-1", bid, fmt.Sprintf("-%d seconds", i%86400)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insFile.Exec("f-"+bid, bid, "/p/"+bid, formats[i%len(formats)], fmt.Sprintf("hash-%d", i/2)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insColl.Exec(bid); err != nil {
			tb.Fatal(err)
		}
	}
	_, _ = insRev.Exec("b-0000000", 3)
	for _, st := range []*sql.Stmt{insBook, insFile, insColl, insRev} {
		_ = st.Close()
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	auditIndexExec(tb, db, `ANALYZE`)
	return db
}

func TestAuditIndexOrderByTempBTree(t *testing.T) {
	db := auditIndexSeedBooks(t, 20000)
	cases := []struct {
		name  string
		query string
		args  []any
	}{
		{"ListBookIDs (books.sql:13)", `SELECT id FROM books WHERE (?1 IS NULL OR datetime(created_at) < datetime(?1) OR (datetime(created_at) = datetime(?1) AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, []any{nil, nil, 24}},
		{"ListBookIDs page2 (cursor set)", `SELECT id FROM books WHERE (?1 IS NULL OR datetime(created_at) < datetime(?1) OR (datetime(created_at) = datetime(?1) AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, []any{"2026-08-06 00:00:00", "b-0000500", 24}},
		{"SearchBookIDs no filters (books.sql:94)", `SELECT b.id FROM books b WHERE (?1 IS NULL OR datetime(b.created_at) < datetime(?1)) AND (?2 IS NULL OR b.library_id = ?2) ORDER BY b.created_at DESC, b.id DESC LIMIT ?3`, []any{nil, nil, 24}},
		{"SearchBookIDs library filter", `SELECT b.id FROM books b WHERE (?1 IS NULL OR datetime(b.created_at) < datetime(?1)) AND (?2 IS NULL OR b.library_id = ?2) ORDER BY b.created_at DESC, b.id DESC LIMIT ?3`, []any{nil, "lib-1", 24}},
		{"GetBookIDsInCollection (user_features.sql:58)", `SELECT b.id FROM books b JOIN collection_books cb ON cb.book_id = b.id WHERE cb.collection_id = ?1 AND (?2 IS NULL OR datetime(b.created_at) < datetime(?2) OR (datetime(b.created_at) = datetime(?2) AND b.id < ?3)) ORDER BY datetime(b.created_at) DESC, b.id DESC LIMIT ?4`, []any{"c-1", nil, nil, 24}},
		{"ListFormatsWithCount (metadata.sql:148)", `SELECT LOWER(bf.format), UPPER(bf.format), COUNT(DISTINCT bf.book_id) FROM book_files bf JOIN books b ON b.id = bf.book_id WHERE b.library_id IN (SELECT value FROM json_each(?1)) GROUP BY LOWER(bf.format) ORDER BY LOWER(bf.format) ASC LIMIT ?2`, []any{`["lib-1"]`, 100}},
		{"GetDuplicateFiles (files.sql:1)", `SELECT bf.hash, COUNT(*), GROUP_CONCAT(bf.id) FROM book_files bf JOIN books b ON bf.book_id = b.id WHERE bf.hash IS NOT NULL AND bf.hash != '' GROUP BY bf.hash HAVING COUNT(*) > 1 LIMIT ?1 OFFSET ?2`, []any{20, 0}},
		{"ListAllReviews (user_features.sql:326)", `SELECT br.user_id, br.book_id FROM book_reviews br JOIN users u ON u.id = br.user_id JOIN books b ON b.id = br.book_id ORDER BY br.updated_at DESC LIMIT ? OFFSET ?`, []any{20, 0}},
		{"PruneFinishedJobs (operations.sql:64)", `DELETE FROM jobs WHERE id IN (SELECT id FROM jobs WHERE status IN ('completed','failed') ORDER BY updated_at DESC LIMIT -1 OFFSET ?)`, []any{500}},
		{"GetFilesByBookId (core.sql:56)", `SELECT id FROM book_files WHERE book_id = ? ORDER BY CASE WHEN LOWER(format)='epub' THEN 0 ELSE 1 END, created_at ASC`, []any{"b-0000001"}},
		{"GetReadingHeatmap (reading_sessions.sql:19)", `SELECT session_date, SUM(duration_seconds) FROM reading_sessions WHERE user_id = ? AND session_date >= date('now','-365 days') GROUP BY session_date ORDER BY session_date ASC`, []any{"u-1"}},
		{"ListAuditActions (audit.sql:29)", `SELECT DISTINCT action FROM audit_logs ORDER BY action ASC`, nil},
	}
	for _, c := range cases {
		p := auditIndexPlan(t, db, c.query, c.args...)
		flags := []string{}
		if strings.Contains(p, "TEMP B-TREE") {
			flags = append(flags, "TEMP B-TREE")
		}
		for _, line := range strings.Split(p, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "SCAN") && !strings.Contains(line, "USING INDEX") && !strings.Contains(line, "USING COVERING INDEX") {
				flags = append(flags, "FULL SCAN")
				break
			}
		}
		verdict := "ok"
		if len(flags) > 0 {
			verdict = "*** " + strings.Join(flags, " + ") + " ***"
		}
		t.Logf("%-46s %s\n%s", c.name, verdict, p)
	}
}

// Does the cursor page actually get cheaper with a cursor, or does datetime() force a
// full index walk every page?
func TestAuditIndexKeysetCursorCost(t *testing.T) {
	var prev time.Duration
	for _, n := range []int{10000, 40000} {
		db := auditIndexSeedBooks(t, n)
		run := func(label, q string, args ...any) time.Duration {
			start := time.Now()
			for i := 0; i < 200; i++ {
				rows, err := db.Query(q, args...)
				if err != nil {
					t.Fatal(err)
				}
				for rows.Next() {
				}
				rows.Close()
			}
			d := time.Since(start)
			t.Logf("  rows=%d %-34s 200 pages = %v (%v/page)", n, label, d, d/200)
			return d
		}
		run("ListBookIDs page1", `SELECT id FROM books WHERE (?1 IS NULL OR datetime(created_at) < datetime(?1) OR (datetime(created_at) = datetime(?1) AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, nil, nil, 24)

		// Page 1 has a NULL cursor, so it proves nothing about the cursor predicate. The
		// question for keyset pagination is whether a DEEP page still costs the same as a
		// shallow one -- that is the entire point of keyset over OFFSET. datetime(created_at)
		// is a function call on the column, so no index on created_at can seek it.
		var deepTime string
		var deepID string
		row := db.QueryRow(`SELECT created_at, id FROM books ORDER BY created_at DESC, id DESC LIMIT 1 OFFSET ?`, n-500)
		if err := row.Scan(&deepTime, &deepID); err != nil {
			t.Fatal(err)
		}
		const cursorQ = `SELECT id FROM books WHERE (?1 IS NULL OR datetime(created_at) < datetime(?1) OR (datetime(created_at) = datetime(?1) AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`
		run("ListBookIDs DEEP page (datetime())", cursorQ, deepTime, deepID, 24)
		t.Logf("  deep-page plan:\n%s", auditIndexPlan(t, db, cursorQ, deepTime, deepID, 24))

		// Same page, same result set, but comparing the raw column instead of datetime(col).
		const sargableQ = `SELECT id FROM books WHERE (?1 IS NULL OR created_at < ?1 OR (created_at = ?1 AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`
		run("ListBookIDs DEEP page (sargable)", sargableQ, deepTime, deepID, 24)
		t.Logf("  sargable plan:\n%s", auditIndexPlan(t, db, sargableQ, deepTime, deepID, 24))
		d := run("GetBookIDsInCollection page1", `SELECT b.id FROM books b JOIN collection_books cb ON cb.book_id = b.id WHERE cb.collection_id = ?1 AND (?2 IS NULL OR datetime(b.created_at) < datetime(?2) OR (datetime(b.created_at) = datetime(?2) AND b.id < ?3)) ORDER BY datetime(b.created_at) DESC, b.id DESC LIMIT ?4`, "c-1", nil, nil, 24)
		run("ListFormatsWithCount", `SELECT LOWER(bf.format), COUNT(DISTINCT bf.book_id) FROM book_files bf JOIN books b ON b.id = bf.book_id WHERE b.library_id IN (SELECT value FROM json_each(?1)) GROUP BY LOWER(bf.format) ORDER BY LOWER(bf.format) ASC LIMIT ?2`, `["lib-1"]`, 100)
		if prev > 0 {
			t.Logf("  -> collection page cost 4x rows: %.2fx", float64(d)/float64(prev))
		}
		prev = d
	}
}

// ---------------------------------------------------------------------------
// 5. Indexes that can never be used.
// ---------------------------------------------------------------------------

func TestAuditIndexUnusable(t *testing.T) {
	db := auditIndexSeedBooks(t, 20000)
	auditIndexExec(t, db, `INSERT INTO series (id,name) VALUES ('s-1','Dragon Ball')`)
	auditIndexExec(t, db, `INSERT INTO publishers (id,name) VALUES ('p-1','Shueisha')`)
	auditIndexExec(t, db, `INSERT INTO languages (id,name) VALUES ('l-1','ja')`)

	cases := []struct {
		name  string
		query string
		args  []any
		want  string
	}{
		{"series LIKE '%x%' (komga.sql:16)", `SELECT s.id FROM series s WHERE (?1 IS NULL OR s.name LIKE '%' || ?1 || '%') ORDER BY s.name ASC, s.id ASC LIMIT ?2`, []any{"ball", 20}, "idx_series_name cannot serve a leading-wildcard LIKE"},
		{"series name = ? (metadata.sql:1)", `SELECT id FROM series WHERE name = ? LIMIT 1`, []any{"Dragon Ball"}, "which index wins: idx_series_name or the UNIQUE autoindex?"},
		{"publishers name = ?", `SELECT id FROM publishers WHERE name = ? LIMIT 1`, []any{"Shueisha"}, ""},
		{"languages name = ?", `SELECT id FROM languages WHERE name = ? LIMIT 1`, []any{"ja"}, ""},
		{"books popular-order index used?", `SELECT id FROM books ORDER BY download_count DESC, average_rating DESC, read_count DESC, created_at DESC LIMIT 20`, nil, "only place idx_books_popular could apply"},
		{"books filter_top_downloaded", `SELECT b.id FROM books b WHERE b.download_count > 0 ORDER BY b.created_at DESC, b.id DESC LIMIT 24`, nil, ""},
		{"jobs status filter (operations.sql:6)", `SELECT id FROM jobs WHERE (?1 = '' OR status = ?1) AND (?2 = '' OR type = ?2) ORDER BY created_at DESC LIMIT ?3 OFFSET ?4`, []any{"", "", 20, 0}, ""},
		{"webhooks is_active (webhooks.sql:14)", `SELECT id FROM webhooks WHERE is_active = 1`, nil, "is_active is 2-valued"},
		{"users LIKE search (users.sql:78)", `SELECT u.id FROM users u WHERE lower(u.email) LIKE '%' || lower(?1) || '%' ORDER BY u.created_at DESC, u.id ASC LIMIT ?2`, []any{"a@b", 20}, ""},
	}
	for _, c := range cases {
		p := auditIndexPlan(t, db, c.query, c.args...)
		note := ""
		if c.want != "" {
			note = "   // " + c.want
		}
		t.Logf("%-40s%s\n%s", c.name, note, p)
	}
}

// Write-amplification: what does each redundant index actually cost on insert?
func TestAuditIndexRedundantWriteCost(t *testing.T) {
	insert := func(drop []string) time.Duration {
		db := auditIndexDB(t)
		for _, ix := range drop {
			auditIndexExec(t, db, "DROP INDEX IF EXISTS "+ix)
		}
		auditIndexExec(t, db, `INSERT INTO libraries (id,name) VALUES ('lib-1','L')`)
		auditIndexExec(t, db, `INSERT INTO users (id,email,password_hash) VALUES ('u-1','a@b.c','x')`)
		tx, _ := db.Begin()
		insBook, _ := tx.Prepare(`INSERT INTO books (id,library_id,title,status) VALUES (?,?,?,'active')`)
		insRev, _ := tx.Prepare(`INSERT INTO book_reviews (user_id,book_id,rating) VALUES ('u-1',?,3)`)
		insBm, _ := tx.Prepare(`INSERT INTO bookmarks (user_id,book_id) VALUES ('u-1',?)`)
		insRP, _ := tx.Prepare(`INSERT INTO reading_progress (user_id,book_id,chapter_ref) VALUES ('u-1',?,'c')`)
		insCh, _ := tx.Prepare(`INSERT INTO chapters (id,book_id,title,chapter_index) VALUES (?,?,'t',1)`)
		start := time.Now()
		for i := 0; i < 20000; i++ {
			bid := fmt.Sprintf("b-%07d", i)
			if _, err := insBook.Exec(bid, "lib-1", bid); err != nil {
				t.Fatal(err)
			}
			_, _ = insRev.Exec(bid)
			_, _ = insBm.Exec(bid)
			_, _ = insRP.Exec(bid)
			_, _ = insCh.Exec("c-"+bid, bid)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		return time.Since(start)
	}
	redundant := []string{
		"idx_reading_progress_user_time",
		"idx_bookmarks_user_time",
		"idx_book_reviews_book",
		"idx_collections_user_id",
		"idx_books_library_id",
		"idx_chapters_book",
		"idx_role_permissions_role_id",
		"idx_role_permissions_permission_key",
		"idx_series_name",
		"idx_publishers_name",
		"idx_languages_name",
		"idx_user_roles_user_id",
		"idx_book_share_events_book",
	}
	// Order matters: the first arm warms the page cache and the tmpfs, so whichever arm runs
	// second is systematically penalised. Running both orders and averaging cancels that bias.
	// Three same-order runs gave -2.2%, -10.2%, -5.0% -- a negative "cost" for ADDING indexes
	// is physically impossible, which is the tell that the harness, not the schema, is being
	// measured.
	withA := insert(nil)
	withoutA := insert(redundant)
	withoutB := insert(redundant)
	withB := insert(nil)
	with := (withA + withB) / 2
	without := (withoutA + withoutB) / 2
	t.Logf("  order bias check: with(first)=%v with(last)=%v | without(last)=%v without(first)=%v", withA, withB, withoutA, withoutB)
	t.Logf("20k books + reviews + bookmarks + progress + chapters")
	t.Logf("  all indexes present     : %v", with)
	t.Logf("  %d redundant dropped    : %v", len(redundant), without)
	t.Logf("  -> redundant index write overhead: %.1f%%", (float64(with)/float64(without)-1)*100)
}

// The two-arm deep-page test conflated two independent non-sargable constructs, so it could
// not attribute cost. There are three suspects in ListBookIDs' WHERE clause and they must be
// separated before any of them is reported as the cause:
//   a) datetime(created_at) -- a function on the column; no index on created_at can seek it.
//   b) the `?1 IS NULL OR ...` disjunction sqlc emits for every narg cursor -- a disjunction
//      over a non-column term also blocks a range seek.
//   c) ORDER BY created_at DESC, id DESC against an index that stores created_at only, which
//      forces the "LAST TERM OF ORDER BY" temp b-tree regardless of the WHERE clause.
func TestAuditIndexKeysetDecompose(t *testing.T) {
	for _, n := range []int{10000, 40000} {
		db := auditIndexSeedBooks(t, n)
		var deepTime, deepID string
		if err := db.QueryRow(`SELECT created_at, id FROM books ORDER BY created_at DESC, id DESC LIMIT 1 OFFSET ?`, n-500).
			Scan(&deepTime, &deepID); err != nil {
			t.Fatal(err)
		}
		var distinct int
		if err := db.QueryRow(`SELECT COUNT(DISTINCT created_at) FROM books`).Scan(&distinct); err != nil {
			t.Fatal(err)
		}
		t.Logf("rows=%d distinct created_at=%d (ties inflate the temp b-tree if this is low)", n, distinct)

		run := func(label, q string, args ...any) {
			for i := 0; i < 20; i++ {
				rows, err := db.Query(q, args...)
				if err != nil {
					t.Fatal(err)
				}
				for rows.Next() {
				}
				rows.Close()
			}
			start := time.Now()
			for i := 0; i < 200; i++ {
				rows, err := db.Query(q, args...)
				if err != nil {
					t.Fatal(err)
				}
				for rows.Next() {
				}
				rows.Close()
			}
			d := time.Since(start)
			plan := strings.ReplaceAll(auditIndexPlan(t, db, q, args...), "\n", " | ")
			t.Logf("  %-38s %9v/page   %s", label, d/200, plan)
		}

		run("a+b+c shipped (datetime + ORnull)",
			`SELECT id FROM books WHERE (?1 IS NULL OR datetime(created_at) < datetime(?1) OR (datetime(created_at) = datetime(?1) AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, deepTime, deepID, 24)
		run("b+c   drop datetime()",
			`SELECT id FROM books WHERE (?1 IS NULL OR created_at < ?1 OR (created_at = ?1 AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, deepTime, deepID, 24)
		run("c     drop datetime() and ORnull",
			`SELECT id FROM books WHERE (created_at < ?1 OR (created_at = ?1 AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, deepTime, deepID, 24)
		run("none  row-value seek, same result",
			`SELECT id FROM books WHERE (created_at, id) < (?1, ?2) ORDER BY created_at DESC, id DESC LIMIT ?3`, deepTime, deepID, 24)

		auditIndexExec(t, db, `CREATE INDEX ix_fix_books_created_id ON books(created_at DESC, id DESC)`)
		run("none  + (created_at,id) index",
			`SELECT id FROM books WHERE (created_at, id) < (?1, ?2) ORDER BY created_at DESC, id DESC LIMIT ?3`, deepTime, deepID, 24)
		run("a+b+c shipped + that index",
			`SELECT id FROM books WHERE (?1 IS NULL OR datetime(created_at) < datetime(?1) OR (datetime(created_at) = datetime(?1) AND id < ?2)) ORDER BY created_at DESC, id DESC LIMIT ?3`, deepTime, deepID, 24)
		db.Close()
	}
}
