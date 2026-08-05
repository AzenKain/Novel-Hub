package services

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
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
		{name: "valid rate limits", raw: map[string]any{
			"limits.rate_limit_auth":                float64(constants.HardMaxRateLimitAuth),
			"limits.rate_limit_auth_window_seconds": float64(constants.MinRuntimeRateLimitAuthWindowSeconds),
		}, ok: true},
		{name: "rate limit below floor rejected", raw: map[string]any{"limits.rate_limit_auth": float64(constants.MinRuntimeRateLimitAuth - 1)}},
		{name: "rate limit above ceiling rejected", raw: map[string]any{"limits.rate_limit_auth": float64(constants.HardMaxRateLimitAuth + 1)}},
		{name: "zero rate limit window rejected", raw: map[string]any{"limits.rate_limit_auth_window_seconds": float64(0)}},
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
	if !public.EnableAniListTracking {
		t.Fatalf("anilist tracking should default to true, got %#v", public)
	}

	updated, err := service.UpdateSettings(ctx, map[string]any{
		"tracker.anilist_enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.EnableAniListTracking {
		t.Fatalf("anilist tracking admin update did not take: %#v", updated)
	}
	public2, err := service.Public(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if public2.EnableAniListTracking {
		t.Fatalf("anilist tracking public read did not reflect update: %#v", public2)
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

func TestSettingsServiceSMTPPasswordLifecycle(t *testing.T) {
	t.Setenv("DB_ENCRYPTION_KEY", "settings-smtp-test-key")
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

	if _, err := service.SMTP(ctx); err == nil {
		t.Fatal("SMTP config should be refused while disabled")
	}

	admin, err := service.UpdateSettings(ctx, map[string]any{
		"smtp.enabled":                true,
		"smtp.host":                   "smtp.example.com",
		"smtp.port":                   465,
		"smtp.username":               "postmaster@example.com",
		"smtp.password":               "s3cret-app-password",
		"smtp.from_email":             "library@example.com",
		"smtp.tls_mode":               "implicit_tls",
		"smtp.allow_private_networks": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !admin.SMTP.PasswordConfigured || admin.SMTP.Port != 465 || admin.SMTP.TLSMode != "implicit_tls" {
		t.Fatalf("admin SMTP view did not reflect the update: %#v", admin.SMTP)
	}

	body, err := jsonx.Marshal(admin)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "s3cret-app-password") {
		t.Fatal("admin settings payload leaked the SMTP password")
	}

	stored, err := repo.Get(ctx, "smtp.password")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.ValueJSON, "s3cret-app-password") || !strings.Contains(stored.ValueJSON, "enc:v1:") {
		t.Fatalf("SMTP password was not stored encrypted: %s", stored.ValueJSON)
	}

	config, err := service.SMTP(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if config.Password != "s3cret-app-password" || config.Host != "smtp.example.com" {
		t.Fatalf("runtime SMTP config did not decrypt: %#v", config)
	}

	unrelated, err := service.UpdateSettings(ctx, map[string]any{"smtp.username": "renamed@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !unrelated.SMTP.PasswordConfigured {
		t.Fatal("omitting the password key cleared the stored credential")
	}

	cleared, err := service.UpdateSettings(ctx, map[string]any{"smtp.password": ""})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.SMTP.PasswordConfigured {
		t.Fatal("empty password did not clear the stored credential")
	}
}

func TestSettingsServiceRejectsIncompleteSMTP(t *testing.T) {
	t.Setenv("DB_ENCRYPTION_KEY", "settings-smtp-test-key")
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

	cases := []struct {
		name     string
		settings map[string]any
	}{
		{"enabled without host", map[string]any{"smtp.enabled": true, "smtp.from_email": "a@example.com"}},
		{"enabled without sender", map[string]any{"smtp.enabled": true, "smtp.host": "smtp.example.com"}},
		{"bad sender", map[string]any{"smtp.from_email": "not-an-email"}},
		{"host as url", map[string]any{"smtp.host": "https://smtp.example.com/submit"}},
		{"header injection", map[string]any{"smtp.from_email": "a@example.com\r\nBcc: victim@example.com"}},
		{"port out of range", map[string]any{"smtp.port": 70000}},
		{"unknown tls mode", map[string]any{"smtp.tls_mode": "ssl"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.UpdateSettings(ctx, test.settings); err == nil {
				t.Fatalf("invalid SMTP settings were accepted: %#v", test.settings)
			}
		})
	}

	admin, err := service.Admin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if admin.SMTP.Enabled || admin.SMTP.Host != "" {
		t.Fatalf("rejected SMTP settings leaked into the snapshot: %#v", admin.SMTP)
	}
}
