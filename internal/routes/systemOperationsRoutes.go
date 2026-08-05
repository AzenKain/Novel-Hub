package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func SystemOperationsRoutes(api fiber.Router, controller *controllers.SystemOperationsController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	logs := api.Group("/system/logs", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermSystemLogRead))
	logs.Get("/", controller.ListLogs)
	logs.Get("/tail", controller.TailLogs)
	logs.Get("/:name/download", controller.DownloadLog)

	backups := api.Group("/system/backups", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermSystemBackup))
	backups.Get("/", controller.ListBackups)
	backups.Post("/", controller.CreateBackup)
	backups.Get("/:name/download", controller.DownloadBackup)
	backups.Delete("/:name", controller.DeleteBackup)
	backups.Post("/:name/restore", controller.RestoreBackup)

	metrics := api.Group("/system/metrics", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermSystemLogRead))
	metrics.Get("/cache", controller.GetCacheStats)
}
