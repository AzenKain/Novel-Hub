package middlewares

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
)

func CSRFProtection() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		path := c.Path()
		if strings.HasPrefix(path, "/kobo/") || strings.HasPrefix(path, "/komga/") || strings.HasPrefix(path, "/api/opds/") || strings.HasPrefix(path, "/api/v1/sync/") || strings.HasPrefix(path, "/api/v1/vbook/") || strings.HasPrefix(path, "/api/v1/auth/") {
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
