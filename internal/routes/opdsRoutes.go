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

func OPDSRoutes(app fiber.Router, opdsController *controllers.OPDSController, authService services.AuthService, settingsService services.SettingsService, _ repositories.UserRepository) {
	auth := func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			settings, err := settingsService.Public(ctx)
			if err != nil || settings.GuestLoginRequired {
				c.Set("WWW-Authenticate", `Basic realm="NovelHub OPDS"`)
				return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
			}
			claims := middlewares.GuestClaims()
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

	v1 := app.Group("/opds/v1", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), auth)
	v1.Get("/", opdsController.GetRootCatalog)
	v1.Get("/opensearch.xml", opdsController.GetOpenSearchDescription)
	v1.Get("/search", opdsController.SearchCatalog)
	v1.Get("/recent", opdsController.GetRecentBooks)
	v1.Get("/authors", opdsController.GetAuthorsCatalog)
	v1.Get("/authors/:name", opdsController.GetAuthorBooks)
	v1.Get("/series", opdsController.GetSeriesCatalog)
	v1.Get("/series/:name", opdsController.GetSeriesBooks)
	v1.Get("/tags", opdsController.GetTagsCatalog)
	v1.Get("/tags/:name", opdsController.GetTagBooks)

	v2 := app.Group("/opds/v2", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), auth)
	v2.Get("/catalog", opdsController.GetOPDS2Catalog)
}
