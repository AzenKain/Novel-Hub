package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func SettingsRoutes(app fiber.Router, controller *controllers.SettingsController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	route := app.Group("/settings")
	app.Post("/setup/logo", controller.UploadSetupLogo)
	route.Get("/public", controller.PublicSettings)
	route.Get("/setup-status", controller.SetupStatus)
	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "setting.manage"),
		controller.AdminSettings,
	)
	route.Put(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "setting.manage"),
		controller.UpdateSettings,
	)
	route.Post(
		"/logo",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "setting.manage"),
		controller.UploadAdminLogo,
	)
	route.Post(
		"/smtp/test",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "setting.manage"),
		controller.TestSMTP,
	)
}
