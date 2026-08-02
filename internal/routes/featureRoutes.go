package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func FeatureRoutes(app fiber.Router, featureController *controllers.FeatureController, highlightController *controllers.HighlightController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	highlightGroup := app.Group("/highlights", middlewares.JwtAccess(userRepo))
	highlightGroup.Post("/", highlightController.CreateHighlight)
	highlightGroup.Get("/", highlightController.GetHighlights)
	highlightGroup.Put("/:id", highlightController.UpdateHighlightNote)
	highlightGroup.Delete("/:id", highlightController.DeleteHighlight)

	app.Get("/library/stats", featureController.GetLibraryStats)

	historyGroup := app.Group("/reader/history", middlewares.JwtAccess(userRepo))
	historyGroup.Get("/", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetRecentReadingHistory)
	historyGroup.Post("/", featureController.RecordReadingActivity)
	historyGroup.Get("/progress/:id", featureController.GetReadingProgress)

	statsGroup := app.Group("/reader/stats", middlewares.JwtAccess(userRepo))
	statsGroup.Post("/session", featureController.RecordReadingSession)
	statsGroup.Get("/heatmap", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReadingHeatmap)

	goalGroup := app.Group("/reader/goals", middlewares.JwtAccess(userRepo))
	goalGroup.Get("/", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReadingGoal)
	goalGroup.Put("/", featureController.UpsertReadingGoal)

	app.Get("/reader/stats/:id", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookReadStats)
	app.Get("/books/:id/download-stats", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookDownloadStats)
	app.Get("/books/:id/engagement", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookEngagementStats)
	app.Get("/books/:id/rating", featureController.GetBookRatingSummary)
	app.Get("/books/:id/reviews", featureController.ListBookReviews)
	app.Post("/books/:id/share", middlewares.OptionalJwtAccess(userRepo), featureController.RecordBookShare)

	collectionGroup := app.Group("/collections", middlewares.JwtAccess(userRepo))
	collectionGroup.Get("/", featureController.GetCollections)
	collectionGroup.Post("/", featureController.CreateCollection)
	collectionGroup.Put("/:id", featureController.UpdateCollection)
	collectionGroup.Delete("/:id", featureController.DeleteCollection)
	collectionGroup.Post("/:id/books", featureController.AddBookToCollection)
	collectionGroup.Delete("/:id/books/:bookId", featureController.RemoveBookFromCollection)

	smartCollectionGroup := app.Group("/smart-collections", middlewares.JwtAccess(userRepo))
	smartCollectionGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookCollection))
	smartCollectionGroup.Get("/", featureController.ListSmartCollections)
	smartCollectionGroup.Post("/", featureController.CreateSmartCollection)
	smartCollectionGroup.Put("/:id", featureController.UpdateSmartCollection)
	smartCollectionGroup.Delete("/:id", featureController.DeleteSmartCollection)

	bookmarkGroup := app.Group("/bookmarks", middlewares.JwtAccess(userRepo))
	bookmarkGroup.Get("/books", featureController.GetBookmarkedBooks)
	bookmarkGroup.Put("/:id", featureController.SetBookmark)

	app.Get("/books/:id/user-state", middlewares.JwtAccess(userRepo), featureController.GetBookUserState)
	app.Put("/books/:id/review", middlewares.JwtAccess(userRepo), featureController.UpsertBookReview)
	app.Delete("/books/:id/review", middlewares.JwtAccess(userRepo), featureController.DeleteBookReview)

	adminReviewGroup := app.Group("/admin/reviews", middlewares.JwtAccess(userRepo))
	adminReviewGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookReviewDelete))
	adminReviewGroup.Get("/", featureController.ListAllReviews)
	adminReviewGroup.Delete("/:bookId/:userId", featureController.AdminDeleteReview)
}
