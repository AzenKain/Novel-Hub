package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestCSRFProtectionMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFProtection())

	app.Get("/api/v1/sync/progress", func(c fiber.Ctx) error {
		return c.SendString("get ok")
	})
	app.Put("/api/v1/sync/progress", func(c fiber.Ctx) error {
		return c.SendString("progress updated")
	})
	app.Put("/api/v1/sync/koreader/syncs/progress", func(c fiber.Ctx) error {
		return c.SendString("kosync progress updated")
	})
	app.Post("/api/v1/auth/login", func(c fiber.Ctx) error {
		return c.SendString("login ok")
	})

	// 1.
	req1 := httptest.NewRequest("GET", "/api/v1/sync/progress", nil)
	resp1, err := app.Test(req1)
	if err != nil || (resp1.StatusCode != fiber.StatusOK && resp1.StatusCode != fiber.StatusNotFound) {
		t.Fatalf("GET request failed: %v", err)
	}

	// 2.
	req2 := httptest.NewRequest("PUT", "/api/v1/sync/progress", nil)
	req2.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-jwt"})
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if resp2.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 for cookie auth without CSRF token on /api/v1/sync/progress, got %d", resp2.StatusCode)
	}

	// 3.
	req3 := httptest.NewRequest("PUT", "/api/v1/sync/progress", nil)
	req3.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-jwt"})
	req3.AddCookie(&http.Cookie{Name: "csrf_token", Value: "secret-csrf-123"})
	req3.Header.Set("X-CSRF-Token", "secret-csrf-123")
	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if resp3.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for valid CSRF token, got %d", resp3.StatusCode)
	}

	// 4.
	req4 := httptest.NewRequest("PUT", "/api/v1/sync/koreader/syncs/progress", nil)
	req4.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-jwt"})
	resp4, err := app.Test(req4)
	if err != nil {
		t.Fatalf("Koreader request failed: %v", err)
	}
	if resp4.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for bypassed KOReader endpoint, got %d", resp4.StatusCode)
	}

	req5 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req5.Header.Set("Origin", "https://evil-attacker.com")
	req5.Host = "novelhub.local"
	resp5, err := app.Test(req5)
	if err != nil {
		t.Fatalf("Auth request failed: %v", err)
	}
	if resp5.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 for cross-origin auth request, got %d", resp5.StatusCode)
	}

	// 6. Reverse proxy forwarding X-Forwarded-Host
	req6 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req6.Header.Set("Origin", "https://calibre.kain.id.vn")
	req6.Header.Set("X-Forwarded-Host", "calibre.kain.id.vn")
	req6.Host = "127.0.0.1:3434"
	resp6, err := app.Test(req6)
	if err != nil {
		t.Fatalf("Auth request with X-Forwarded-Host failed: %v", err)
	}
	if resp6.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for matching X-Forwarded-Host, got %d", resp6.StatusCode)
	}

	// 7. Dev loopback origin (localhost:5173 to 127.0.0.1:3434)
	req7 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req7.Header.Set("Origin", "http://localhost:5173")
	req7.Host = "127.0.0.1:3434"
	resp7, err := app.Test(req7)
	if err != nil {
		t.Fatalf("Auth request with loopback dev origin failed: %v", err)
	}
	if resp7.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for dev loopback origin, got %d", resp7.StatusCode)
	}

	// 8. Cross-origin auth with verified double-submit CSRF cookie + header
	req8 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req8.Header.Set("Origin", "https://external-client.com")
	req8.Host = "novelhub.local"
	req8.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-token-abc"})
	req8.Header.Set("X-CSRF-Token", "csrf-token-abc")
	resp8, err := app.Test(req8)
	if err != nil {
		t.Fatalf("Auth request with double-submit CSRF token failed: %v", err)
	}
	if resp8.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for double-submit CSRF token, got %d", resp8.StatusCode)
	}

	// 9. Cross-origin auth with mismatched CSRF token
	req9 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req9.Header.Set("Origin", "https://external-client.com")
	req9.Host = "novelhub.local"
	req9.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-token-abc"})
	req9.Header.Set("X-CSRF-Token", "csrf-token-WRONG")
	resp9, err := app.Test(req9)
	if err != nil {
		t.Fatalf("Auth request with mismatched CSRF token failed: %v", err)
	}
	if resp9.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 for mismatched CSRF token, got %d", resp9.StatusCode)
	}
}
