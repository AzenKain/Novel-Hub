package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func LibraryRoutes(router fiber.Router, libraryController *controllers.LibraryController, userRepo repositories.UserRepository, permissionCache services.PermissionCache) {
	libraryGroup := router.Group("/libraries")

	libraryGroup.Post("/", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "library.manage"), libraryController.CreateLibrary)
	libraryGroup.Get("/", middlewares.OptionalJwtAccess(userRepo), libraryController.ListLibraries)
	libraryGroup.Get("/:id", middlewares.OptionalJwtAccess(userRepo), libraryController.GetLibrary)
	libraryGroup.Post("/:id/upload", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "library.manage", middlewares.LibraryIDParam("id")), libraryController.UploadFiles)
	libraryGroup.Put("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "library.manage", middlewares.LibraryIDParam("id")), libraryController.UpdateLibrary)
	libraryGroup.Delete("/:id", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "library.manage", middlewares.LibraryIDParam("id")), libraryController.DeleteLibrary)
	libraryGroup.Post("/:id/inbox/setup", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, "library.manage", middlewares.LibraryIDParam("id")), libraryController.SetupInbox)
}
