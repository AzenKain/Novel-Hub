package routes

import (
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
)

func NotFoundRoute(app *fiber.App) {
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Not found"})
	})
}
