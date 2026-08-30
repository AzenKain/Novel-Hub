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

func WebDAVAuth(authService services.AuthService, settingsService services.SettingsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		// 1. Basic Auth
		if strings.HasPrefix(authHeader, "Basic ") {
			payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
			if err == nil {
				email, password, found := strings.Cut(string(payload), ":")
				if found {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{
						Email:    email,
						Password: password,
					})
					if err == nil && claims != nil {
						c.Locals("user_claims", claims)
						c.Locals("uid", claims.UId)
						return c.Next()
					}
				}
			}

			c.Set("WWW-Authenticate", `Basic realm="NovelHub WebDAV"`)
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
		}

		// 2. Bearer Token Auth
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{
				Email:    token,
				Password: token,
			})
			if err == nil && claims != nil {
				c.Locals("user_claims", claims)
				c.Locals("uid", claims.UId)
				return c.Next()
			}
		}

		// 3. Query Token Auth (?token=...)
		if token := c.Query("token"); token != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{
				Email:    token,
				Password: token,
			})
			if err == nil && claims != nil {
				c.Locals("user_claims", claims)
				c.Locals("uid", claims.UId)
				return c.Next()
			}
		}

		// 4. Guest / Anonymous Fallback
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		settings, err := settingsService.Public(ctx)
		if err == nil && !settings.GuestLoginRequired {
			claims := GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}

		c.Set("WWW-Authenticate", `Basic realm="NovelHub WebDAV"`)
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}
}
