package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func FeatureRoutes(app fiber.Router, featureController *controllers.FeatureController, highlightController *controllers.HighlightController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	// Highlights
	highlightGroup := app.Group("/highlights", middlewares.JwtAccess(userRepo))
	highlightGroup.Use(middlewares.RequirePermission(permissionCache, "book.read"))
	highlightGroup.Post("/", highlightController.CreateHighlight)
	highlightGroup.Get("/", highlightController.GetHighlights)
	highlightGroup.Put("/:id", highlightController.UpdateHighlightNote)
	highlightGroup.Delete("/:id", highlightController.DeleteHighlight)

	app.Get("/library/stats", featureController.GetLibraryStats)

	// Reading Stats & Heatmap (Static routes MUST be registered before wildcard :id parameters)
	statsGroup := app.Group("/reader/stats", middlewares.JwtAccess(userRepo))
	statsGroup.Use(middlewares.RequirePermission(permissionCache, "book.read"))
	statsGroup.Post("/session", featureController.RecordReadingSession)
	statsGroup.Get("/heatmap", featureController.GetReadingHeatmap)

	app.Get("/reader/stats/:id", featureController.GetBookReadStats)
	app.Get("/books/:id/download-stats", featureController.GetBookDownloadStats)
	app.Get("/books/:id/engagement", featureController.GetBookEngagementStats)
	app.Get("/books/:id/rating", featureController.GetBookRatingSummary)
	app.Get("/books/:id/reviews", featureController.ListBookReviews)
	app.Post("/books/:id/share", featureController.RecordBookShare)

	adminReviewGroup := app.Group("/admin/reviews", middlewares.JwtAccess(userRepo))
	adminReviewGroup.Use(middlewares.RequirePermission(permissionCache, "book.review.delete"))
	adminReviewGroup.Get("/", featureController.ListAllReviews)
	adminReviewGroup.Delete("/:bookId/:userId", featureController.AdminDeleteReview)
}
