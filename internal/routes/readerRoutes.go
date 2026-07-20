package routes

import (
	"github.com/gofiber/fiber/v3"
	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func SetupReaderRoutes(router fiber.Router, controller *controllers.ReaderController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	reader := router.Group("/reader")

	reader.Get("/:id/bootstrap", middlewares.OptionalJwtAccess(userRepo), controller.GetBootstrap)
	reader.Get("/:id/chapter/:chapterId", middlewares.OptionalJwtAccess(userRepo), controller.GetChapter)
	reader.Get("/:id/file", middlewares.OptionalJwtAccess(userRepo), controller.GetFile)
	reader.Get("/:id/images", middlewares.OptionalJwtAccess(userRepo), controller.ListImages)
	reader.Post("/:id/cover", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "book.manage", middlewares.BookLibraryAttr(bookRepo, "id")), controller.UpdateCover)
	reader.Get("/:id/asset/*", middlewares.OptionalJwtAccess(userRepo), controller.GetAsset)
}
