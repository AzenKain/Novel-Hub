package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func BookRoutes(app fiber.Router, bookController *controllers.BookController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	// For now, these are not protected, but we can wrap them in middleware later
	bookGroup := app.Group("/books")

	bookGroup.Get("/", middlewares.OptionalJwtAccess(userRepo), bookController.ListBooks)
	bookGroup.Get("/search/deep", middlewares.JwtAccess(userRepo), bookController.SearchDeep)
	bookGroup.Get("/files/duplicates", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookDuplicateManage), bookController.GetDuplicates)
	bookGroup.Delete("/files/:fileID", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookDelete, middlewares.BookFileLibraryAttr(bookRepo, "fileID")), bookController.DeleteBookFile)
	bookGroup.Get("/:id/download", middlewares.OptionalJwtAccess(userRepo), bookController.DownloadBook)
	bookGroup.Get("/:id/files", middlewares.OptionalJwtAccess(userRepo), bookController.ListBookFiles)
	bookGroup.Post("/:id/files", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookUpload, middlewares.BookLibraryAttr(bookRepo, "id")), bookController.UploadBookFiles)
	bookGroup.Get("/:id", middlewares.OptionalJwtAccess(userRepo), bookController.GetBook)
	bookGroup.Get("/:id/chapters", middlewares.OptionalJwtAccess(userRepo), bookController.ListChapters)
	bookGroup.Get("/:id/series", middlewares.OptionalJwtAccess(userRepo), bookController.GetBookSeries)
	bookGroup.Get("/:id/search", middlewares.OptionalJwtAccess(userRepo), bookController.SearchInBook)
	bookGroup.Put("/:id/metadata", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "id")), bookController.UpdateMetadata)
	bookGroup.Post("/:id/enrich", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "id")), bookController.EnrichBook)
	bookGroup.Post("/batch-enrich", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookBulkManage), bookController.BatchEnrichBooks)
	bookGroup.Patch("/:id/archive", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookArchive, middlewares.BookLibraryAttr(bookRepo, "id")), bookController.ArchiveBook)
	bookGroup.Post("/:id/send-email", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookSendEmail, middlewares.BookLibraryAttr(bookRepo, "id")), bookController.SendBookToEmail)
	bookGroup.Post("/bulk-delete", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookBulkManage), bookController.BulkDeleteBooks)
	bookGroup.Post("/bulk-move", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookBulkManage), bookController.BulkMoveBooks)
	bookGroup.Post("/bulk-assign-collections", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookBulkManage), bookController.BulkAssignCollections)
	bookGroup.Post("/bulk-add-tags", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookBulkManage), bookController.BulkAddTags)
	bookGroup.Delete("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookDelete, middlewares.BookLibraryAttr(bookRepo, "id")), bookController.DeleteBook)
}
