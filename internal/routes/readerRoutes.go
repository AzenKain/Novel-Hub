package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/etag"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func SetupReaderRoutes(router fiber.Router, controller *controllers.ReaderController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	reader := router.Group("/reader")

	reader.Get("/:id/bootstrap", middlewares.OptionalJwtAccess(userRepo), controller.GetBootstrap)
	reader.Get("/:id/chapter/:chapterId", middlewares.OptionalJwtAccess(userRepo), controller.GetChapter)
	reader.Get("/:id/file", middlewares.OptionalJwtAccess(userRepo), controller.GetFile)
	reader.Get("/:id/images", middlewares.OptionalJwtAccess(userRepo), controller.ListImages)
	reader.Post("/:id/cover", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookEdit, middlewares.BookLibraryAttr(bookRepo, "id")), controller.UpdateCover)
	// Assets are one request per comic page — a 200-page volume is 200 hits that
	// each re-open the archive. The ETag lets a re-read return 304 instead.
	reader.Get("/:id/asset/*", etag.New(), middlewares.OptionalJwtAccess(userRepo), controller.GetAsset)
}
