package middlewares

import (
	"crypto/subtle"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
)

// sameOrigin reports whether the request's Origin (or Referer) header points at this host.
func sameOrigin(c fiber.Ctx) bool {
	source := c.Get("Origin")
	if source == "" {
		source = c.Get("Referer")
	}
	if source == "" {
		return true
	}
	u, err := url.Parse(source)
	if err != nil || u.Hostname() == "" {
		return false
	}
	host := c.Host()
	if h, _, found := strings.Cut(host, ":"); found {
		host = h
	}
	return strings.EqualFold(u.Hostname(), host)
}

func CSRFProtection() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		path := c.Path()
		if strings.HasPrefix(path, "/kobo/") || strings.HasPrefix(path, "/komga/") || strings.HasPrefix(path, "/api/opds/") || strings.HasPrefix(path, "/api/v1/sync/koreader/") || strings.HasPrefix(path, "/api/v1/vbook/") {
			return c.Next()
		}

		if strings.HasPrefix(path, "/api/v1/auth/") {
			if !sameOrigin(c) {
				return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
					Status:  false,
					Message: "Cross-origin auth request rejected",
				})
			}
			return c.Next()
		}

		if auth := c.Get("Authorization"); auth != "" {
			return c.Next()
		}

		if cookieToken := c.Cookies("access_token"); cookieToken == "" {
			return c.Next()
		}

		csrfCookie := c.Cookies("csrf_token")
		headerToken := c.Get("X-CSRF-Token")

		if csrfCookie == "" || headerToken == "" || subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(headerToken)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
				Status:  false,
				Message: "CSRF token mismatch or missing",
			})
		}

		return c.Next()
	}
}
