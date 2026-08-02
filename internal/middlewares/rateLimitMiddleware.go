package middlewares

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
)

// RateLimitScope selects how a limiter counts requests.
//
// Both scopes share the same admin-tunable budget because they guard the same
// scarce resource: bcrypt.CompareHashAndPassword, which burns ~50-100ms of CPU
// per call. Nothing else in this app is worth rate limiting — a self-hosted
// library has a handful of accounts, and throttling the reader only throttles
// the person who owns the server.
type RateLimitScope int

const (
	// RateLimitAuth guards signin/register/setup. Every attempt counts, including
	// successful ones, so registration spam is bounded too.
	RateLimitAuth RateLimitScope = iota
	// RateLimitOPDS guards the OPDS catalog, which re-runs bcrypt on every single
	// request because Basic auth carries no session. Only failures count: a reader
	// app polling with valid credentials is normal traffic, not an attack, and
	// counting it would throttle every OPDS client in existence.
	RateLimitOPDS
)

// RateLimit builds a limiter whose budget is read from SettingsService on every
// request, so an admin changing the numbers takes effect without a restart —
// the same trick RequestBodyLimit uses.
func RateLimit(settings services.SettingsService, scope RateLimitScope) fiber.Handler {
	return limiter.New(limiter.Config{
		MaxFunc: func(fiber.Ctx) int { return settings.Limits().RateLimitAuth },
		ExpirationFunc: func(fiber.Ctx) time.Duration {
			return time.Duration(settings.Limits().RateLimitAuthWindowSeconds) * time.Second
		},
		SkipSuccessfulRequests: scope == RateLimitOPDS,
		KeyGenerator: func(c fiber.Ctx) string {
			// Reached before any token exists, so IP is the only available key.
			// Collapses to the proxy address unless TRUST_PROXY is set.
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
