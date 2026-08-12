package middlewares

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

func VBookAuth(authService services.AuthService, settingsService services.SettingsService, userRepo repositories.UserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		tokenQuery := strings.TrimSpace(c.Query("token"))
		hasToken := authHeader != "" || tokenQuery != "" || c.Cookies("access_token") != ""

		if strings.HasPrefix(authHeader, "Basic ") {
			payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
			if err != nil {
				return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
			}
			parts := strings.SplitN(string(payload), ":", 2)
			if len(parts) != 2 {
				return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{Email: parts[0], Password: parts[1]})
			if err != nil {
				return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
			}
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}

		if hasToken {
			if authHeader == "" && tokenQuery != "" {
				c.Request().Header.Set("Authorization", "Bearer "+tokenQuery)
			}
			return OptionalJwtAccess(userRepo)(c)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		settings, err := settingsService.Public(ctx)
		if err != nil || settings.GuestLoginRequired {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		}
		claims := GuestClaims()
		c.Locals("user_claims", claims)
		c.Locals("uid", claims.UId)
		return c.Next()
	}
}
