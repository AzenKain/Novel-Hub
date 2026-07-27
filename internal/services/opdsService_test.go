package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func TestOPDSService_GetOPDS2Catalog(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "novelhub_opds_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	schemaFiles := []string{
		"../../db/schema/10_auth.sql",
		"../../db/schema/15_libraries.sql",
		"../../db/schema/20_books.sql",
		"../../db/schema/25_metadata.sql",
		"../../db/schema/30_files_and_jobs.sql",
		"../../db/schema/45_metadata_fts.sql",
		"../../db/schema/50_user_features.sql",
		"../../db/schema/55_reading_activity.sql",
		"../../db/schema/65_permissions_settings.sql",
		"../../db/schema/90_seed_roles.sql",
		"../../db/schema/95_rbac_restructure.sql",
	}

	for _, sf := range schemaFiles {
		sqlBytes, err := os.ReadFile(sf)
		if err != nil {
			t.Fatalf("failed to read schema %s: %v", sf, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("failed to execute schema %s: %v", sf, err)
		}
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permissionCache := NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatalf("failed to load permissions: %v", err)
	}
	settingsService := NewSettingsService(settingsRepo, database.NewTxManager(db), permissionCache)
	bookService := NewBookService(bookRepo, nil, nil, nil, bookparser.NewRegistry(), database.NewTxManager(db), settingsService, permissionCache, nil)
	book := &models.BookEntity{
		ID:     "test-book-opds-2",
		Title:  "Test OPDS 2.0 Book Title",
		Status: "active",
	}
	if err := bookRepo.CreateBook(context.Background(), book); err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	opdsService := NewOPDSService(bookService, permissionCache)
	claims := &response.JWTClaims{UId: "0", Roles: []constants.RoleType{constants.RoleTypeGuest}}
	catalog, err := opdsService.GetOPDS2Catalog(context.Background(), "http://localhost:8080", claims)
	if err != nil {
		t.Fatalf("failed to get OPDS 2.0 catalog: %v", err)
	}

	metadata, ok := catalog["metadata"].(map[string]any)
	if !ok || metadata["title"] != "NovelHub OPDS 2.0 Catalog" {
		t.Errorf("unexpected metadata: %v", catalog["metadata"])
	}

	publications, ok := catalog["publications"].([]map[string]any)
	if !ok || len(publications) == 0 {
		t.Fatalf("expected publications in OPDS 2.0 catalog, got %v", catalog["publications"])
	}

	pubTitle := publications[0]["metadata"].(map[string]any)["title"]
	if pubTitle != "Test OPDS 2.0 Book Title" {
		t.Errorf("expected publication title 'Test OPDS 2.0 Book Title', got %v", pubTitle)
	}

	t.Logf("OPDS 2.0 Catalog test passed successfully!")
}
