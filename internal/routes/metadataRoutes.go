package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
)

func RegisterMetadataRoutes(api fiber.Router, metadataController *controllers.MetadataController, userRepo repositories.UserRepository) {
	group := api.Group("/metadata", middlewares.OptionalJwtAccess(userRepo))

	group.Get("/authors", metadataController.ListAuthors)
	group.Get("/series", metadataController.ListSeries)
	group.Get("/publishers", metadataController.ListPublishers)
	group.Get("/languages", metadataController.ListLanguages)
	group.Get("/tags", metadataController.ListTags)
	group.Get("/formats", metadataController.ListFormats)
}
