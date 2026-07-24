package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func JobRoutes(api fiber.Router, jobController *controllers.JobController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	group := api.Group("/jobs")
	group.Use(middlewares.JwtAccess(userRepo))
	group.Get("/", middlewares.RequirePermission(permissionCache, constants.PermJobRead), jobController.ListJobs)
	group.Get("/tasks", middlewares.RequireAnyPermission(permissionCache, constants.PermJobRead, constants.PermJobManage), jobController.ListTasks)
	group.Post("/", middlewares.RequirePermission(permissionCache, constants.PermJobManage), jobController.TriggerJob)
	group.Get("/schedules", middlewares.RequireAnyPermission(permissionCache, constants.PermJobRead, constants.PermJobManage), jobController.ListSchedules)
	group.Post("/schedules", middlewares.RequirePermission(permissionCache, constants.PermJobManage), jobController.CreateSchedule)
	group.Put("/schedules/:id", middlewares.RequirePermission(permissionCache, constants.PermJobManage), jobController.UpdateSchedule)
	group.Delete("/schedules/:id", middlewares.RequirePermission(permissionCache, constants.PermJobManage), jobController.DeleteSchedule)
	group.Post("/schedules/:id/run", middlewares.RequirePermission(permissionCache, constants.PermJobManage), jobController.RunSchedule)
	group.Get("/:id", middlewares.RequirePermission(permissionCache, constants.PermJobRead), jobController.GetJob)
}
