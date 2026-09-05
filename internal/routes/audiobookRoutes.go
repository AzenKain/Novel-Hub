package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func AudiobookRoutes(app fiber.Router, audiobookController *controllers.AudiobookController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	group := app.Group("/books/:book_id/audiobook/chapters")

	group.Get("/", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookRead, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.ListChapters)
	group.Post("/", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.UpsertChapter)
	group.Put("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.UpsertChapter)
	group.Delete("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.DeleteChapter)
	group.Delete("/", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.DeleteAllChapters)
	group.Post("/lookup", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.LookupChapters)

	mergeGroup := app.Group("/books/:book_id/merge-audio", middlewares.JwtAccess(userRepo))
	mergeGroup.Post("", middlewares.RequirePermission(permissionCache, constants.PermBookUpload, middlewares.BookLibraryAttr(bookRepo, "book_id")), audiobookController.MergeAudio)
}
