package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
)

func AuthRoutes(app fiber.Router, controller *controllers.AuthController, oauthController *controllers.OAuthController, userRepo repositories.UserRepository, settingsService services.SettingsService) {
	route := app.Group("/auth")

	authLimiter := middlewares.RateLimit(settingsService, middlewares.RateLimitAuth)

	route.Post("/signin", authLimiter, controller.Signin)
	route.Post("/register", authLimiter, controller.Register)
	route.Post("/otp/request", authLimiter, controller.RequestOTP)
	route.Post("/otp/verify", authLimiter, controller.VerifyOTP)
	route.Post("/password/reset", authLimiter, controller.ResetPasswordWithOTP)
	route.Post("/refresh", middlewares.JwtRefresh(userRepo), controller.RefreshToken)
	route.Post("/logout", middlewares.JwtAccess(userRepo), controller.Logout)

	route.Get("/oauth2/:provider/login", authLimiter, oauthController.OAuth2Login)
	route.Get("/oauth2/:provider/callback", authLimiter, oauthController.OAuth2Callback)

	app.Post("/setup", authLimiter, controller.SubmitSetup)
}

func MagicCodeRoutes(app fiber.Router, controller *controllers.MagicCodeController, userRepo repositories.UserRepository, settingsService services.SettingsService) {
	route := app.Group("/auth/magic-code")

	authLimiter := middlewares.RateLimit(settingsService, middlewares.RateLimitAuth)

	route.Post("/request", authLimiter, controller.RequestCode)
	route.Post("/poll", authLimiter, controller.PollCode)
	route.Post("/activate", authLimiter, middlewares.JwtAccess(userRepo), controller.ActivateCode)
}
