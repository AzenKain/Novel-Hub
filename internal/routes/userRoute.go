package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func UserRoutes(app fiber.Router, controller *controllers.UserController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	route := app.Group("/users")

	route.Get("/current", middlewares.JwtAccess(userRepo), controller.GetUserCurrent)
	route.Put("/current", middlewares.JwtAccess(userRepo), controller.UpdateProfile)
	route.Patch("/current/password", middlewares.JwtAccess(userRepo), controller.ChangePassword)

	route.Post(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.CreateUser,
	)
	route.Get(
		"/",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.SearchUser,
	)
	route.Get(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.GetUserByID,
	)
	route.Put(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.AdminUpdateProfile,
	)
	route.Patch(
		"/:id/password",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.AdminResetPassword,
	)
	route.Patch(
		"/:id/role",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.ChangeRoleUser,
	)
	route.Post(
		"/:id/email",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.SendUserEmail,
	)
	route.Patch(
		"/:id/restore",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.RestoreUser,
	)
	route.Delete(
		"/:id",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.DeleteUser,
	)
}
