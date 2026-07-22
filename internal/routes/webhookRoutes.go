package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"

	"github.com/gofiber/fiber/v3"
)

func WebhookRoutes(
	router fiber.Router,
	controller *controllers.WebhookController,
	userRepo repositories.UserRepository,
	permissionCache services.PermissionCache,
) {
	group := router.Group("/admin/webhooks")
	group.Use(middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, "settings.edit"))

	group.Get("", controller.ListWebhooks)
	group.Post("", controller.CreateWebhook)
	group.Get("/:id", controller.GetWebhook)
	group.Put("/:id", controller.UpdateWebhook)
	group.Delete("/:id", controller.DeleteWebhook)
	group.Post("/:id/test", controller.TestPing)
}
