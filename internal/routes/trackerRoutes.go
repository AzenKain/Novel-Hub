package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func TrackerRoutes(app fiber.Router, trackerController *controllers.TrackerController, userRepo repositories.UserRepository) {
	group := app.Group("/trackers")

	group.Post("/connect", middlewares.JwtAccess(userRepo), trackerController.ConnectTracker)
	group.Post("/map", middlewares.JwtAccess(userRepo), trackerController.MapBookTracker)
	group.Get("/search", middlewares.JwtAccess(userRepo), trackerController.SearchAniList)
	group.Post("/sync", middlewares.JwtAccess(userRepo), trackerController.SyncProgress)
}
