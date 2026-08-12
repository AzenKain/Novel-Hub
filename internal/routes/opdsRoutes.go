package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func OPDSRoutes(app fiber.Router, opdsController *controllers.OPDSController, authService services.AuthService, settingsService services.SettingsService, _ repositories.UserRepository) {
	v1 := app.Group("/opds/v1", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), middlewares.OPDSAuth(authService, settingsService))
	v1.Get("/", opdsController.GetRootCatalog)
	v1.Get("/opensearch.xml", opdsController.GetOpenSearchDescription)
	v1.Get("/search", opdsController.SearchCatalog)
	v1.Get("/recent", opdsController.GetRecentBooks)
	v1.Get("/authors", opdsController.GetAuthorsCatalog)
	v1.Get("/authors/:name", opdsController.GetAuthorBooks)
	v1.Get("/series", opdsController.GetSeriesCatalog)
	v1.Get("/series/:name", opdsController.GetSeriesBooks)
	v1.Get("/tags", opdsController.GetTagsCatalog)
	v1.Get("/tags/:name", opdsController.GetTagBooks)

	v2 := app.Group("/opds/v2", middlewares.RateLimit(settingsService, middlewares.RateLimitOPDS), middlewares.OPDSAuth(authService, settingsService))
	v2.Get("/catalog", opdsController.GetOPDS2Catalog)
}

