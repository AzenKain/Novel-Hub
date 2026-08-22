package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/services"
)

// The Mihon Komga extension appends /api/v1 itself and only attaches Basic credentials after a
// 401 (its OkHttp authenticator is challenge-driven), so returning 403 here breaks auth entirely.
func KomgaRoutes(app fiber.Router, komgaController *controllers.KomgaController, authService services.AuthService, settingsService services.SettingsService) {
	group := app.Group("/komga", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), middlewares.KomgaAuth(authService, settingsService))

	v1 := group.Group("/api/v1")
	v1.Get("/users/me", komgaController.GetUserMe)
	v1.Get("/libraries", komgaController.ListLibraries)
	v1.Get("/series", komgaController.ListSeries)
	v1.Get("/series/:seriesId", komgaController.GetSeries)
	v1.Get("/series/:seriesId/books", komgaController.ListSeriesBooks)
	v1.Get("/series/:seriesId/thumbnail", komgaController.GetSeriesThumbnail)
	v1.Get("/books/:bookId", komgaController.GetBook)
	v1.Get("/books/:bookId/pages", komgaController.ListBookPages)
	v1.Get("/books/:bookId/pages/:pageNumber", komgaController.GetBookPage)
	v1.Get("/books/:bookId/thumbnail", komgaController.GetBookThumbnail)

	// Dual-routes for progress tracker under /api/v1 as well as /api/v2
	v1.Get("/series/:seriesId/read-progress/tachiyomi", komgaController.GetSeriesProgressV1)
	v1.Put("/series/:seriesId/read-progress/tachiyomi", komgaController.UpdateSeriesProgress)

	// Book-level read progress
	v1.Get("/books/:bookId/read-progress", komgaController.GetBookReadProgress)
	v1.Patch("/books/:bookId/read-progress", komgaController.UpdateBookReadProgress)
	v1.Delete("/books/:bookId/read-progress", komgaController.DeleteBookReadProgress)

	// Read lists (playlists)
	v1.Get("/readlists", komgaController.ListReadLists)
	v1.Get("/readlists/:readListId", komgaController.GetReadList)
	v1.Get("/readlists/:readListId/books", komgaController.GetReadListBooks)

	// Progress sync is called by Mihon's built-in tracker on /api/v2 as well
	v2 := group.Group("/api/v2")
	v2.Get("/series/:seriesId/read-progress/tachiyomi", komgaController.GetSeriesProgress)
	v2.Put("/series/:seriesId/read-progress/tachiyomi", komgaController.UpdateSeriesProgress)
}
