package middlewares

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/config"
)

// RateLimitScope selects which pair of runtime limits a limiter reads.
type RateLimitScope int

const (
	// RateLimitAPI covers the whole /api surface with a generous budget.
	RateLimitAPI RateLimitScope = iota
	// RateLimitAuth guards signin/register/setup with a much tighter one.
	RateLimitAuth
)

// RateLimit builds a limiter whose budget is read from SettingsService on every
// request, so an admin changing the numbers takes effect without a restart —
// the same trick RequestBodyLimit uses.
func RateLimit(settings services.SettingsService, scope RateLimitScope) fiber.Handler {
	maxFor := func() int {
		if scope == RateLimitAuth {
			return settings.Limits().RateLimitAuth
		}
		return settings.Limits().RateLimitAPI
	}
	windowFor := func() time.Duration {
		limits := settings.Limits()
		seconds := limits.RateLimitAPIWindowSeconds
		if scope == RateLimitAuth {
			seconds = limits.RateLimitAuthWindowSeconds
		}
		return time.Duration(seconds) * time.Second
	}

	// This middleware runs before JwtAccess, so c.Locals("uid") is not populated
	// yet. We verify the signature ourselves rather than trusting the raw token:
	// bucketing on an unverified string would let anyone mint a fresh bucket per
	// request by sending random garbage, which defeats the limiter entirely.
	accessSecret := []byte(config.GetConfigWithDefault("JWT_SECRET", ""))

	return limiter.New(limiter.Config{
		MaxFunc:        func(fiber.Ctx) int { return maxFor() },
		ExpirationFunc: func(fiber.Ctx) time.Duration { return windowFor() },
		KeyGenerator: func(c fiber.Ctx) string {
			if uid := verifiedUserID(c, accessSecret); uid != "" {
				return "u:" + uid
			}
			// Behind a proxy this collapses to the proxy IP unless TRUST_PROXY is
			// configured — see NewHTTPServer.
			return "ip:" + c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(response.CommonResponse{
				Status:  false,
				Message: "Too many requests, please slow down",
			})
		},
	})
}

// verifiedUserID returns the subject of a valid access token, or "" for anyone
// unauthenticated (who then gets bucketed by IP instead).
func verifiedUserID(c fiber.Ctx, secret []byte) string {
	if len(secret) == 0 {
		return ""
	}
	raw := c.Cookies("access_token")
	if raw == "" {
		if header := c.Get(fiber.HeaderAuthorization); len(header) > 7 && strings.EqualFold(header[:7], "bearer ") {
			raw = header[7:]
		}
	}
	if raw == "" {
		return ""
	}

	claims := &response.JWTClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid || claims.UId == "" {
		return ""
	}
	return claims.UId
}
