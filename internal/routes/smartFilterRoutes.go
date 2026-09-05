package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func SmartFilterRoutes(app fiber.Router, controller *controllers.SmartFilterController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	group := app.Group("/smart-filters", middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, constants.PermBookCollection))

	group.Get("/", controller.ListSmartFilters)
	group.Post("/", controller.CreateSmartFilter)
	group.Put("/reorder-home", controller.ReorderHome)
	group.Get("/:id", controller.GetSmartFilter)
	group.Put("/:id", controller.UpdateSmartFilter)
	group.Delete("/:id", controller.DeleteSmartFilter)
	group.Put("/:id/pin-sidebar", controller.PinSidebar)
	group.Put("/:id/pin-home", controller.PinHome)

	app.Get("/smart-filters/:id/books", middlewares.OptionalJwtAccess(userRepo), controller.GetSmartFilterBooks)
}
