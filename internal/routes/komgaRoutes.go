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
	v1.Get("/libraries", komgaController.ListLibraries)
	v1.Get("/series", komgaController.ListSeries)
	v1.Get("/series/:seriesId", komgaController.GetSeries)
	v1.Get("/series/:seriesId/books", komgaController.ListSeriesBooks)
	v1.Get("/series/:seriesId/thumbnail", komgaController.GetSeriesThumbnail)
	v1.Get("/books/:bookId", komgaController.GetBook)
	v1.Get("/books/:bookId/pages", komgaController.ListBookPages)
	v1.Get("/books/:bookId/pages/:pageNumber", komgaController.GetBookPage)
	v1.Get("/books/:bookId/thumbnail", komgaController.GetBookThumbnail)

	// Progress sync is called by Mihon's built-in tracker, a different client from the extension.
	v2 := group.Group("/api/v2")
	v2.Get("/series/:seriesId/read-progress/tachiyomi", komgaController.GetSeriesProgress)
	v2.Put("/series/:seriesId/read-progress/tachiyomi", komgaController.UpdateSeriesProgress)
}
