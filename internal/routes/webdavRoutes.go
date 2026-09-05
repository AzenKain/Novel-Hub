package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func WebDAVRoutes(
	app fiber.Router,
	webdavController *controllers.WebDAVController,
	authService services.AuthService,
	settingsService services.SettingsService,
	permissionCache services.PermissionCache,
	userRepo ...repositories.UserRepository,
) {
	auth := middlewares.WebDAVAuth(authService, settingsService, permissionCache, userRepo...)
	perm := middlewares.RequirePermission(permissionCache, constants.PermWebDAVRead)

	registerGroup := func(g fiber.Router) {
		// OPTIONS capability discovery MUST be unauthenticated per RFC 4918 §10.1
		g.Add([]string{fiber.MethodOptions}, "", webdavController.HandleOptions)
		g.Add([]string{fiber.MethodOptions}, "/", webdavController.HandleOptions)
		g.Add([]string{fiber.MethodOptions}, "/*", webdavController.HandleOptions)

		// Protected WebDAV methods require authentication and read permission
		g.Add([]string{"PROPFIND"}, "", auth, perm, webdavController.HandlePropfind)
		g.Add([]string{"PROPFIND"}, "/", auth, perm, webdavController.HandlePropfind)
		g.Add([]string{"PROPFIND"}, "/*", auth, perm, webdavController.HandlePropfind)

		g.Get("", auth, perm, webdavController.HandleGet)
		g.Get("/", auth, perm, webdavController.HandleGet)
		g.Get("/*", auth, perm, webdavController.HandleGet)

		g.Head("", auth, perm, webdavController.HandleHead)
		g.Head("/", auth, perm, webdavController.HandleHead)
		g.Head("/*", auth, perm, webdavController.HandleHead)
	}

	registerGroup(app.Group("/webdav"))
	registerGroup(app.Group("/api/webdav"))
}
