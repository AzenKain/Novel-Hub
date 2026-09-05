package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func TOTPRoutes(router fiber.Router, controller *controllers.TOTPController, userRepo repositories.UserRepository, settingsService services.SettingsService) {
	group := router.Group("/auth/totp", middlewares.JwtAccess(userRepo), middlewares.RateLimit(settingsService, middlewares.RateLimitAuth))

	group.Get("", controller.GetStatus)
	group.Post("/enroll", controller.BeginEnrollment)
	group.Post("/confirm", controller.ConfirmEnrollment)
	group.Post("/disable", controller.Disable)
	group.Post("/recovery-codes", controller.RegenerateRecoveryCodes)
}
