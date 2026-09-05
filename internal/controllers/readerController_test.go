package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/etag"
)

// GetAsset serves comic pages — one request per page, so a 200-page volume hits it 200 times and re-opens the archive each time.
func TestAssetResponseIsRevalidatable(t *testing.T) {
	app := fiber.New()
	app.Get("/asset", etag.New(), func(c fiber.Ctx) error {
		c.Set("Content-Type", "image/jpeg")
		c.Set(fiber.HeaderCacheControl, "private, max-age=3600")
		return c.Send([]byte("page-bytes"))
	})

	first, err := app.Test(httptest.NewRequest(http.MethodGet, "/asset", nil))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", first.StatusCode)
	}
	tag := first.Header.Get(fiber.HeaderETag)
	if tag == "" {
		t.Fatal("no ETag on the first response, so a re-read can never be revalidated")
	}
	if got := first.Header.Get(fiber.HeaderCacheControl); got != "private, max-age=3600" {
		t.Fatalf("Cache-Control = %q, want the handler's own value", got)
	}

	revalidate := httptest.NewRequest(http.MethodGet, "/asset", nil)
	revalidate.Header.Set(fiber.HeaderIfNoneMatch, tag)
	second, err := app.Test(revalidate)
	if err != nil {
		t.Fatalf("revalidation request failed: %v", err)
	}
	if second.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidation with a matching ETag: expected 304, got %d", second.StatusCode)
	}
}

// The auth cookie's Secure flag is derived from the request instead of an env var.
func TestAuthCookieSecureFollowsRequestScheme(t *testing.T) {
	newApp := func(trustProxy bool) *fiber.App {
		cfg := fiber.Config{TrustProxy: trustProxy}
		if trustProxy {
			cfg.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0"}}
		}
		app := fiber.New(cfg)
		app.Get("/login", func(c fiber.Ctx) error {
			setAuthCookie(c, "access_token", "token-value", time.Hour)
			return c.SendStatus(fiber.StatusOK)
		})
		return app
	}

	cookieFor := func(t *testing.T, app *fiber.App, forwardedProto string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		if forwardedProto != "" {
			req.Header.Set("X-Forwarded-Proto", forwardedProto)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("login request failed: %v", err)
		}
		header := resp.Header.Get("Set-Cookie")
		if header == "" {
			t.Fatal("no Set-Cookie on the login response")
		}
		return strings.ToLower(header)
	}

	if got := cookieFor(t, newApp(false), ""); strings.Contains(got, "secure") {
		t.Errorf("plain HTTP cookie must not be Secure: %s", got)
	}

	if got := cookieFor(t, newApp(true), "https"); !strings.Contains(got, "secure") {
		t.Errorf("cookie behind an HTTPS proxy must be Secure: %s", got)
	}

	if got := cookieFor(t, newApp(false), "https"); strings.Contains(got, "secure") {
		t.Errorf("X-Forwarded-Proto from an untrusted source must not set Secure: %s", got)
	}

	if got := cookieFor(t, newApp(false), ""); strings.Contains(got, "domain=") {
		t.Errorf("cookie must stay host-scoped, got a Domain: %s", got)
	}
}
