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
	app.Get("/reader/stats/:id", featureController.GetBookReadStats)
	app.Get("/books/:id/download-stats", featureController.GetBookDownloadStats)
	app.Get("/books/:id/engagement", featureController.GetBookEngagementStats)
	app.Get("/books/:id/rating", featureController.GetBookRatingSummary)
	app.Get("/books/:id/reviews", featureController.ListBookReviews)
	app.Post("/books/:id/share", featureController.RecordBookShare)

	collectionGroup := app.Group("/collections", middlewares.JwtAccess(userRepo))
	collectionGroup.Use(middlewares.RequirePermission(permissionCache, "book.collection"))
	collectionGroup.Get("/", featureController.GetCollections)
	collectionGroup.Post("/", featureController.CreateCollection)
	collectionGroup.Put("/:id", featureController.UpdateCollection)
	collectionGroup.Delete("/:id", featureController.DeleteCollection)
	collectionGroup.Post("/:id/books", featureController.AddBookToCollection)
	collectionGroup.Delete("/:id/books/:bookId", featureController.RemoveBookFromCollection)

	bookmarkGroup := app.Group("/bookmarks", middlewares.JwtAccess(userRepo))
	bookmarkGroup.Use(middlewares.RequirePermission(permissionCache, "book.bookmark"))
	bookmarkGroup.Get("/books", featureController.GetBookmarkedBooks)
	bookmarkGroup.Put("/:id", middlewares.RequirePermission(permissionCache, "book.bookmark", middlewares.BookLibraryAttr(bookRepo, "id")), featureController.SetBookmark)

	bookUserGroup := app.Group("/books", middlewares.JwtAccess(userRepo))
	bookUserGroup.Get("/:id/user-state", middlewares.RequirePermission(permissionCache, "book.read", middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookUserState)
	bookUserGroup.Put("/:id/review", middlewares.RequirePermission(permissionCache, "book.review.create", middlewares.BookLibraryAttr(bookRepo, "id")), featureController.UpsertBookReview)
	bookUserGroup.Delete("/:id/review", middlewares.RequirePermission(permissionCache, "book.review.create", middlewares.BookLibraryAttr(bookRepo, "id")), featureController.DeleteBookReview)

	readerHistoryGroup := app.Group("/reader/history", middlewares.JwtAccess(userRepo))
	readerHistoryGroup.Use(middlewares.RequirePermission(permissionCache, "book.read"))
	readerHistoryGroup.Get("/", featureController.GetRecentReadingHistory)
	readerHistoryGroup.Get("/progress/:bookId", featureController.GetReadingProgress)
	readerHistoryGroup.Post("/", featureController.RecordReadingActivity)

	// Reading Stats & Heatmap
	statsGroup := app.Group("/reader/stats", middlewares.JwtAccess(userRepo))
	statsGroup.Use(middlewares.RequirePermission(permissionCache, "book.read"))
	statsGroup.Post("/session", featureController.RecordReadingSession)
	statsGroup.Get("/heatmap", featureController.GetReadingHeatmap)

	adminReviewGroup := app.Group("/admin/reviews", middlewares.JwtAccess(userRepo))
	adminReviewGroup.Use(middlewares.RequirePermission(permissionCache, "book.review.delete"))
	adminReviewGroup.Get("/", featureController.ListAllReviews)
	adminReviewGroup.Delete("/:bookId/:userId", featureController.AdminDeleteReview)
}
