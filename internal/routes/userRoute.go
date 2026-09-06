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
	route.Post("/current/avatar", middlewares.JwtAccess(userRepo), controller.UploadAvatar)
	route.Patch("/current/password", middlewares.JwtAccess(userRepo), controller.ChangePassword)
	route.Post("/current/revoke-sessions", middlewares.JwtAccess(userRepo), controller.RevokeUserSessions)

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
	route.Post(
		"/bulk/delete",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.BulkDeleteUsers,
	)
	route.Post(
		"/bulk/restore",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.BulkRestoreUsers,
	)
	route.Post(
		"/bulk/roles",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.BulkChangeUserRoles,
	)
	route.Patch(
		"/bulk/info",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.BulkUpdateUserInfo,
	)
	route.Post(
		"/bulk/email",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.BulkSendEmail,
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
	route.Post(
		"/:id/avatar",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.AdminUploadAvatar,
	)
	route.Patch(
		"/:id/password",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.AdminResetPassword,
	)
	route.Post(
		"/:id/revoke-sessions",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, "user.manage"),
		controller.RevokeUserSessions,
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
