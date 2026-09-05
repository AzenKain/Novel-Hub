package routes

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func OPDSRoutes(app fiber.Router, opdsController *controllers.OPDSController, authService services.AuthService, settingsService services.SettingsService, userRepo repositories.UserRepository) {
	opdsCors := func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Next()
	}

	redirectHandler := func(c fiber.Ctx) error {
		accept := c.Get(fiber.HeaderAccept)
		base := controllers.GetOPDSBasePath(c)
		if strings.Contains(accept, "application/opds+json") {
			return c.Redirect().To(base + "/v2/catalog")
		}
		return c.Redirect().To(base + "/v1")
	}

	app.Get("/opds", opdsCors, redirectHandler)
	app.Get("/opds/", opdsCors, redirectHandler)
	app.Options("/opds", opdsCors)
	app.Options("/opds/*", opdsCors)

	rateLimit := middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS)
	auth := middlewares.OPDSAuth(authService, settingsService, userRepo)

	v1 := app.Group("/opds/v1", opdsCors, rateLimit, auth)
	v1.Get("", opdsController.GetRootCatalog)
	v1.Get("/", opdsController.GetRootCatalog)
	v1.Get("/opensearch.xml", opdsController.GetOpenSearchDescription)
	v1.Get("/search", opdsController.SearchCatalog)
	v1.Get("/books", opdsController.GetAllBooks)
	v1.Get("/recent", opdsController.GetRecentBooks)
	v1.Get("/hot", opdsController.GetHotBooks)
	v1.Get("/random", opdsController.GetRandomBooks)
	v1.Get("/authors", opdsController.GetAuthorsCatalog)
	v1.Get("/authors/:name", opdsController.GetAuthorBooks)
	v1.Get("/series", opdsController.GetSeriesCatalog)
	v1.Get("/series/:name", opdsController.GetSeriesBooks)
	v1.Get("/tags", opdsController.GetTagsCatalog)
	v1.Get("/tags/:name", opdsController.GetTagBooks)
	v1.Get("/books/:id/download", opdsController.DownloadBook)

	// Covers do not consume OPDS rate limit budget
	v1Cover := app.Group("/opds/v1", opdsCors, auth)
	v1Cover.Get("/books/:id/cover", opdsController.GetBookCover)

	v2 := app.Group("/opds/v2", opdsCors, rateLimit, auth)
	v2.Get("", func(c fiber.Ctx) error {
		return c.Redirect().To(controllers.GetOPDSBasePath(c) + "/v2/catalog")
	})
	v2.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().To(controllers.GetOPDSBasePath(c) + "/v2/catalog")
	})
	v2.Get("/catalog", opdsController.GetOPDS2Catalog)
	v2.Get("/books", opdsController.GetOPDS2AllBooks)
	v2.Get("/recent", opdsController.GetOPDS2RecentBooks)
	v2.Get("/hot", opdsController.GetOPDS2HotBooks)
	v2.Get("/random", opdsController.GetOPDS2RandomBooks)
	v2.Get("/authors", opdsController.GetOPDS2AuthorsCatalog)
	v2.Get("/authors/:name", opdsController.GetOPDS2AuthorBooks)
	v2.Get("/series", opdsController.GetOPDS2SeriesCatalog)
	v2.Get("/series/:name", opdsController.GetOPDS2SeriesBooks)
	v2.Get("/tags", opdsController.GetOPDS2TagsCatalog)
	v2.Get("/tags/:name", opdsController.GetOPDS2TagBooks)
	v2.Get("/search", opdsController.GetOPDS2Search)
	v2.Get("/books/:id/download", opdsController.DownloadBook)

	v2Cover := app.Group("/opds/v2", opdsCors, auth)
	v2Cover.Get("/books/:id/cover", opdsController.GetBookCover)
}
