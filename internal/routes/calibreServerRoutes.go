package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func CalibreServerRoutes(
	app *fiber.App,
	controller *controllers.CalibreServerController,
	authService services.AuthService,
	settingsService services.SettingsService,
	userRepo repositories.UserRepository,
) {
	calibre := app.Group("/calibre", middlewares.CalibreAuth(authService, settingsService, userRepo))

	calibre.Get("/ajax/library-info", controller.GetLibraryInfo)

	calibre.Get("/ajax/categories", controller.GetCategories)
	calibre.Get("/ajax/categories/:library_id", controller.GetCategories)

	calibre.Get("/ajax/category/:encoded_name", controller.GetCategory)
	calibre.Get("/ajax/category/:encoded_name/:library_id", controller.GetCategory)

	calibre.Get("/ajax/books_in/:encoded_category/:encoded_item", controller.GetBooksInCategory)
	calibre.Get("/ajax/books_in/:encoded_category/:encoded_item/:library_id", controller.GetBooksInCategory)

	calibre.Get("/ajax/search", controller.Search)
	calibre.Get("/ajax/search/:library_id", controller.Search)

	calibre.Get("/ajax/books", controller.GetBooks)
	calibre.Get("/ajax/books/:library_id", controller.GetBooks)

	calibre.Get("/ajax/book/:book_id", controller.GetBook)
	calibre.Get("/ajax/book/:book_id/:library_id", controller.GetBook)

	calibre.Get("/get/:what/:book_id", controller.GetContent)
	calibre.Get("/get/:what/:book_id/:library_id", controller.GetContent)
}
