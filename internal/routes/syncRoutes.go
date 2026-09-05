package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func SyncRoutes(app fiber.Router, syncController *controllers.SyncController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	apiV1 := app.Group("/v1/sync", middlewares.JwtAccess(userRepo))
	apiV1.Use(middlewares.RequirePermission(permissionCache, constants.PermTrackerSync))
	apiV1.Get("/progress/:bookId", syncController.GetProgress)
	apiV1.Put("/progress", syncController.UpdateProgress)

	koreader := app.Group("/v1/sync/koreader", middlewares.JwtAccess(userRepo))
	koreader.Use(middlewares.RequirePermission(permissionCache, constants.PermTrackerSync))
	koreader.Get("/syncs/progress/:document", syncController.KosyncGetProgress)
	koreader.Put("/syncs/progress", syncController.KosyncPushProgress)
}
