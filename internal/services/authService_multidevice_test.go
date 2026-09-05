package services

import (
	"context"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestMultiDeviceRefreshTokenSessions(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "multidevice.db"))
	t.Setenv("JWT_SECRET", "test-secret-12345678901234567890")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret-12345678901234567890")

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("Password123@"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	userID := "01920000-0000-7000-8000-0000000000a1"
	if _, err := db.Exec(`INSERT INTO users (id, email, password_hash, auth_provider, token_version) VALUES (?, ?, ?, 'LOCAL', 1)`,
		userID, "user@example.com", string(hashed)); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	userRepo := repositories.NewUserRepository(db, c)
	svc := NewAuthService(
		userRepo,
		repositories.NewRoleRepository(db, c),
		database.NewTxManager(db),
		repositories.NewSettingsRepository(db, c),
		nil,
	)

	ctx := context.Background()

	// 1. Device A (e.g. PC) signs in
	resA, err := svc.Signin(ctx, &request.SignInDto{
		Email:    "user@example.com",
		Password: "Password123@",
	})
	if err != nil {
		t.Fatalf("Device A signin failed: %v", err)
	}

	// 2. Device B (e.g. Phone) signs in
	resB, err := svc.Signin(ctx, &request.SignInDto{
		Email:    "user@example.com",
		Password: "Password123@",
	})
	if err != nil {
		t.Fatalf("Device B signin failed: %v", err)
	}

	if resA.RefreshToken == resB.RefreshToken {
		t.Fatalf("expected different refresh tokens for separate signins")
	}

	// 3. Device A refreshes its token (should succeed even after Device B signed in!)
	refreshedA, err := svc.RefreshToken(ctx, userID, resA.RefreshToken)
	if err != nil {
		t.Fatalf("Device A refresh failed: %v", err)
	}
	if refreshedA.RefreshToken == "" || refreshedA.RefreshToken == resA.RefreshToken {
		t.Fatalf("expected rotated refresh token for Device A")
	}

	// 4. Device B refreshes its token (should also succeed!)
	refreshedB, err := svc.RefreshToken(ctx, userID, resB.RefreshToken)
	if err != nil {
		t.Fatalf("Device B refresh failed: %v", err)
	}
	if refreshedB.RefreshToken == "" || refreshedB.RefreshToken == resB.RefreshToken {
		t.Fatalf("expected rotated refresh token for Device B")
	}

	// 5. Replaying old token from Device A must fail
	_, err = svc.RefreshToken(ctx, userID, resA.RefreshToken)
	if err == nil {
		t.Fatalf("expected error when replaying old refresh token from Device A")
	}

	// 6. Device A refreshes again with its new rotated token (should succeed)
	refreshedA2, err := svc.RefreshToken(ctx, userID, refreshedA.RefreshToken)
	if err != nil {
		t.Fatalf("Device A second refresh failed: %v", err)
	}
	if refreshedA2.RefreshToken == "" {
		t.Fatalf("expected rotated refresh token for Device A")
	}

	// 7. Logout revokes sessions
	if err := svc.Logout(ctx, userID); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// 8. Subsequent refresh from any device must fail
	_, err = svc.RefreshToken(ctx, userID, refreshedA2.RefreshToken)
	if err == nil {
		t.Fatalf("expected error refreshing after logout")
	}
	_, err = svc.RefreshToken(ctx, userID, refreshedB.RefreshToken)
	if err == nil {
		t.Fatalf("expected error refreshing Device B after logout")
	}
}

func TestConcurrentMultiDeviceRefreshTokens(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "multidevice_concurrent.db"))
	t.Setenv("JWT_SECRET", "test-secret-12345678901234567890")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret-12345678901234567890")

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("Password123@"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	userID := "01920000-0000-7000-8000-0000000000a2"
	if _, err := db.Exec(`INSERT INTO users (id, email, password_hash, auth_provider, token_version) VALUES (?, ?, ?, 'LOCAL', 1)`,
		userID, "user2@example.com", string(hashed)); err != nil {
		t.Fatal(err)
	}

	c := cache.NewRamCache()
	userRepo := repositories.NewUserRepository(db, c)
	svc := NewAuthService(
		userRepo,
		repositories.NewRoleRepository(db, c),
		database.NewTxManager(db),
		repositories.NewSettingsRepository(db, c),
		nil,
	)

	ctx := context.Background()

	// 5 devices sign in sequentially
	tokens := make([]string, 5)
	for i := 0; i < 5; i++ {
		res, err := svc.Signin(ctx, &request.SignInDto{
			Email:    "user2@example.com",
			Password: "Password123@",
		})
		if err != nil {
			t.Fatalf("Signin device %d failed: %v", i, err)
		}
		tokens[i] = res.RefreshToken
	}

	// Now all 5 devices refresh concurrently
	type result struct {
		idx int
		err error
	}
	resChan := make(chan result, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			_, err := svc.RefreshToken(ctx, userID, tokens[idx])
			resChan <- result{idx: idx, err: err}
		}(i)
	}

	for i := 0; i < 5; i++ {
		r := <-resChan
		if r.err != nil {
			t.Fatalf("Concurrent refresh for device %d failed: %v", r.idx, r.err)
		}
	}
}
