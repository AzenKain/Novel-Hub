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

func OPDSAuth(authService services.AuthService, settingsService services.SettingsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			settings, err := settingsService.Public(ctx)
			if err != nil || settings.GuestLoginRequired {
				c.Set("WWW-Authenticate", `Basic realm="NovelHub OPDS"`)
				return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
			}
			claims := GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}
		if !strings.HasPrefix(authHeader, "Basic ") {
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid authentication method")
		}
		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err != nil {
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials")
		}
		parts := strings.SplitN(string(payload), ":", 2)
		if len(parts) != 2 {
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{Email: parts[0], Password: parts[1]})
		if err != nil {
			c.Set("WWW-Authenticate", `Basic realm="NovelHub OPDS"`)
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials")
		}
		c.Locals("user_claims", claims)
		c.Locals("uid", claims.UId)
		return c.Next()
	}
}
