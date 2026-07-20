package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func JobRoutes(api fiber.Router, jobController *controllers.JobController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	group := api.Group("/jobs")
	group.Use(middlewares.JwtAccess(userRepo))
	group.Use(middlewares.RequirePermission(permissionCache, "job.read"))
	group.Get("/:id", jobController.GetJob)
}
