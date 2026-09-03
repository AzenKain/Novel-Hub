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

	// 1. Safe methods (GET) pass through without CSRF
	req1 := httptest.NewRequest("GET", "/api/v1/sync/progress", nil)
	resp1, err := app.Test(req1)
	if err != nil || (resp1.StatusCode != fiber.StatusOK && resp1.StatusCode != fiber.StatusNotFound) {
		t.Fatalf("GET request failed: %v", err)
	}

	// 2. Cookie auth on /api/v1/sync/progress without CSRF token is rejected with 403
	req2 := httptest.NewRequest("PUT", "/api/v1/sync/progress", nil)
	req2.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-jwt"})
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if resp2.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 for cookie auth without CSRF token on /api/v1/sync/progress, got %d", resp2.StatusCode)
	}

	// 3. Cookie auth on /api/v1/sync/progress with matching CSRF cookie and header succeeds
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

	// 4. KOReader sync endpoint is bypassed from CSRF requirement
	req4 := httptest.NewRequest("PUT", "/api/v1/sync/koreader/syncs/progress", nil)
	req4.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-jwt"})
	resp4, err := app.Test(req4)
	if err != nil {
		t.Fatalf("Koreader request failed: %v", err)
	}
	if resp4.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for bypassed KOReader endpoint, got %d", resp4.StatusCode)
	}

	// 5. Auth endpoint with cross-origin Origin is blocked
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
}
