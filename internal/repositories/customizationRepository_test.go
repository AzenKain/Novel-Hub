package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestCustomizationRepositoryCRUDAndCacheByIDs(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test_custom.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	repo := NewCustomizationRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider) VALUES ('u1', 'test@example.com', 'LOCAL')`); err != nil {
		t.Fatal(err)
	}

	createdSound1, err := repo.CreateSoundscape(ctx, sqlc.CreateSoundscapeParams{
		ID:       "s1",
		UserID:   sql.NullString{String: "u1", Valid: true},
		Name:     "Rain",
		Category: "rain",
		FilePath: "/path/to/rain.mp3",
		Icon:     "Rain",
		Volume:   0.6,
		IsSystem: 0,
	})
	if err != nil {
		t.Fatalf("CreateSoundscape failed: %v", err)
	}
	if createdSound1.Name != "Rain" {
		t.Fatalf("expected Name Rain, got %s", createdSound1.Name)
	}

	createdSound2, err := repo.CreateSoundscape(ctx, sqlc.CreateSoundscapeParams{
		ID:       "s2",
		UserID:   sql.NullString{String: "u1", Valid: true},
		Name:     "Ocean",
		Category: "waves",
		FilePath: "/path/to/ocean.mp3",
		Icon:     "Waves",
		Volume:   0.7,
		IsSystem: 1,
	})
	if err != nil {
		t.Fatalf("CreateSoundscape failed: %v", err)
	}
	_ = createdSound2

	fetchedSound, err := repo.GetSoundscapeByID(ctx, "s1")
	if err != nil || fetchedSound == nil {
		t.Fatalf("GetSoundscapeByID failed: %v", err)
	}

	batchSounds, err := repo.GetSoundscapesByIDs(ctx, []string{"s1", "s2"})
	if err != nil || len(batchSounds) != 2 {
		t.Fatalf("GetSoundscapesByIDs expected 2, got %d", len(batchSounds))
	}

	uID := "u1"
	soundList, err := repo.ListSoundscapes(ctx, &uID, nil, "", 10)
	if err != nil || len(soundList) != 2 {
		t.Fatalf("ListSoundscapes expected 2, got %d", len(soundList))
	}

	if err := repo.DeleteSoundscape(ctx, "s1"); err != nil {
		t.Fatalf("DeleteSoundscape failed: %v", err)
	}
	deletedSound, _ := repo.GetSoundscapeByID(ctx, "s1")
	if deletedSound != nil {
		t.Fatalf("expected nil after delete, got %v", deletedSound)
	}
	if err := repo.DeleteSoundscape(ctx, "s2"); err != nil {
		t.Fatalf("DeleteSoundscape failed: %v", err)
	}

	createdFont, err := repo.CreateCustomFont(ctx, sqlc.CreateCustomFontParams{
		ID:         "f1",
		UserID:     sql.NullString{String: "u1", Valid: true},
		Name:       "Bookerly",
		FontFamily: "Bookerly",
		SourceType: "file",
		FilePath:   "/path/to/bookerly.woff2",
		FontUrl:    "",
		IsSystem:   0,
	})
	if err != nil {
		t.Fatalf("CreateCustomFont failed: %v", err)
	}
	if createdFont.Name != "Bookerly" {
		t.Fatalf("expected font name Bookerly, got %s", createdFont.Name)
	}

	fontList, err := repo.ListCustomFonts(ctx, &uID, nil, "", 10)
	if err != nil || len(fontList) != 1 {
		t.Fatalf("ListCustomFonts expected 1, got %d", len(fontList))
	}

	batchFonts, err := repo.GetCustomFontsByIDs(ctx, []string{"f1"})
	if err != nil || len(batchFonts) != 1 {
		t.Fatalf("GetCustomFontsByIDs expected 1, got %d", len(batchFonts))
	}

	if err := repo.DeleteCustomFont(ctx, "f1"); err != nil {
		t.Fatalf("DeleteCustomFont failed: %v", err)
	}

	createdTheme, err := repo.CreateCustomTheme(ctx, sqlc.CreateCustomThemeParams{
		ID:          "t1",
		UserID:      sql.NullString{String: "u1", Valid: true},
		Name:        "Cyberpunk",
		BgColor:     "#1a1b26",
		TextColor:   "#c0caf5",
		AccentColor: "#7aa2f7",
		CustomCss:   "",
		IsSystem:    0,
	})
	if err != nil {
		t.Fatalf("CreateCustomTheme failed: %v", err)
	}
	if createdTheme.Name != "Cyberpunk" {
		t.Fatalf("expected theme name Cyberpunk, got %s", createdTheme.Name)
	}

	themeList, err := repo.ListCustomThemes(ctx, &uID, nil, "", 10)
	if err != nil || len(themeList) != 1 {
		t.Fatalf("ListCustomThemes expected 1, got %d", len(themeList))
	}

	batchThemes, err := repo.GetCustomThemesByIDs(ctx, []string{"t1"})
	if err != nil || len(batchThemes) != 1 {
		t.Fatalf("GetCustomThemesByIDs expected 1, got %d", len(batchThemes))
	}

	if err := repo.DeleteCustomTheme(ctx, "t1"); err != nil {
		t.Fatalf("DeleteCustomTheme failed: %v", err)
	}
}
