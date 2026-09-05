package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func FeatureRoutes(app fiber.Router, featureController *controllers.FeatureController, highlightController *controllers.HighlightController, readListController *controllers.ReadListController, userRepo repositories.UserRepository, bookRepo repositories.BookDBRepository, permissionCache services.PermissionCache) {
	highlightGroup := app.Group("/highlights", middlewares.JwtAccess(userRepo))
	highlightGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookHighlight))
	highlightGroup.Post("/", highlightController.CreateHighlight)
	highlightGroup.Get("/", highlightController.GetHighlights)
	highlightGroup.Put("/:id", highlightController.UpdateHighlightNote)
	highlightGroup.Delete("/:id", highlightController.DeleteHighlight)

	app.Get("/library/stats", middlewares.OptionalJwtAccess(userRepo), featureController.GetLibraryStats)
	app.Get("/library/stats/breakdown", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermLibraryRead), featureController.GetLibraryBreakdown)

	historyGroup := app.Group("/reader/history", middlewares.JwtAccess(userRepo))
	historyGroup.Get("/", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetRecentReadingHistory)
	historyGroup.Post("/", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.RecordReadingActivity)
	historyGroup.Get("/progress/:id", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReadingProgress)

	statsGroup := app.Group("/reader/stats", middlewares.JwtAccess(userRepo))
	statsGroup.Post("/session", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.RecordReadingSession)
	statsGroup.Get("/heatmap", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReadingHeatmap)
	statsGroup.Get("/summary", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReadingStatsSummary)
	statsGroup.Get("/eta/:book_id", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReaderETA)

	goalGroup := app.Group("/reader/goals", middlewares.JwtAccess(userRepo))
	goalGroup.Get("/", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.GetReadingGoal)
	goalGroup.Put("/", middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead), featureController.UpsertReadingGoal)

	app.Get("/reader/stats/:id", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookReadStats)
	app.Get("/books/:id/download-stats", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookDownloadStats)
	app.Get("/books/:id/engagement", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermUserStatsRead, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.GetBookEngagementStats)
	app.Get("/books/:id/rating", middlewares.OptionalJwtAccess(userRepo), featureController.GetBookRatingSummary)
	app.Get("/books/:id/reviews", middlewares.OptionalJwtAccess(userRepo), featureController.ListBookReviews)
	app.Post("/books/:id/share", middlewares.OptionalJwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookShare, middlewares.BookLibraryAttr(bookRepo, "id")), featureController.RecordBookShare)

	collectionGroup := app.Group("/collections", middlewares.JwtAccess(userRepo))
	collectionGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookCollection))
	collectionGroup.Get("/", featureController.GetCollections)
	collectionGroup.Post("/", featureController.CreateCollection)
	collectionGroup.Put("/:id", featureController.UpdateCollection)
	collectionGroup.Delete("/:id", featureController.DeleteCollection)
	collectionGroup.Post("/:id/books", featureController.AddBookToCollection)
	collectionGroup.Delete("/:id/books/:bookId", featureController.RemoveBookFromCollection)

	readListGroup := app.Group("/read-lists", middlewares.JwtAccess(userRepo))
	readListGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookCollection))
	readListGroup.Get("/", readListController.GetReadLists)
	readListGroup.Post("/", readListController.CreateReadList)
	readListGroup.Post("/import", readListController.ImportCBL)
	readListGroup.Get("/:id", readListController.GetReadList)
	readListGroup.Put("/:id", readListController.UpdateReadList)
	readListGroup.Delete("/:id", readListController.DeleteReadList)
	readListGroup.Get("/:id/books", readListController.GetReadListBooks)
	readListGroup.Post("/:id/books", readListController.AddBook)
	readListGroup.Delete("/:id/books/:bookId", readListController.RemoveBook)
	readListGroup.Put("/:id/order", readListController.Reorder)
	readListGroup.Get("/:id/next", readListController.NextInOrder)

	smartCollectionGroup := app.Group("/smart-collections", middlewares.JwtAccess(userRepo))
	smartCollectionGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookCollection))
	smartCollectionGroup.Get("/", featureController.ListSmartCollections)
	smartCollectionGroup.Post("/", featureController.CreateSmartCollection)
	smartCollectionGroup.Put("/:id", featureController.UpdateSmartCollection)
	smartCollectionGroup.Delete("/:id", featureController.DeleteSmartCollection)

	bookmarkGroup := app.Group("/bookmarks", middlewares.JwtAccess(userRepo))
	bookmarkGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookBookmark))
	bookmarkGroup.Get("/books", featureController.GetBookmarkedBooks)
	bookmarkGroup.Put("/:id", featureController.SetBookmark)

	app.Get("/books/:id/user-state", middlewares.JwtAccess(userRepo), featureController.GetBookUserState)
	app.Put("/books/:id/review", middlewares.JwtAccess(userRepo), middlewares.RequirePermission(permissionCache, constants.PermBookReviewCreate), featureController.UpsertBookReview)
	app.Delete("/books/:id/review", middlewares.JwtAccess(userRepo), middlewares.RequireAnyPermission(permissionCache, constants.PermBookReviewCreate, constants.PermBookReviewDelete), featureController.DeleteBookReview)

	adminReviewGroup := app.Group("/admin/reviews", middlewares.JwtAccess(userRepo))
	adminReviewGroup.Use(middlewares.RequirePermission(permissionCache, constants.PermBookReviewDelete))
	adminReviewGroup.Get("/", featureController.ListAllReviews)
	adminReviewGroup.Delete("/:bookId/:userId", featureController.AdminDeleteReview)
}
