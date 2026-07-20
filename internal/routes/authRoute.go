package routes

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"novelhub/internal/controllers"
	"novelhub/internal/middlewares"
	"novelhub/internal/repositories"
)

func AuthRoutes(app fiber.Router, controller *controllers.AuthController, userRepo repositories.UserRepository) {
	route := app.Group("/auth")
	
	authLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
	})

	route.Post("/signin", authLimiter, controller.Signin)
	route.Post("/register", authLimiter, controller.Register)
	route.Post("/refresh", middlewares.JwtRefresh(userRepo), controller.RefreshToken)
	route.Post("/logout", middlewares.JwtAccess(userRepo), controller.Logout)

	app.Post("/setup", authLimiter, controller.SubmitSetup)
}
