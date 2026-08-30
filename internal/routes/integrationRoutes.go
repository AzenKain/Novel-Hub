package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func IntegrationRoutes(app fiber.Router, integrationsController *controllers.IntegrationsController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	trackerGroup := app.Group("/trackers", middlewares.JwtAccess(userRepo))
	trackerGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermTrackerSync))
	trackerGroup.Post("/readwise/export", integrationsController.ExportHighlightsToReadwise)

	app.Get("/highlights/:book_id/export.md", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookHighlight, middlewares.BookLibraryAttr(bookRepo, "book_id")), integrationsController.ExportHighlightsMarkdown)
	app.Get("/highlights/:book_id/export.apkg", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookHighlight, middlewares.BookLibraryAttr(bookRepo, "book_id")), integrationsController.ExportHighlightsAnki)
	app.Get("/highlights/:book_id/export.csv", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookHighlight, middlewares.BookLibraryAttr(bookRepo, "book_id")), integrationsController.ExportHighlightsCSV)
}