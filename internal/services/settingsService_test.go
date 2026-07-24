package services

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func TestRuntimeLimitsFromRaw(t *testing.T) {
	defaults := defaultRuntimeLimits()
	tests := []struct {
		name string
		raw  map[string]any
		ok   bool
	}{
		{name: "defaults", raw: map[string]any{}, ok: true},
		{name: "valid integers", raw: map[string]any{
			"limits.upload_chunk_bytes":         float64(constants.MinRuntimeUploadChunkBytes),
			"limits.upload_chunks":              float64(constants.HardMaxUploadChunks),
			"limits.upload_sessions":            float64(constants.HardMaxUploadSessions),
			"limits.upload_bytes":               float64(constants.MinRuntimeUploadChunkBytes),
			"limits.upload_session_ttl_seconds": float64(constants.MinRuntimeUploadSessionTTL.Seconds()),
			"limits.cover_bytes":                float64(constants.HardMaxCoverBytes),
			"limits.site_asset_bytes":           float64(constants.HardMaxSiteAssetBytes),
		}, ok: true},
		{name: "string rejected", raw: map[string]any{"limits.upload_chunks": "256"}},
		{name: "boolean rejected", raw: map[string]any{"limits.upload_chunks": true}},
		{name: "fraction rejected", raw: map[string]any{"limits.upload_chunks": 2.5}},
		{name: "overflow rejected", raw: map[string]any{"limits.upload_bytes": math.Inf(1)}},
		{name: "hard ceiling rejected", raw: map[string]any{"limits.upload_chunks": float64(constants.HardMaxUploadChunks + 1)}},
		{name: "chunk exceeds total", raw: map[string]any{
			"limits.upload_chunk_bytes": float64(2 << 20),
			"limits.upload_bytes":       float64(1 << 20),
		}},
		{name: "total exceeds chunk capacity", raw: map[string]any{
			"limits.upload_chunk_bytes": float64(1 << 20),
			"limits.upload_chunks":      float64(2),
			"limits.upload_bytes":       float64(3 << 20),
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limits, err := runtimeLimitsFromRaw(test.raw)
			if (err == nil) != test.ok {
				t.Fatalf("runtimeLimitsFromRaw() error = %v", err)
			}
			if test.name == "defaults" && limits != defaults {
				t.Fatalf("defaults = %#v, want %#v", limits, defaults)
			}
		})
	}
}

func TestSettingsServiceRuntimeLimitsSnapshot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "settings.db")
	t.Setenv("SQLITE_DB_PATH", dbPath)
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatal(err)
	}

	repo := repositories.NewSettingsRepository(db, nil)
	service := NewSettingsService(repo, database.NewTxManager(db))
	ctx := context.Background()
	if err := service.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	if got := service.Limits().UploadBytes; got != constants.MaxUploadBytes {
		t.Fatalf("seeded upload bytes = %d, want %d", got, int64(constants.MaxUploadBytes))
	}

	admin, err := service.UpdateSettings(ctx, map[string]any{
		"limits.upload_chunk_bytes": int64(16 << 20),
		"limits.upload_chunks":      256,
		"limits.upload_bytes":       int64(4) << 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if admin.Limits.UploadBytes != int64(4)<<30 || service.Limits().UploadBytes != int64(4)<<30 {
		t.Fatalf("runtime snapshot was not updated after commit: %#v", admin.Limits)
	}
	if admin.Bounds.Max.UploadBytes != constants.HardMaxUploadBytes {
		t.Fatalf("admin bounds missing hard ceiling: %#v", admin.Bounds)
	}

	public, err := service.Public(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if public.Site.Title == "" {
		t.Fatal("public settings were not preserved")
	}

	before := service.Limits()
	if _, err := service.UpdateSettings(ctx, map[string]any{
		"site.title":          "must not persist",
		"limits.upload_bytes": int64(constants.HardMaxUploadBytes) + 1,
	}); err == nil {
		t.Fatal("invalid multi-key update succeeded")
	}
	if service.Limits() != before {
		t.Fatal("invalid update changed the RAM snapshot")
	}
	row, err := repo.Get(ctx, "site.title")
	if err != nil {
		t.Fatal(err)
	}
	if row.ValueJSON == `"must not persist"` {
		t.Fatal("invalid multi-key update partially persisted")
	}
}

func TestSettingsServiceReloadRejectsMalformedPersistedLimit(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "settings.db")
	t.Setenv("SQLITE_DB_PATH", dbPath)
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatal(err)
	}

	repo := repositories.NewSettingsRepository(db, nil)
	if err := repo.Upsert(context.Background(), "limits.upload_chunks", `"not-a-number"`); err != nil {
		t.Fatal(err)
	}
	service := NewSettingsService(repo, database.NewTxManager(db))
	if err := service.Reload(context.Background()); err == nil {
		t.Fatal("malformed persisted runtime limit did not fail reload")
	}
}
