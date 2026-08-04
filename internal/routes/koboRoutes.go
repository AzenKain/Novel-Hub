package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func KoboRoutes(
	app fiber.Router,
	koboController *controllers.KoboController,
	koboRepo repositories.KoboRepository,
	userRepo repositories.UserRepository,
	permissionCache services.PermissionCache,
	settings services.SettingsService,
) {
	group := app.Group("/kobo/:kobo_token",
		middlewares.RequestBodyLimit(settings),
		middlewares.KoboAuth(koboRepo, userRepo),
	)
	group.Use(middlewares.RequirePermission(permissionCache, constants.PermKoboSync))

	group.Get("/v1/initialization", koboController.GetInitialization)

	group.Post("/v1/auth/device", koboController.AuthDevice)
	group.Post("/v1/auth/refresh", koboController.AuthDevice)

	group.Get("/v1/user/profile", koboController.GetUserProfile)
	group.Get("/v1/library/sync", koboController.GetSyncList)
	group.Get("/v1/library/:uuid/metadata", koboController.GetBookMetadata)
	group.Get("/v1/library/:uuid/state", koboController.GetReadingState)
	group.Put("/v1/library/:uuid/state", koboController.PutReadingState)

	group.Get("/:uuid/:width/:height/:isGreyscale/image.jpg", koboController.GetCoverImage)
	group.Get("/:uuid/:width/:height/:quality/:isGreyscale/image.jpg", koboController.GetCoverImage)

	group.Get("/download/:id/:format", koboController.DownloadKePub)
}

func KoboSetupRoutes(app fiber.Router, koboController *controllers.KoboController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	route := app.Group("/kobo",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, constants.PermKoboSync),
	)

	route.Get("/setup", koboController.GetSetup)
	route.Post("/setup/regenerate", koboController.RegenerateSetup)
	route.Delete("/setup", koboController.RevokeSetup)
}
