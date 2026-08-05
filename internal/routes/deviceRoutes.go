package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func DeviceRoutes(
	api fiber.Router,
	deviceController *controllers.DeviceController,
	userRepo repositories.UserRepository,
	bookRepo repositories.BookDBRepository,
	permissionCache services.PermissionCache,
) {
	devices := api.Group("/user/devices", middlewares.JwtAccess(userRepo))
	devices.Get("/", deviceController.ListDevices)
	devices.Post("/", deviceController.CreateDevice)
	devices.Delete("/:id", deviceController.DeleteDevice)

	books := api.Group("/books")
	books.Post(
		"/:id/push",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, constants.PermBookSendEmail, middlewares.BookLibraryAttr(bookRepo, "id")),
		deviceController.PushBook,
	)
}
