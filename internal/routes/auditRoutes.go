package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func AuditRoutes(
	router fiber.Router,
	controller *controllers.AuditController,
	userRepo repositories.UserRepository,
	permissionCache services.PermissionCache,
) {
	group := router.Group("/admin/audit")
	group.Use(middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, constants.PermSystemLogRead))

	group.Get("", controller.ListAuditLogs)
	group.Get("/actions", controller.ListAuditActions)
}
