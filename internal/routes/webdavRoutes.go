package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func WebDAVRoutes(
	app fiber.Router,
	webdavController *controllers.WebDAVController,
	authService services.AuthService,
	settingsService services.SettingsService,
	permissionCache services.PermissionCache,
) {
	auth := middlewares.WebDAVAuth(authService, settingsService)

	webdavGroup := app.Group("/webdav")
	webdavGroup.Use(auth)
	webdavGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermWebDAVRead))

	webdavGroup.Add([]string{"OPTIONS"}, "", webdavController.HandleOptions)
	webdavGroup.Add([]string{"OPTIONS"}, "/*", webdavController.HandleOptions)

	webdavGroup.Add([]string{"PROPFIND"}, "", webdavController.HandlePropfind)
	webdavGroup.Add([]string{"PROPFIND"}, "/*", webdavController.HandlePropfind)

	webdavGroup.Get("", webdavController.HandleGet)
	webdavGroup.Get("/*", webdavController.HandleGet)

	webdavGroup.Head("", webdavController.HandleHead)
	webdavGroup.Head("/*", webdavController.HandleHead)
}
