package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func TrackerRoutes(app fiber.Router, trackerController *controllers.TrackerController, userRepo repositories.UserRepository, permissionCache services.PermissionCache, settingsService services.SettingsService) {
	group := app.Group("/trackers", middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, constants.PermTrackerSync))
	group.Use(middlewares.RequireAniListTrackingEnabled(settingsService))

	group.Post("/connect", trackerController.ConnectTracker)
	group.Post("/map", trackerController.MapBookTracker)
	group.Get("/search", trackerController.SearchAniList)
	group.Post("/sync", trackerController.SyncProgress)
}
