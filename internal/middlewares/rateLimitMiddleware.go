package middlewares

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
)

// RateLimitScope selects how a limiter counts requests.
type RateLimitScope int

const (
	RateLimitAuth RateLimitScope = iota
	RateLimitOPDS
)

// RateLimit builds a limiter whose budget is read from SettingsService on every request, so an admin changing the numbers takes effect without a restart — the same trick RequestBodyLimit uses.
func RateLimit(settings services.SettingsService, scope RateLimitScope) fiber.Handler {
	return limiter.New(limiter.Config{
		MaxFunc: func(fiber.Ctx) int { return settings.Limits().RateLimitAuth },
		ExpirationFunc: func(fiber.Ctx) time.Duration {
			return time.Duration(settings.Limits().RateLimitAuthWindowSeconds) * time.Second
		},
		SkipSuccessfulRequests: scope == RateLimitOPDS,
		KeyGenerator: func(c fiber.Ctx) string {
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
