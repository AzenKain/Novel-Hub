package middlewares

import (
	"crypto/subtle"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
)

func isLoopbackHost(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || h == "[::1]"
}

func hostOnly(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if first, _, found := strings.Cut(raw, ","); found {
		raw = strings.TrimSpace(first)
	}
	if h, _, found := strings.Cut(raw, ":"); found {
		return strings.TrimSpace(h)
	}
	return raw
}

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
	originHost := u.Hostname()

	candidateHosts := []string{
		hostOnly(c.Host()),
		hostOnly(c.Get("X-Forwarded-Host")),
	}
	if fwd := c.Get("Forwarded"); fwd != "" {
		for _, part := range strings.Split(fwd, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(strings.ToLower(part), "host=") {
				candidateHosts = append(candidateHosts, hostOnly(part[5:]))
			}
		}
	}

	for _, ch := range candidateHosts {
		if ch == "" {
			continue
		}
		if strings.EqualFold(originHost, ch) {
			return true
		}
		if isLoopbackHost(originHost) && isLoopbackHost(ch) {
			return true
		}
	}

	return false
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
				csrfCookie := c.Cookies("csrf_token")
				headerToken := c.Get("X-CSRF-Token")
				if csrfCookie == "" || headerToken == "" || subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(headerToken)) != 1 {
					return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
						Status:  false,
						Message: "Cross-origin auth request rejected",
					})
				}
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
