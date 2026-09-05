package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/opds"
)

// Acquisition feeds used to return a hardcoded 50 books with no cursor and no rel="next", so a reader had no way to reach book 51 and no way to tell the feed had been truncated — the rest of the library was simply unreachable over OPDS.
func TestOPDSFeedPagesThroughEveryBook(t *testing.T) {
	svc, claims := newOPDSPagingService(t, 7)
	ctx := context.Background()
	const serverURL = "http://localhost:3434"
	const basePath = "/opds"

	seen := map[string]bool{}
	cursor := ""
	for page := 0; page < 10; page++ {
		feed, err := svc.GetRecentBooks(ctx, serverURL, basePath, request.OPDSPageDto{Limit: 3, Cursor: cursor}, claims)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range feed.Entries {
			if seen[entry.ID] {
				t.Fatalf("page %d repeated entry %s", page, entry.ID)
			}
			seen[entry.ID] = true
		}
		next := feedLink(feed.Links, "next")
		if next == "" {
			break
		}
		cursor = cursorFromHref(t, next)
		if cursor == "" {
			t.Fatalf("next link carries no cursor: %s", next)
		}
	}

	if len(seen) != 7 {
		t.Fatalf("paged through %d books, want all 7 — the tail of the library is unreachable", len(seen))
	}
}

// The last page must not advertise a next link, otherwise a reader loops forever asking for a page that is always empty.
func TestOPDSLastPageHasNoNextLink(t *testing.T) {
	svc, claims := newOPDSPagingService(t, 2)
	feed, err := svc.GetRecentBooks(context.Background(), "http://localhost:3434", "/opds", request.OPDSPageDto{Limit: 50}, claims)
	if err != nil {
		t.Fatal(err)
	}
	if got := feedLink(feed.Links, "next"); got != "" {
		t.Fatalf("short page still advertises next: %s", got)
	}
}

func feedLink(links []opds.Link, rel string) string {
	for _, l := range links {
		if l.Rel == rel {
			return l.Href
		}
	}
	return ""
}

func cursorFromHref(t *testing.T, href string) string {
	t.Helper()
	parsed, err := url.Parse(href)
	if err != nil {
		t.Fatalf("next link is not a URL: %s", href)
	}
	return parsed.Query().Get("cursor")
}

func newOPDSPagingService(t *testing.T, books int) (OPDSService, *response.JWTClaims) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "opds.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1','Lib')`); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permissionCache := NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	settingsService := NewSettingsService(repositories.NewSettingsRepository(db, ramCache), database.NewTxManager(db), permissionCache)
	bookService := NewBookService(bookRepo, nil, nil, nil, bookparser.NewRegistry(), database.NewTxManager(db), settingsService, permissionCache, nil, nil)
	metadataService := NewMetadataService(bookRepo, nil)

	for i := range books {
		if _, err := db.Exec(
			`INSERT INTO books (id, library_id, title, status, created_at) VALUES (?, 'lib-1', ?, 'active', datetime('now', ?))`,
			fmt.Sprintf("book-%02d", i), fmt.Sprintf("Book %02d", i), fmt.Sprintf("-%d minutes", i),
		); err != nil {
			t.Fatal(err)
		}
	}

	claims := &response.JWTClaims{UId: "admin", Roles: []constants.RoleType{constants.RoleTypeAdmin}}
	return NewOPDSService(bookService, metadataService, settingsService, permissionCache), claims
}
