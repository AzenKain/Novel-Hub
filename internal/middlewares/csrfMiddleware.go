package middlewares

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
)

// CSRFProtection returns a middleware that validates the X-CSRF-Token header
// against the csrf_token cookie for state-mutating requests (POST, PUT, PATCH, DELETE)
// when authenticated via session cookies.
func CSRFProtection() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		path := c.Path()
		if strings.HasPrefix(path, "/kobo/") || strings.HasPrefix(path, "/komga/") || strings.HasPrefix(path, "/api/opds/") || strings.HasPrefix(path, "/api/v1/sync/") {
			return c.Next()
		}

		// Authorization header (Bearer/Basic) bypasses cookie-based CSRF checks
		if auth := c.Get("Authorization"); auth != "" {
			return c.Next()
		}

		// If no access_token cookie is present, skip CSRF check
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
