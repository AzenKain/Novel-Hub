package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

type customPermissionStub struct {
	admin   bool
	allowed bool
}

func (p *customPermissionStub) Reload(context.Context) error { return nil }
func (p *customPermissionStub) Can(context.Context, string, string, map[string]any) bool {
	return p.allowed || p.admin
}
func (p *customPermissionStub) CanRoles(roleIDs []string, roles []constants.RoleType, perm string, attrs map[string]any) bool {
	if p.admin {
		return true
	}
	if strings.HasPrefix(perm, "admin.") {
		return false
	}
	return p.allowed
}
func (p *customPermissionStub) IsAdmin(roleIDs []string, roles []constants.RoleType) bool { return p.admin }
func (p *customPermissionStub) GetGuestPermissions() []string               { return nil }
func (p *customPermissionStub) DescribeRoles(roleIDs []string) []*models.RoleSimple { return nil }

func TestCustomizationServiceSoundscapesAndFonts(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(tempDir, "test_custom_svc.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	// Seed test user
	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider) VALUES ('u1', 'test@example.com', 'LOCAL')`); err != nil {
		t.Fatal(err)
	}

	repo := repositories.NewCustomizationRepository(db, cache.NewRamCache())
	permStub := &customPermissionStub{admin: false, allowed: true}
	svc := NewCustomizationService(repo, permStub, nil, tempDir)

	ctx := context.Background()
	userClaims := &response.JWTClaims{
		UId:   "u1",
		Roles: []constants.RoleType{constants.RoleTypeUser},
	}

	// 1. Upload Soundscape with file bytes
	audioData := []byte("ID3_FAKE_MP3_AUDIO_STREAM_DATA")
	sound, err := svc.CreateSoundscape(ctx, userClaims, &request.UploadSoundscapeDto{
		Name:     "Rain On Leaves",
		Category: "rain",
		Icon:     "Rain",
		Volume:   0.8,
	}, audioData, "rain.mp3")
	if err != nil {
		t.Fatalf("CreateSoundscape failed: %v", err)
	}
	if sound.Name != "Rain On Leaves" {
		t.Fatalf("expected Name 'Rain On Leaves', got '%s'", sound.Name)
	}

	// Security: Attempt to upload fake audio / malware should fail
	if _, err := svc.CreateSoundscape(ctx, userClaims, &request.UploadSoundscapeDto{Name: "Bad"}, []byte("MZ_WINDOWS_EXE"), "hack.exe"); err == nil {
		t.Fatal("expected error on .exe upload as soundscape, got nil")
	}
	if _, err := svc.CreateSoundscape(ctx, userClaims, &request.UploadSoundscapeDto{Name: "Bad"}, []byte("MZ_WINDOWS_EXE"), "hack.mp3"); err == nil {
		t.Fatal("expected error on fake mp3 magic header, got nil")
	}

	// List with cursor pagination
	sounds, nextCursor, err := svc.ListSoundscapes(ctx, userClaims, nil, "", 20)
	if err != nil || len(sounds) != 1 {
		t.Fatalf("ListSoundscapes expected 1, got %d, err: %v", len(sounds), err)
	}
	_ = nextCursor

	// Get file path
	path, name, err := svc.GetSoundscapeFilePath(ctx, sound.ID, userClaims)
	if err != nil || path == "" || name != "Rain On Leaves" {
		t.Fatalf("GetSoundscapeFilePath failed: %v, path: %s", err, path)
	}

	// Delete
	if err := svc.DeleteSoundscape(ctx, sound.ID, userClaims); err != nil {
		t.Fatalf("DeleteSoundscape failed: %v", err)
	}

	// 2. Custom Fonts
	fontData := []byte("wOF2_FAKE_WOFF2_FONT_DATA")
	font, err := svc.CreateCustomFont(ctx, userClaims, &request.UploadFontDto{
		Name:       "Bookerly Serif",
		FontFamily: "Bookerly",
		SourceType: "file",
	}, fontData, "bookerly.woff2")
	if err != nil {
		t.Fatalf("CreateCustomFont failed: %v", err)
	}
	if font.Name != "Bookerly Serif" {
		t.Fatalf("expected Font Name 'Bookerly Serif', got '%s'", font.Name)
	}

	// Security: Attempt to upload fake font / malware should fail
	if _, err := svc.CreateCustomFont(ctx, userClaims, &request.UploadFontDto{Name: "Bad", SourceType: "file"}, []byte("ELF_LINUX_BINARY"), "hack.woff2"); err == nil {
		t.Fatal("expected error on fake woff2 magic header, got nil")
	}

	fonts, _, err := svc.ListCustomFonts(ctx, userClaims, nil, "", 20)
	if err != nil || len(fonts) != 1 {
		t.Fatalf("ListCustomFonts expected 1, got %d", len(fonts))
	}

	if err := svc.DeleteCustomFont(ctx, font.ID, userClaims); err != nil {
		t.Fatalf("DeleteCustomFont failed: %v", err)
	}

	// 3. Custom Themes
	theme, err := svc.CreateCustomTheme(ctx, userClaims, &request.CreateCustomThemeDto{
		Name:        "Cyberpunk Neon",
		BgColor:     "#0f0f17",
		TextColor:   "#a9b1d6",
		AccentColor: "#f7768e",
		CustomCss:   ".reader-container { padding: 20px; }",
	})
	if err != nil {
		t.Fatalf("CreateCustomTheme failed: %v", err)
	}
	if theme.Name != "Cyberpunk Neon" {
		t.Fatalf("expected Theme Name 'Cyberpunk Neon', got '%s'", theme.Name)
	}

	// Security: XSS injection in CustomCss should fail
	if _, err := svc.CreateCustomTheme(ctx, userClaims, &request.CreateCustomThemeDto{
		Name:      "XSS Theme",
		CustomCss: "<script>alert(1)</script>",
	}); err == nil {
		t.Fatal("expected error on XSS script in CustomCss, got nil")
	}

	themes, _, err := svc.ListCustomThemes(ctx, userClaims, nil, "", 20)
	if err != nil || len(themes) != 1 {
		t.Fatalf("ListCustomThemes expected 1, got %d", len(themes))
	}

	// Update Theme
	updatedTheme, err := svc.UpdateCustomTheme(ctx, theme.ID, userClaims, &request.UpdateCustomThemeDto{
		Name:        "Cyberpunk Neon 2077",
		BgColor:     "#0a0a10",
		TextColor:   "#c0caf5",
		AccentColor: "#bb9af7",
	})
	if err != nil || updatedTheme.Name != "Cyberpunk Neon 2077" {
		t.Fatalf("UpdateCustomTheme failed: %v", err)
	}

	// Get Theme
	gotTheme, err := svc.GetCustomTheme(ctx, theme.ID, userClaims)
	if err != nil || gotTheme.Name != "Cyberpunk Neon 2077" {
		t.Fatalf("GetCustomTheme failed: %v", err)
	}

	// Other user cannot get private theme
	otherClaims := &response.JWTClaims{UId: "u2", Roles: []constants.RoleType{constants.RoleTypeUser}}
	if _, err := svc.GetCustomTheme(ctx, theme.ID, otherClaims); err == nil {
		t.Fatal("expected error when other user gets private theme, got nil")
	}

	// Delete Theme
	if err := svc.DeleteCustomTheme(ctx, theme.ID, userClaims); err != nil {
		t.Fatalf("DeleteCustomTheme failed: %v", err)
	}
}
