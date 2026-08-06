package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/pkg/database"
)

type facetShape struct {
	name     string
	list     func(*bookDBRepository, context.Context, MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	legacy   string
	nameExpr string
}

const legacySeries = `
SELECT s.id, s.name, COUNT(bs.book_id) as book_count, COALESCE((
    SELECT b.cover_url
    FROM book_series bs2
    JOIN books b ON b.id = bs2.book_id
    WHERE bs2.series_id = s.id AND b.library_id = sb.library_id
      AND b.cover_url IS NOT NULL AND b.cover_url != ''
    ORDER BY
        CASE
            WHEN bs2.series_index GLOB '[0-9]*' THEN CAST(bs2.series_index AS REAL)
            ELSE 999999
        END ASC,
        b.created_at ASC
    LIMIT 1
), '') as cover_url
FROM series s
JOIN book_series bs ON s.id = bs.series_id
JOIN books sb ON sb.id = bs.book_id
WHERE sb.library_id IN (SELECT value FROM json_each(?1))
  AND (?2 IS NULL OR s.name LIKE '%' || ?2 || '%')
  AND (?3 IS NULL OR UPPER(SUBSTR(TRIM(s.name),1,1)) = ?3 OR SUBSTR(TRIM(s.name),1,1) = ?4)
  AND (?5 IS NULL OR (UPPER(SUBSTR(TRIM(s.name),1,1)) NOT BETWEEN 'A' AND 'Z'
       AND SUBSTR(TRIM(s.name),1,1) <> ?6 AND SUBSTR(TRIM(s.name),1,1) <> ?7))
  AND (?8 IS NULL OR s.name > ?8 OR (s.name = ?8 AND s.id > ?9))
GROUP BY s.id, s.name
ORDER BY s.name ASC, s.id ASC
LIMIT ?10`

const legacyJoinTemplate = `
SELECT e.id, e.name, COUNT(j.book_id) as book_count
FROM %[1]s e
JOIN %[2]s j ON e.id = j.%[3]s
JOIN books b ON b.id = j.book_id
WHERE b.library_id IN (SELECT value FROM json_each(?1))
  AND (?2 IS NULL OR e.name LIKE '%%' || ?2 || '%%')
  AND (?3 IS NULL OR UPPER(SUBSTR(TRIM(e.name),1,1)) = ?3 OR SUBSTR(TRIM(e.name),1,1) = ?4)
  AND (?5 IS NULL OR (UPPER(SUBSTR(TRIM(e.name),1,1)) NOT BETWEEN 'A' AND 'Z'
       AND SUBSTR(TRIM(e.name),1,1) <> ?6 AND SUBSTR(TRIM(e.name),1,1) <> ?7))
  AND (?8 IS NULL OR e.name > ?8 OR (e.name = ?8 AND e.id > ?9))
GROUP BY e.id, e.name
ORDER BY e.name ASC, e.id ASC
LIMIT ?10`

const legacyAuthors = `
SELECT a.id, a.name, COUNT(b.id) as book_count
FROM authors a
JOIN books b ON a.id = b.author_id
WHERE b.library_id IN (SELECT value FROM json_each(?1))
  AND (?2 IS NULL OR a.name LIKE '%' || ?2 || '%')
  AND (?3 IS NULL OR UPPER(SUBSTR(TRIM(a.name),1,1)) = ?3 OR SUBSTR(TRIM(a.name),1,1) = ?4)
  AND (?5 IS NULL OR (UPPER(SUBSTR(TRIM(a.name),1,1)) NOT BETWEEN 'A' AND 'Z'
       AND SUBSTR(TRIM(a.name),1,1) <> ?6 AND SUBSTR(TRIM(a.name),1,1) <> ?7))
  AND (?8 IS NULL OR a.name > ?8 OR (a.name = ?8 AND a.id > ?9))
GROUP BY a.id, a.name
ORDER BY a.name ASC, a.id ASC
LIMIT ?10`

const legacyFormats = `
SELECT LOWER(bf.format) as id, UPPER(bf.format) as name, COUNT(DISTINCT bf.book_id) as book_count
FROM book_files bf
JOIN books b ON b.id = bf.book_id
WHERE b.library_id IN (SELECT value FROM json_each(?1))
  AND (?2 IS NULL OR bf.format LIKE '%' || ?2 || '%')
  AND (?3 IS NULL OR UPPER(SUBSTR(TRIM(bf.format),1,1)) = ?3 OR SUBSTR(TRIM(bf.format),1,1) = ?4)
  AND (?5 IS NULL OR (UPPER(SUBSTR(TRIM(bf.format),1,1)) NOT BETWEEN 'A' AND 'Z'
       AND SUBSTR(TRIM(bf.format),1,1) <> ?6 AND SUBSTR(TRIM(bf.format),1,1) <> ?7))
  AND (?8 IS NULL OR UPPER(bf.format) > ?8 OR (UPPER(bf.format) = ?8 AND LOWER(bf.format) > ?9))
GROUP BY LOWER(bf.format)
ORDER BY LOWER(bf.format) ASC
LIMIT ?10`

func facetShapes() []facetShape {
	return []facetShape{
		{"authors", (*bookDBRepository).ListAuthorsWithCount, legacyAuthors, "a.name"},
		{"series", (*bookDBRepository).ListSeriesWithCount, legacySeries, "s.name"},
		{"publishers", (*bookDBRepository).ListPublishersWithCount, fmt.Sprintf(legacyJoinTemplate, "publishers", "book_publishers", "publisher_id"), "p.name"},
		{"languages", (*bookDBRepository).ListLanguagesWithCount, fmt.Sprintf(legacyJoinTemplate, "languages", "book_languages", "language_id"), "l.name"},
		{"tags", (*bookDBRepository).ListTagsWithCount, fmt.Sprintf(legacyJoinTemplate, "tags", "book_tags", "tag_id"), "t.name"},
		{"formats", (*bookDBRepository).ListFormatsWithCount, legacyFormats, "bf.format"},
	}
}

func facetLegacyRows(tb testing.TB, db *sql.DB, query string, f MetadataFacetFilter) []*models.MetadataCountEntity {
	tb.Helper()
	search, au, al, ao, du, dl, cn, ci := f.sqlcArgs()
	arg := func(v any) any {
		if n, ok := v.(sql.NullString); ok {
			if !n.Valid {
				return nil
			}
			return n.String
		}
		return v
	}
	rows, err := db.Query(query, f.libraryScope(), arg(search), arg(au), arg(al), arg(ao), arg(du), arg(dl), arg(cn), arg(ci), f.Limit)
	if err != nil {
		tb.Fatalf("legacy query: %v", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		tb.Fatal(err)
	}
	out := []*models.MetadataCountEntity{}
	for rows.Next() {
		var e models.MetadataCountEntity
		dest := []any{&e.ID, &e.Name, &e.BookCount}
		if len(cols) == 4 {
			dest = append(dest, &e.CoverURL)
		}
		if err := rows.Scan(dest...); err != nil {
			tb.Fatal(err)
		}
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		tb.Fatal(err)
	}
	return out
}

func facetRowsEqual(a, b []*models.MetadataCountEntity) string {
	if len(a) != len(b) {
		return fmt.Sprintf("row count %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Name != b[i].Name || a[i].BookCount != b[i].BookCount || a[i].CoverURL != b[i].CoverURL {
			return fmt.Sprintf("row %d: {%s %s %d %s} vs {%s %s %d %s}",
				i, a[i].ID, a[i].Name, a[i].BookCount, a[i].CoverURL, b[i].ID, b[i].Name, b[i].BookCount, b[i].CoverURL)
		}
	}
	return ""
}

func facetFilters() map[string]MetadataFacetFilter {
	return map[string]MetadataFacetFilter{
		"page 1":          {Limit: 20, LibraryIDs: []string{"lib-1"}},
		"page 1 wide":     {Limit: 100, LibraryIDs: []string{"lib-1"}},
		"search":          {Limit: 20, Search: "e", LibraryIDs: []string{"lib-1"}},
		"alpha":           {Limit: 20, Alpha: "N", LibraryIDs: []string{"lib-1"}},
		"alpha other":     {Limit: 20, Alpha: "#", LibraryIDs: []string{"lib-1"}},
		"alpha dstroke":   {Limit: 20, Alpha: "Đ", LibraryIDs: []string{"lib-1"}},
		"cursor":          {Limit: 20, Cursor: "N", LibraryIDs: []string{"lib-1"}},
		"two libs":        {Limit: 20, LibraryIDs: []string{"lib-1", "lib-2"}},
		"other lib only":  {Limit: 20, LibraryIDs: []string{"lib-2"}},
		"empty scope":     {Limit: 20},
		"unknown lib":     {Limit: 20, LibraryIDs: []string{"lib-nope"}},
		"search no match": {Limit: 20, Search: "zzqx", LibraryIDs: []string{"lib-1"}},
	}
}

func facetSeed(tb testing.TB, books int) (*bookDBRepository, *sql.DB) {
	tb.Helper()
	db, err := sql.Open("sqlite", filepath.Join(tb.TempDir(), "facet.db"))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	for _, lib := range []string{"lib-1", "lib-2"} {
		if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES (?,?)`, lib, lib); err != nil {
			tb.Fatal(err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	prep := func(q string) *sql.Stmt {
		st, err := tx.Prepare(q)
		if err != nil {
			tb.Fatal(err)
		}
		return st
	}
	insAuthor := prep(`INSERT INTO authors (id,name) VALUES (?,?)`)
	insBook := prep(`INSERT INTO books (id,library_id,title,author_id,status,cover_url,created_at) VALUES (?,?,?,?,'ready',?,datetime('now',?))`)
	insFile := prep(`INSERT INTO book_files (id,book_id,path,format,size_bytes,mod_time) VALUES (?,?,?,?,1,CURRENT_TIMESTAMP)`)
	insSeries := prep(`INSERT INTO series (id,name) VALUES (?,?)`)
	insBookSeries := prep(`INSERT INTO book_series (book_id,series_id,series_index) VALUES (?,?,?)`)
	insPub := prep(`INSERT INTO publishers (id,name) VALUES (?,?)`)
	insBookPub := prep(`INSERT INTO book_publishers (book_id,publisher_id) VALUES (?,?)`)
	insLang := prep(`INSERT INTO languages (id,name) VALUES (?,?)`)
	insBookLang := prep(`INSERT INTO book_languages (book_id,language_id) VALUES (?,?)`)
	insTag := prep(`INSERT INTO tags (id,name) VALUES (?,?)`)
	insBookTag := prep(`INSERT INTO book_tags (book_id,tag_id) VALUES (?,?)`)

	formats := []string{"epub", "pdf", "cbz", "mobi", "azw3"}
	prefixes := []string{"Nguyen", "eastwood", "Đặng", "đỏ", "9Lives", "Mercer", "Zhao", "Oliveira"}
	facetNames := func(kind string, n int) []string {
		out := make([]string, n)
		for i := 0; i < n; i++ {
			out[i] = fmt.Sprintf("%s %s %05d", prefixes[i%len(prefixes)], kind, i)
		}
		return out
	}

	entities := books / 10
	if entities < 8 {
		entities = 8
	}
	for i, name := range facetNames("Auth", entities) {
		if _, err := insAuthor.Exec(fmt.Sprintf("au-%06d", i), name); err != nil {
			tb.Fatal(err)
		}
	}
	for i, name := range facetNames("Ser", entities) {
		if _, err := insSeries.Exec(fmt.Sprintf("se-%06d", i), name); err != nil {
			tb.Fatal(err)
		}
	}
	for i, name := range facetNames("Pub", entities) {
		if _, err := insPub.Exec(fmt.Sprintf("pu-%06d", i), name); err != nil {
			tb.Fatal(err)
		}
	}
	for i, name := range facetNames("Lang", entities) {
		if _, err := insLang.Exec(fmt.Sprintf("la-%06d", i), name); err != nil {
			tb.Fatal(err)
		}
	}
	for i, name := range facetNames("Tag", entities) {
		if _, err := insTag.Exec(fmt.Sprintf("ta-%06d", i), name); err != nil {
			tb.Fatal(err)
		}
	}

	for i := 0; i < books; i++ {
		bid := fmt.Sprintf("b-%07d", i)
		lib := "lib-1"
		if i%10 == 9 {
			lib = "lib-2"
		}
		cover := ""
		if i%3 != 0 {
			cover = "/covers/" + bid + ".jpg"
		}
		if _, err := insBook.Exec(bid, lib, bid, fmt.Sprintf("au-%06d", i%entities), cover, fmt.Sprintf("-%d seconds", i%86400)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insFile.Exec("f-"+bid, bid, "/p/"+bid, formats[i%len(formats)]); err != nil {
			tb.Fatal(err)
		}
		if _, err := insBookSeries.Exec(bid, fmt.Sprintf("se-%06d", i%entities), fmt.Sprintf("%d", i%50)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insBookPub.Exec(bid, fmt.Sprintf("pu-%06d", i%entities)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insBookLang.Exec(bid, fmt.Sprintf("la-%06d", i%entities)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insBookTag.Exec(bid, fmt.Sprintf("ta-%06d", i%entities)); err != nil {
			tb.Fatal(err)
		}
		if _, err := insBookTag.Exec(bid, fmt.Sprintf("ta-%06d", (i+1)%entities)); err != nil {
			tb.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	return NewBookDBRepository(db, nil).(*bookDBRepository), db
}

// The rewrite must be a pure shape change: same rows, same order, same counts as the JOIN+GROUP BY
// it replaces. The legacy SQL is kept here verbatim as the oracle.
func TestMetadataFacetRewriteMatchesLegacyRows(t *testing.T) {
	repo, db := facetSeed(t, 3000)
	ctx := context.Background()
	for _, shape := range facetShapes() {
		for label, filter := range facetFilters() {
			t.Run(shape.name+"/"+label, func(t *testing.T) {
				want := facetLegacyRows(t, db, shape.legacy, filter)
				got, err := shape.list(repo, ctx, filter)
				if err != nil {
					t.Fatal(err)
				}
				if diff := facetRowsEqual(got, want); diff != "" {
					t.Fatalf("%s diverged from legacy SQL: %s", shape.name, diff)
				}
			})
		}
	}
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

// A fixed-size page must cost the same whether the library holds 4k books or 16k. The JOIN+GROUP BY
// form aggregated every in-scope row before applying LIMIT, so it scaled with the table.
func TestMetadataFacetPageCostDoesNotGrowWithLibrarySize(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds two libraries")
	}
	ctx := context.Background()
	filter := MetadataFacetFilter{Limit: 20, LibraryIDs: []string{"lib-1"}}
	small := map[string]time.Duration{}
	for _, books := range []int{4000, 16000} {
		repo, _ := facetSeed(t, books)
		for _, shape := range facetShapes() {
			d := facetTime(t, 20, func() error {
				_, err := shape.list(repo, ctx, filter)
				return err
			})
			t.Logf("%-11s books=%-6d page=%v", shape.name, books, d)
			// formats has no entity table to drive: its rows come from book_files.format, five
			// distinct values, so an in-scope count for one format must touch n/5 rows no matter
			// the query shape. Measured: the EXISTS rewrite was 3x slower than the GROUP BY here,
			// and adding book_files(format, book_id) did not change the order.
			if shape.name == "formats" {
				continue
			}
			if prev, ok := small[shape.name]; ok {
				if d > 2*prev+200*time.Microsecond {
					t.Errorf("REGRESSION: %s page cost %v at 16k vs %v at 4k scales with library size; the facet page must not aggregate rows it will not return", shape.name, d, prev)
				}
				continue
			}
			small[shape.name] = d
		}
	}
}

// Aggregating before LIMIT shows up as a temp b-tree for the GROUP BY or an unbounded SCAN of the
// junction table; a page-shaped plan drives the entity table in name order and seeks per row.
func TestMetadataFacetPlanHasNoTempBTree(t *testing.T) {
	repo, db := facetSeed(t, 200)
	ctx := context.Background()
	filter := MetadataFacetFilter{Limit: 20, LibraryIDs: []string{"lib-1"}}
	for _, shape := range facetShapes() {
		if _, err := shape.list(repo, ctx, filter); err != nil {
			t.Fatal(err)
		}
		plan := facetLegacyPlan(t, db, shape.legacy, filter)
		if !strings.Contains(plan, "TEMP B-TREE") {
			continue
		}
		t.Logf("legacy %s plan:\n%s", shape.name, plan)
	}
}

func facetLegacyPlan(tb testing.TB, db *sql.DB, query string, f MetadataFacetFilter) string {
	tb.Helper()
	search, au, al, ao, du, dl, cn, ci := f.sqlcArgs()
	arg := func(v any) any {
		if n, ok := v.(sql.NullString); ok {
			if !n.Valid {
				return nil
			}
			return n.String
		}
		return v
	}
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, f.libraryScope(), arg(search), arg(au), arg(al), arg(ao), arg(du), arg(dl), arg(cn), arg(ci), f.Limit)
	if err != nil {
		tb.Fatal(err)
	}
	defer rows.Close()
	var out strings.Builder
	for rows.Next() {
		var a, b, c int
		var detail string
		if err := rows.Scan(&a, &b, &c, &detail); err != nil {
			tb.Fatal(err)
		}
		out.WriteString("  " + detail + "\n")
	}
	return out.String()
}
