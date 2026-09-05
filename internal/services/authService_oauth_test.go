package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newOAuthTestAuthService(t *testing.T) (AuthService, SettingsService, repositories.UserRepository, repositories.RoleRepository) {
	t.Helper()
	t.Setenv("JWT_SECRET", "oauth-test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "oauth-test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "oauth-test-encryption-key-32bytes-long-!!")
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "oauth_test.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	settingsRepo := repositories.NewSettingsRepository(db, c)
	settings := NewSettingsService(settingsRepo, database.NewTxManager(db))
	if err := settings.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}

	userRepo := repositories.NewUserRepository(db, c)
	roleRepo := repositories.NewRoleRepository(db, c)
	store := NewOTPStore(c)

	svc := NewAuthService(
		userRepo,
		roleRepo,
		database.NewTxManager(db),
		settingsRepo,
		settings,
		store,
	)
	return svc, settings, userRepo, roleRepo
}

func TestSigninOrRegisterOAuth_SuccessRegistration(t *testing.T) {
	svc, settings, userRepo, _ := newOAuthTestAuthService(t)
	ctx := context.Background()

	_, err := settings.UpdateSettings(ctx, map[string]any{
		"auth.registration_enabled": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := svc.SigninOrRegisterOAuth(ctx, "GOOGLE", "newuser@example.com", "John Doe", "https://avatar.url/john.png", "google-sub-12345")
	if err != nil {
		t.Fatalf("Failed to signin or register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("Expected access and refresh tokens to be returned")
	}

	user, err := userRepo.GetAuthByEmail(ctx, "newuser@example.com")
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}
	if user.Email != "newuser@example.com" {
		t.Errorf("Expected email 'newuser@example.com', got %q", user.Email)
	}
	if user.FullName != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %q", user.FullName)
	}
	if user.AvatarUrl != "https://avatar.url/john.png" {
		t.Errorf("Expected avatar 'https://avatar.url/john.png', got %q", user.AvatarUrl)
	}
	if user.AuthProvider != "GOOGLE" {
		t.Errorf("Expected provider 'GOOGLE', got %q", user.AuthProvider)
	}
}

func TestSigninOrRegisterOAuth_RegistrationDisabled(t *testing.T) {
	svc, settings, _, _ := newOAuthTestAuthService(t)
	ctx := context.Background()

	_, err := settings.UpdateSettings(ctx, map[string]any{
		"auth.registration_enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.SigninOrRegisterOAuth(ctx, "GITHUB", "stranger@example.com", "Stranger", "", "github-id-54321")
	if err == nil {
		t.Fatal("Expected registration to fail when registration is disabled")
	}
	if !errors.Is(err, apperrors.ErrForbidden) || !strings.Contains(err.Error(), "registration is disabled") {
		t.Fatalf("Expected forbidden error, got: %v", err)
	}
}

func TestSigninOrRegisterOAuth_SignInExistingAndUpdateProfile(t *testing.T) {
	svc, settings, userRepo, _ := newOAuthTestAuthService(t)
	ctx := context.Background()

	_, _ = settings.UpdateSettings(ctx, map[string]any{"auth.registration_enabled": true})
	_, err := svc.SigninOrRegisterOAuth(ctx, "DISCORD", "user@example.com", "Old Name", "https://avatar.url/old.png", "discord-id-9999")
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := svc.SigninOrRegisterOAuth(ctx, "DISCORD", "user@example.com", "New Name", "https://avatar.url/new.png", "discord-id-9999")
	if err != nil {
		t.Fatalf("Failed to signin: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("Expected tokens to be returned")
	}

	user, err := userRepo.GetAuthByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.FullName != "New Name" {
		t.Errorf("Expected updated name 'New Name', got %q", user.FullName)
	}
	if user.AvatarUrl != "https://avatar.url/new.png" {
		t.Errorf("Expected updated avatar 'https://avatar.url/new.png', got %q", user.AvatarUrl)
	}
}

func TestSigninOrRegisterOAuth_BannedUser(t *testing.T) {
	svc, settings, userRepo, roleRepo := newOAuthTestAuthService(t)
	ctx := context.Background()

	_, _ = settings.UpdateSettings(ctx, map[string]any{"auth.registration_enabled": true})
	_, err := svc.SigninOrRegisterOAuth(ctx, "GOOGLE", "banned@example.com", "Banned User", "", "google-sub-banned")
	if err != nil {
		t.Fatal(err)
	}

	user, err := userRepo.GetAuthByEmail(ctx, "banned@example.com")
	if err != nil {
		t.Fatal(err)
	}

	err = roleRepo.CreateUserRole(ctx, user.ID, "01920000-0000-7000-8000-000000000004")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.SigninOrRegisterOAuth(ctx, "GOOGLE", "banned@example.com", "Banned User", "", "google-sub-banned")
	if err == nil {
		t.Fatal("Expected banned user login to fail")
	}
	if !errors.Is(err, apperrors.ErrForbidden) || !strings.Contains(err.Error(), "account is banned") {
		t.Fatalf("Expected forbidden/banned error, got: %v", err)
	}
}
