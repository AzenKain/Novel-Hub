package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func BookRoutes(app fiber.Router, bookController *controllers.BookController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	// For now, these are not protected, but we can wrap them in middleware later
	bookGroup := app.Group("/books")

	bookGroup.Get("/", middlewares.OptionalJwtAccess(userRepo), bookController.ListBooks)
	bookGroup.Get("/search/deep", middlewares.JwtAccess(userRepo), bookController.SearchDeep)
	bookGroup.Get("/files/duplicates", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "book.manage"), bookController.GetDuplicates)
	bookGroup.Get("/:id/download", middlewares.OptionalJwtAccess(userRepo), bookController.DownloadBook)
	bookGroup.Get("/:id/files", middlewares.OptionalJwtAccess(userRepo), bookController.ListBookFiles)
	bookGroup.Post("/:id/files", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "book.manage", middlewares.BookLibraryAttr(bookRepo, "id")), bookController.UploadBookFiles)
	bookGroup.Get("/:id", middlewares.OptionalJwtAccess(userRepo), bookController.GetBook)
	bookGroup.Get("/:id/chapters", middlewares.OptionalJwtAccess(userRepo), bookController.ListChapters)
	bookGroup.Put("/:id/metadata", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "book.manage", middlewares.BookLibraryAttr(bookRepo, "id")), bookController.UpdateMetadata)
	bookGroup.Patch("/:id/archive", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "book.manage", middlewares.BookLibraryAttr(bookRepo, "id")), bookController.ArchiveBook)
	bookGroup.Delete("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "book.manage", middlewares.BookLibraryAttr(bookRepo, "id")), bookController.DeleteBook)
}
