package middlewares

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

func KomgaAuth(authService services.AuthService, settingsService services.SettingsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		email, password, ok := komgaCredentials(c)
		if !ok {
			c.Set("WWW-Authenticate", `Basic realm="NovelHub Komga"`)
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{Email: email, Password: password})
		if err != nil {
			c.Set("WWW-Authenticate", `Basic realm="NovelHub Komga"`)
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
		}
		c.Locals("user_claims", claims)
		c.Locals("uid", claims.UId)
		return c.Next()
	}
}

// X-API-Key carries "<email>:<password>": there is no separate key store.
func komgaCredentials(c fiber.Ctx) (string, string, bool) {
	if apiKey := strings.TrimSpace(c.Get("X-API-Key")); apiKey != "" {
		if email, password, found := strings.Cut(apiKey, ":"); found {
			return email, password, true
		}
		return "", "", false
	}

	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", "", false
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return "", "", false
	}
	email, password, found := strings.Cut(string(payload), ":")
	if !found {
		return "", "", false
	}
	return email, password, true
}
