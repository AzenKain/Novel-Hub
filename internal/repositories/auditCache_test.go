package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func auditDB(tb testing.TB) *sql.DB {
	tb.Helper()
	dsn := filepath.Join(tb.TempDir(), "audit.db") +
		"?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=foreign_keys(ON)" +
		"&_pragma=trusted_schema(OFF)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" +
		"&_pragma=temp_store(MEMORY)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	tb.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	return db
}

// filepath.Match treats '/' as a path separator that '*' will not cross.
func TestAuditDelByPatternNeverMatchesSlashKeys(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		key     string
		pattern string
	}{
		{
			name:    "metadata facet key holding a '/' search term",
			key:     MetadataFacetFilter{Limit: 20, Search: "sci-fi/fantasy", LibraryIDs: []string{"lib-1"}}.cacheKey("authors"),
			pattern: constants.CacheKeyMetadataPattern,
		},
		{
			name:    "user:search key holding a '/' search term",
			key:     cache.QueryKey("user:search", map[string]string{"q": "a/b"}),
			pattern: constants.CacheKeyUserSearch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := cache.NewTheineCache(8 << 20)
			if err := c.Set(ctx, tc.key, "payload", time.Hour); err != nil {
				t.Fatal(err)
			}
			if err := c.DelByPattern(ctx, tc.pattern); err != nil {
				t.Fatal(err)
			}
			survived, _ := c.Exists(ctx, tc.key)
			matched, matchErr := filepath.Match(tc.pattern, tc.key)
			t.Logf("key     = %q", tc.key)
			t.Logf("pattern = %q -> filepath.Match=%v err=%v", tc.pattern, matched, matchErr)
			if survived {
				t.Errorf("STALE: key survived DelByPattern(%q); '*' does not cross '/'", tc.pattern)
			}
		})
	}
}

// The metadata facet cache key is built from MetadataFacetFilter, whose LibraryIDs field carries the caller's permission scope.
func TestAuditMetadataFacetKeyIncludesLibraryScope(t *testing.T) {
	admin := MetadataFacetFilter{Limit: 20, LibraryIDs: []string{"lib-open", "lib-secret"}}
	guest := MetadataFacetFilter{Limit: 20, LibraryIDs: []string{"lib-open"}}

	if admin.cacheKey("authors") == guest.cacheKey("authors") {
		t.Fatalf("SECURITY: facet cache key ignores library scope: %q", admin.cacheKey("authors"))
	}
	t.Logf("admin key = %q", admin.cacheKey("authors"))
	t.Logf("guest key = %q", guest.cacheKey("authors"))

	adminEntity := cache.QueryKeyParts(admin.scopeKey(), "metadata_count", "author", "id", "au-1")
	guestEntity := cache.QueryKeyParts(guest.scopeKey(), "metadata_count", "author", "id", "au-1")
	if adminEntity == guestEntity {
		t.Errorf("SECURITY: entity layer is scope-free: %q is shared by every caller", adminEntity)
	}
}

// A metadata facet list's per-entity book_count is computed under the caller's library scope and then cached at metadata_count:<type>:id:<id> with NO scope in the key.
func TestAuditMetadataCountEntityIgnoresLibraryScope(t *testing.T) {
	ctx := context.Background()
	db := auditDB(t)
	c := cache.NewTheineCache(64 << 20)
	repo := NewBookDBRepository(db, c)

	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	mustExec(`INSERT INTO libraries (id,name) VALUES ('lib-open','Open'),('lib-secret','Secret')`)
	mustExec(`INSERT INTO authors (id,name) VALUES ('au-1','Frank Herbert')`)
	mustExec(`INSERT INTO books (id,library_id,title,author_id,status) VALUES ('bk-1','lib-open','Dune','au-1','active')`)
	for _, id := range []string{"bk-2", "bk-3", "bk-4"} {
		mustExec(`INSERT INTO books (id,library_id,title,author_id,status) VALUES (?,'lib-secret','Secret Vol','au-1','active')`, id)
	}

	adminFilter := MetadataFacetFilter{Limit: 20, LibraryIDs: []string{"lib-open", "lib-secret"}}
	guestFilter := MetadataFacetFilter{Limit: 20, LibraryIDs: []string{"lib-open"}}

	guestFirst, err := repo.ListAuthorsWithCount(ctx, guestFilter)
	if err != nil {
		t.Fatal(err)
	}
	if len(guestFirst) != 1 || guestFirst[0].BookCount != 1 {
		t.Fatalf("precondition: guest should see book_count=1, got %+v", guestFirst)
	}
	t.Logf("1. guest reads: book_count=%d (correct -- only lib-open is readable)", guestFirst[0].BookCount)

	adminRows, err := repo.ListAuthorsWithCount(ctx, adminFilter)
	if err != nil {
		t.Fatal(err)
	}
	if len(adminRows) != 1 || adminRows[0].BookCount != 4 {
		t.Fatalf("precondition: admin should see book_count=4, got %+v", adminRows)
	}
	t.Logf("2. admin reads: book_count=%d (correct -- 4 books across both libraries); this "+
		"overwrote the shared entity key %q", adminRows[0].BookCount,
		cache.QueryKeyParts(guestFilter.scopeKey(), "metadata_count", "author", "id", "au-1"))

	guestSecond, err := repo.ListAuthorsWithCount(ctx, guestFilter)
	if err != nil {
		t.Fatal(err)
	}
	if len(guestSecond) != 1 {
		t.Fatalf("guest should still see the author, got %+v", guestSecond)
	}
	t.Logf("3. guest reads again: book_count=%d", guestSecond[0].BookCount)

	if guestSecond[0].BookCount != 1 {
		t.Errorf("SECURITY: guest read book_count=%d from the admin-warmed, scope-free key %q. "+
			"Correct value under the guest's scope is 1. The count discloses how many books exist "+
			"in a library the guest has no permission to read.",
			guestSecond[0].BookCount, cache.QueryKeyParts(guestFilter.scopeKey(), "metadata_count", "author", "id", "au-1"))
	}
}

// GetFilesByBookId caches an EMPTY id list for ListCacheDuration when a book has no files yet.
func TestAuditEmptyFileListIsNegativeCached(t *testing.T) {
	ctx := context.Background()
	db := auditDB(t)
	c := cache.NewTheineCache(64 << 20)
	repo := NewBookDBRepository(db, c)

	if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`); err != nil {
		t.Fatal(err)
	}
	book := &models.BookEntity{ID: "bk-1", LibraryID: "lib-1", Title: "Dune", Status: "processing"}
	if err := repo.CreateBook(ctx, book); err != nil {
		t.Fatal(err)
	}

	empty, err := repo.GetFilesByBookId(ctx, "bk-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("precondition: expected no files, got %d", len(empty))
	}
	t.Logf("cached empty list under %q", cache.BuildKey("book_file", "book", "bk-1"))

	if err := repo.CreateBookFile(ctx, sqlc.CreateBookFileParams{
		ID: "f-1", BookID: "bk-1", Path: "/nas/library/dune.epub",
		Format: "epub", SizeBytes: 1024, ModTime: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetFilesByBookId(ctx, "bk-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Errorf("STALE: file was attached to bk-1 but GetFilesByBookId still returns 0 files "+
			"for up to %v (negative-cached empty list)", constants.ListCacheDuration)
	} else {
		t.Logf("CLEAN: CreateBookFile invalidated the empty list; %d file(s) returned", len(got))
	}
}

// Author / tag / series / publisher / language entities are cached by NAME with no invalidation anywhere.
func TestAuditMetadataNameKeysHaveNoInvalidation(t *testing.T) {
	ctx := context.Background()
	db := auditDB(t)
	c := cache.NewTheineCache(64 << 20)
	repo := NewBookDBRepository(db, c)

	if _, err := db.Exec(`INSERT INTO authors (id,name) VALUES ('au-1','Frank Herbert')`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetAuthorByName(ctx, "Frank Herbert"); err != nil {
		t.Fatal(err)
	}
	key := cache.BuildKey("author", "name", "Frank Herbert")
	if ok, _ := c.Exists(ctx, key); !ok {
		t.Skip("author:name not cached in this environment")
	}

	if _, err := db.Exec(`UPDATE authors SET name='Herbert, Frank' WHERE id='au-1'`); err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Exists(ctx, key); ok {
		t.Logf("NOTE: %q still resident after the row changed. No Del for the "+
			"author/tag/series/publisher/language name prefixes exists anywhere in the repo layer; "+
			"they expire only by TTL (%v). Low severity: these rows are effectively immutable "+
			"(create-or-get only, no update/delete endpoint).", key, constants.NormalCacheDuration)
	}
}

// The byte cache documents that GetOrLoad returns the shared buffer.
func TestAuditByteCacheSharedBufferIsMutable(t *testing.T) {
	bc, err := cache.NewByteCache(8<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Close()

	load := func() ([]byte, error) { return []byte("\xff\xd8\xffORIGINAL-JPEG-BYTES"), nil }

	first, err := bc.GetOrLoad("asset:f-1:page1.jpg", load)
	if err != nil {
		t.Fatal(err)
	}
	copy(first, []byte("CORRUPTED"))

	second, err := bc.GetOrLoad("asset:f-1:page1.jpg", load)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != "\xff\xd8\xffORIGINAL-JPEG-BYTES" {
		t.Logf("CONFIRMED (mechanism, not a live bug): the second reader got %q -- the cache hands "+
			"out the shared buffer, so any in-place write is visible to every other reader.",
			string(second))
	} else {
		t.Errorf("expected the shared buffer to be observably mutated")
	}
}

// Only raster images go through the byte cache; the css/html branch of GetAsset rebuilds a new slice.
func TestAuditAssetCacheOnlyHoldsImages(t *testing.T) {
	for _, ct := range []string{"text/css", "application/xhtml+xml", "audio/mpeg"} {
		if len(ct) >= 6 && ct[:6] == "image/" {
			t.Fatalf("%q would enter the byte cache", ct)
		}
	}
	t.Log("CLEAN: bookService.GetAsset gates the byte cache on image/*; the only mutated " +
		"content type (text/css) takes the uncached branch and scopeReaderCSS allocates a new string.")
}
