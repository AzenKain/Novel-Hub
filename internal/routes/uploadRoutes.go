package routes

import (
	"github.com/gofiber/fiber/v3"
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
)

func SetupUploadRoutes(router fiber.Router, controller *controllers.UploadController, userRepo repositories.UserRepository) {
	uploadGroup := router.Group("/upload", middlewares.JwtAccess(userRepo))
	
	uploadGroup.Post("/init", controller.InitUpload)
	uploadGroup.Post("/:uploadId/chunk", controller.UploadChunk)
	uploadGroup.Post("/:uploadId/commit", controller.CommitUpload)
}
