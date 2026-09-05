package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newTestFeatureService(t *testing.T) (FeatureService, repositories.FeatureRepository, repositories.BookCatalogRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "feature_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	featureRepo := repositories.NewFeatureRepository(db, ramCache)
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	txManager := database.NewTxManager(db)

	svc := NewFeatureService(featureRepo, bookRepo, nil, nil, txManager)
	return svc, featureRepo, bookRepo, db
}

func TestSmartFilterLifecycleAndBooks(t *testing.T) {
	svc, _, bookRepo, db := newTestFeatureService(t)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO users (id, email, full_name, password_hash) VALUES ('user-1', 'test@example.com', 'Test User', 'hash')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main Lib')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, author_id, average_rating, created_at, updated_at) VALUES
		('book-1', 'lib-1', 'Book One', NULL, 4.5, '2026-08-11 00:00:00', '2026-08-11 00:00:00'),
		('book-2', 'lib-1', 'Book Two', NULL, 3.2, '2026-08-11 00:00:00', '2026-08-11 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES
		('file-1', 'book-1', '/path/book1.epub', 'epub', 1000, '2026-08-11 00:00:00')`); err != nil {
		t.Fatal(err)
	}

	rules := []request.SmartFilterRuleItemDto{
		{Field: "format", Operator: "eq", Value: "epub"},
		{Field: "rating_gte", Operator: "gte", Value: "4.0"},
	}
	dto := request.UpsertSmartFilterDto{
		Name:            "High Rated EPUBs",
		Rules:           rules,
		IsPinnedSidebar: true,
		IsPinnedHome:    true,
	}

	sf, err := svc.CreateSmartFilter(ctx, "user-1", dto)
	if err != nil {
		t.Fatalf("CreateSmartFilter failed: %v", err)
	}
	if sf.Name != "High Rated EPUBs" || len(sf.Rules) != 2 || !sf.IsPinnedSidebar || !sf.IsPinnedHome || sf.HomePosition != 0 {
		t.Fatalf("Unexpected smart filter content: %+v", sf)
	}

	sfGet, err := svc.GetSmartFilter(ctx, sf.ID, "user-1")
	if err != nil {
		t.Fatalf("GetSmartFilter failed: %v", err)
	}
	if sfGet.ID != sf.ID {
		t.Fatalf("Expected ID %s, got %s", sf.ID, sfGet.ID)
	}

	dtoUpdate := request.UpsertSmartFilterDto{
		Name:            "Updated Name",
		Rules:           []request.SmartFilterRuleItemDto{{Field: "format", Operator: "eq", Value: "epub"}},
		IsPinnedSidebar: false,
		IsPinnedHome:    false,
	}
	sfUpd, err := svc.UpdateSmartFilter(ctx, sf.ID, "user-1", dtoUpdate)
	if err != nil {
		t.Fatalf("UpdateSmartFilter failed: %v", err)
	}
	if sfUpd.Name != "Updated Name" || len(sfUpd.Rules) != 1 || sfUpd.IsPinnedSidebar || sfUpd.IsPinnedHome {
		t.Fatalf("Unexpected updated smart filter: %+v", sfUpd)
	}

	sfPinS, err := svc.UpdateSmartFilterPinSidebar(ctx, sf.ID, "user-1", true)
	if err != nil || !sfPinS.IsPinnedSidebar {
		t.Fatalf("UpdateSmartFilterPinSidebar failed: %v", err)
	}
	sfPinH, err := svc.UpdateSmartFilterPinHome(ctx, sf.ID, "user-1", true)
	if err != nil || !sfPinH.IsPinnedHome {
		t.Fatalf("UpdateSmartFilterPinHome failed: %v", err)
	}

	books, err := bookRepo.SearchSmartFilterBooks(ctx, nil, rules, nil, "", 20, "user-1")
	if err != nil {
		t.Fatalf("SearchSmartFilterBooks failed: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("Expected 1 matching book, got %d", len(books))
	}
	if books[0].ID != "book-1" {
		t.Fatalf("Expected book-1, got %s", books[0].ID)
	}

	sf2, err := svc.CreateSmartFilter(ctx, "user-1", request.UpsertSmartFilterDto{
		Name:  "Second Filter",
		Rules: []request.SmartFilterRuleItemDto{},
	})
	if err != nil {
		t.Fatal(err)
	}

	reorderDto := request.ReorderHomeShelvesDto{
		Shelves: []request.ReorderHomeShelfItemDto{
			{ID: sf.ID, Position: 5},
			{ID: sf2.ID, Position: 2},
		},
	}
	err = svc.ReorderSmartFiltersHome(ctx, "user-1", reorderDto)
	if err != nil {
		t.Fatalf("ReorderSmartFiltersHome failed: %v", err)
	}

	filters, err := svc.ListSmartFilters(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(filters) != 2 {
		t.Fatalf("Expected 2 filters, got %d", len(filters))
	}
	if filters[0].ID != sf2.ID || filters[1].ID != sf.ID {
		t.Fatalf("Unexpected sort order: first=%s, second=%s", filters[0].Name, filters[1].Name)
	}

	bookSvc := newTestBookService(t, db)
	dtoSearch := &request.SearchBookDto{}
	dtoSearch.Limit = 10
	res, err := bookSvc.SearchSmartFilterBooksByFilter(ctx, sf.ID, "user-1", dtoSearch, &response.JWTClaims{UId: "user-1"})
	if err != nil {
		t.Fatalf("SearchSmartFilterBooksByFilter failed: %v", err)
	}
	if res == nil || !res.Status {
		t.Fatal("Expected non-nil successful paginated response")
	}

	err = svc.DeleteSmartFilter(ctx, sf.ID, "user-1")
	if err != nil {
		t.Fatalf("DeleteSmartFilter failed: %v", err)
	}
	_, err = svc.GetSmartFilter(ctx, sf.ID, "user-1")
	if err == nil {
		t.Fatal("Expected error getting deleted filter")
	}
}

func newTestBookService(_ *testing.T, db *sql.DB) BookService {
	ramCache := cache.NewRamCache()
	featureRepo := repositories.NewFeatureRepository(db, ramCache)
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	txManager := database.NewTxManager(db)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	settingsService := NewSettingsService(settingsRepo, txManager, allowAllPermissions{})
	return NewBookService(bookRepo, featureRepo, nil, nil, nil, txManager, settingsService, allowAllPermissions{}, nil, nil)
}
