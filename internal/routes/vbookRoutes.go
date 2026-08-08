package routes

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/request"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

func VBookRoutes(app fiber.Router, vbookController *controllers.VBookController, authService services.AuthService, settingsService services.SettingsService, userRepo repositories.UserRepository) {
	auth := func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		tokenQuery := strings.TrimSpace(c.Query("token"))

		if authHeader == "" && tokenQuery != "" {
			authHeader = "Bearer " + tokenQuery
		}

		if authHeader == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			settings, err := settingsService.Public(ctx)
			if err != nil || settings.GuestLoginRequired {
				return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
			}
			claims := middlewares.GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}

		if strings.HasPrefix(authHeader, "Basic ") {
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
				return apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials")
			}
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}

		// Fallback for Bearer token authorization
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		settings, err := settingsService.Public(ctx)
		if err == nil && !settings.GuestLoginRequired {
			claims := middlewares.GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}

		return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}

	v1 := app.Group("/api/v1/vbook", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), auth)
	v1.Get("/home", vbookController.GetHome)
	v1.Get("/genres", vbookController.GetGenres)
	v1.Get("/books", vbookController.GetBooks)
	v1.Get("/search", vbookController.SearchBooks)
	v1.Get("/detail", vbookController.GetDetail)
	v1.Get("/toc", vbookController.GetTOC)
	v1.Get("/chap", vbookController.GetChapterContent)
}
