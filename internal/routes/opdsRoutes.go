package routes

import (
	"encoding/base64"
	"strings"

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
			settings, err := settingsService.Public(c.Context())
			if err != nil || settings.GuestAccess.Mode == "login_required" {
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
		claims, err := authService.ValidateCredentials(c.Context(), &request.SignInDto{Email: parts[0], Password: parts[1]})
		if err != nil {
			c.Set("WWW-Authenticate", `Basic realm="NovelHub OPDS"`)
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials")
		}
		c.Locals("user_claims", claims)
		c.Locals("uid", claims.UId)
		return c.Next()
	}

	v1 := app.Group("/opds/v1", auth)
	v1.Get("/", opdsController.GetRootCatalog)
	v1.Get("/recent", opdsController.GetRecentBooks)
	v2 := app.Group("/opds/v2", auth)
	v2.Get("/catalog", opdsController.GetOPDS2Catalog)
}
