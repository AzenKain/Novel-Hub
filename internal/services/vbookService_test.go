package services

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestVBookPaginationNextPageDetection(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "vbook.db"))
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

	// Insert 25 books into database
	for i := 0; i < 25; i++ {
		if _, err := db.Exec(
			`INSERT INTO books (id, library_id, title, status, created_at) VALUES (?, 'lib-1', ?, 'active', datetime('now', ?))`,
			fmt.Sprintf("vbook-%02d", i), fmt.Sprintf("VBook %02d", i), fmt.Sprintf("-%d minutes", i),
		); err != nil {
			t.Fatal(err)
		}
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permissionCache := NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	settingsService := NewSettingsService(repositories.NewSettingsRepository(db, ramCache), database.NewTxManager(db), permissionCache)
	bookService := NewBookService(bookRepo, nil, nil, nil, nil, database.NewTxManager(db), settingsService, permissionCache, nil, nil)
	vbookSvc := NewVBookService(bookRepo, bookRepo, bookService, nil, ramCache)

	ctx := context.Background()
	const baseURL = "http://localhost:3434"

	// Page 1: limit 20. Expect 20 items and Next = "2"
	res1, err := vbookSvc.GetBooks(ctx, baseURL, nil, "", "", "", 1, 20)
	if err != nil {
		t.Fatalf("GetBooks page 1 failed: %v", err)
	}
	if len(res1.List) != 20 {
		t.Fatalf("Page 1 list len = %d, want 20", len(res1.List))
	}
	if res1.Next == nil || *res1.Next != "2" {
		t.Fatalf("Page 1 Next = %v, want '2'", res1.Next)
	}

	// Page 2: limit 20. Expect 5 items and Next = nil
	res2, err := vbookSvc.GetBooks(ctx, baseURL, nil, "", "", "", 2, 20)
	if err != nil {
		t.Fatalf("GetBooks page 2 failed: %v", err)
	}
	if len(res2.List) != 5 {
		t.Fatalf("Page 2 list len = %d, want 5", len(res2.List))
	}
	if res2.Next != nil {
		t.Fatalf("Page 2 Next = %v, want nil", *res2.Next)
	}
}
