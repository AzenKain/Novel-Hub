package routes

import (
	"github.com/gofiber/fiber/v3"
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func SetupUploadRoutes(router fiber.Router, controller *controllers.UploadController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	uploadGroup := router.Group("/upload", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookUpload))

	uploadGroup.Post("/init", controller.InitUpload)
	uploadGroup.Post("/:uploadId/chunk", controller.UploadChunk)
	uploadGroup.Post("/:uploadId/commit", controller.CommitUpload)
}
