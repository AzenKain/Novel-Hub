package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func VBookRoutes(app fiber.Router, vbookController *controllers.VBookController, authService services.AuthService, settingsService services.SettingsService, userRepo repositories.UserRepository) {
	pub := app.Group("/v1/vbook", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS))
	pub.Get("/plugin.json", vbookController.GetPluginJSON)
	pub.Get("/plugin.zip", vbookController.GetPluginZip)
	pub.Get("/plugin-audio.zip", vbookController.GetPluginZipAudio)

	v1 := app.Group("/v1/vbook", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), middlewares.VBookAuth(authService, settingsService, userRepo))
	v1.Get("/home", vbookController.GetHome)
	v1.Get("/genres", vbookController.GetGenres)
	v1.Get("/books", vbookController.GetBooks)
	v1.Get("/search", vbookController.SearchBooks)
	v1.Get("/detail", vbookController.GetDetail)
	v1.Get("/toc", vbookController.GetTOC)
	v1.Get("/chap", vbookController.GetChapterContent)
	v1.Get("/audio/books", vbookController.GetAudioBooks)
	v1.Get("/audio/playlist", vbookController.GetAudioPlaylist)
	v1.Get("/audio/stream", vbookController.StreamAudio)
}

