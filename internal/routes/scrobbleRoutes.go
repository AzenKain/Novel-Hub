package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func ScrobbleRoutes(app fiber.Router, scrobbleController *controllers.ScrobbleController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	group := app.Group("/scrobble/hardcover", middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, constants.PermTrackerSync))

	group.Post("/connect", scrobbleController.ConnectHardcover)
	group.Post("/sync", scrobbleController.SyncHardcoverProgress)

	// The OAuth callback is a browser redirect with no JWT — the state param is the auth.
	app.Get("/scrobble/hardcover/callback", scrobbleController.HardcoverCallback)
}