package routes

import (
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

func KoboRoutes(app fiber.Router, koboController *controllers.KoboController, userRepo repositories.UserRepository) {
	group := app.Group("/kobo/v1", middlewares.JwtAccess(userRepo))

	group.Get("/initialization", koboController.GetInitialization)
	group.Get("/user/profile", koboController.GetUserProfile)
	group.Get("/library/sync", koboController.GetSyncList)
	group.Get("/books/:id/file", koboController.DownloadKePub)
	group.Post("/library/state", koboController.SyncState)
}
