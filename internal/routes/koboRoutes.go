package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func KoboRoutes(app fiber.Router, koboController *controllers.KoboController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	group := app.Group("/kobo/v1", middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, constants.PermKoboSync))

	group.Get("/initialization", koboController.GetInitialization)
	group.Get("/user/profile", koboController.GetUserProfile)
	group.Get("/library/sync", koboController.GetSyncList)
	group.Get("/books/:id/file", koboController.DownloadKePub)
	group.Post("/library/state", koboController.SyncState)
}
