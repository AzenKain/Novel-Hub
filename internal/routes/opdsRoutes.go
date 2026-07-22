package routes

import (
	"encoding/base64"
	"strings"

	"novelhub/internal/controllers"
	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"

	"github.com/gofiber/fiber/v3"
)

func OPDSRoutes(app fiber.Router, opdsController *controllers.OPDSController, authService services.AuthService, settingsService services.SettingsService, userRepo repositories.UserRepository) {
	group := app.Group("/opds/v1")

	group.Use(func(c fiber.Ctx) error {
		guestAllows := settingsService.GuestAllows("")
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			if guestAllows {
				return c.Next()
			}
			c.Set("WWW-Authenticate", `Basic realm="NovelHub OPDS"`)
			return apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
		}

		if !strings.HasPrefix(authHeader, "Basic ") {
			if guestAllows {
				return c.Next()
			}
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid authentication method")
		}

		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err != nil {
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid base64 encoding")
		}
		parts := strings.SplitN(string(payload), ":", 2)
		if len(parts) != 2 {
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid basic auth payload")
		}

		email := parts[0]
		password := parts[1]

		_, err = authService.Signin(c.Context(), &request.SignInDto{
			Email:    email,
			Password: password,
		})
		if err != nil {
			if guestAllows {
				return c.Next()
			}
			c.Set("WWW-Authenticate", `Basic realm="NovelHub OPDS"`)
			return apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials")
		}
		return c.Next()
	})

	group.Get("/", opdsController.GetRootCatalog)
	group.Get("/recent", opdsController.GetRecentBooks)

	v2Group := app.Group("/opds/v2")
	v2Group.Get("/catalog", opdsController.GetOPDS2Catalog)
}
