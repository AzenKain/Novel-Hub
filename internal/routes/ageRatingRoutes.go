package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func AgeRatingRoutes(
	v1 fiber.Router,
	controller *controllers.AgeRatingController,
	userRepo repositories.UserRepository,
	permissionCache services.PermissionCache,
) {
	v1.Get("/content-warnings", middlewares.JwtAccess(userRepo), controller.GetContentWarnings)
	v1.Get("/books/:id/content-warnings", middlewares.JwtAccess(userRepo), controller.GetBookContentWarnings)
	v1.Put(
		"/books/:id/age-rating",
		middlewares.JwtAccess(userRepo),
		middlewares.RequirePermission(permissionCache, constants.PermBookEdit),
		controller.UpdateBookAgeRating,
	)

	userGroup := v1.Group("/user/kids-mode", middlewares.JwtAccess(userRepo))
	userGroup.Get("/info", controller.GetKidsModeInfo)
	userGroup.Post("/pin", controller.SetKidsModePin)
	userGroup.Post("/toggle", controller.ToggleKidsMode)
}
