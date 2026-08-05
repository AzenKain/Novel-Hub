package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func AuthRoutes(app fiber.Router, controller *controllers.AuthController, userRepo repositories.UserRepository, settingsService services.SettingsService) {
	route := app.Group("/auth")

	authLimiter := middlewares.RateLimit(settingsService, middlewares.RateLimitAuth)

	route.Post("/signin", authLimiter, controller.Signin)
	route.Post("/register", authLimiter, controller.Register)
	route.Post("/otp/request", authLimiter, controller.RequestOTP)
	route.Post("/otp/verify", authLimiter, controller.VerifyOTP)
	route.Post("/password/reset", authLimiter, controller.ResetPasswordWithOTP)
	route.Post("/refresh", middlewares.JwtRefresh(userRepo), controller.RefreshToken)
	route.Post("/logout", middlewares.JwtAccess(userRepo), controller.Logout)

	app.Post("/setup", authLimiter, controller.SubmitSetup)
}
