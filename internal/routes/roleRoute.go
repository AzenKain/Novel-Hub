package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func RoleRoutes(app fiber.Router, controller *controllers.RoleController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	route := app.Group("/roles")
	route.Get("/", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.GetAllRole)
	route.Post("/", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.CreateRole)
	route.Get("/permissions", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.GetPermissions)
	route.Put("/reorder", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.ReorderRoles)
	route.Get("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.GetRoleByID)
	route.Put("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.UpdateRole)
	route.Put("/:id/permissions", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.UpdateRolePermissions)
	route.Delete("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "role.manage"), controller.DeleteRole)
}
