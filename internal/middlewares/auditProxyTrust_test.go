package middlewares_test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	_ "modernc.org/sqlite"

	"novelhub/internal/middlewares"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newAuditProxyTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "audit_proxy.db"))
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	return db, nil
}

func serveAuditApp(t *testing.T, app *fiber.App) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = app.Listener(ln) }()
	return fmt.Sprintf("http://%s", ln.Addr().String())
}

// TestProxyAuthBlocksSpoofedXFF is the regression test for the ProxyAuth IP-spoofing fix.
func TestProxyAuthBlocksSpoofedXFF(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh")

	db, err := newAuditProxyTestDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO roles (id, name, auto_assign) VALUES ('role-user', 'user', 1)`); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	userRepo := repositories.NewUserRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)

	buildApp := func(trusted []string) *fiber.App {
		mockSettings := &models.AdminSettings{
			ProxyAuth: models.ProxyAuthSettings{
				Enabled:        true,
				HeaderNames:    []string{"X-Forwarded-User"},
				TrustedProxies: trusted,
				AutoCreate:     true,
			},
		}
		settingsSvc := &MockSettingsService{settings: mockSettings}
		authSvc := &MockAuthService{}
		app := fiber.New(fiber.Config{
			TrustProxy:       true,
			TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true, Private: true, LinkLocal: true},
			ProxyHeader:      fiber.HeaderXForwardedFor,
		})
		app.Use(middlewares.ProxyAuth(settingsSvc, authSvc, userRepo, roleRepo, database.NewTxManager(db)))
		app.Get("/test", func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"auth": c.Get("Authorization")})
		})
		return app
	}

	doSpoofedRequest := func(baseURL string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/test", nil)
		req.Header.Set("X-Forwarded-For", "10.99.99.99")
		req.Header.Set("X-Forwarded-User", "admin@example.com")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}

	t.Run("spoofed XFF from untrusted raw peer is blocked", func(t *testing.T) {
		baseURL := serveAuditApp(t, buildApp([]string{"10.99.99.99"}))
		resp := doSpoofedRequest(baseURL)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d; expected request to pass through unauthenticated", resp.StatusCode)
		}
		u, err := userRepo.GetByEmail(context.Background(), "admin@example.com")
		if err == nil && u != nil {
			t.Fatal("user was auto-created from spoofed headers; raw-peer trust check is not in effect")
		}
		if resp.Header.Get("Set-Cookie") != "" {
			t.Fatal("session cookies were set from a spoofed, untrusted peer")
		}
	})

	t.Run("genuine trusted loopback peer still authenticates", func(t *testing.T) {
		baseURL := serveAuditApp(t, buildApp([]string{"127.0.0.1"}))
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/test", nil)
		req.Header.Set("X-Forwarded-User", "legit@example.com")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d; expected 200", resp.StatusCode)
		}
		u, err := userRepo.GetByEmail(context.Background(), "legit@example.com")
		if err != nil || u == nil {
			t.Fatal("user was not auto-created for a genuine trusted peer")
		}
		if resp.Header.Get("Set-Cookie") == "" {
			t.Fatal("no session cookies were set for a genuine trusted peer")
		}
	})
}
