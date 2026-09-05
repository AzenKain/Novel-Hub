package apperrors

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/worker"
)

// HandleError is the single place a domain error becomes an HTTP status.
func HandleError(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, ErrBadRequest):
		code = fiber.StatusBadRequest
	case errors.Is(err, ErrConflict):
		code = fiber.StatusConflict
	case errors.Is(err, ErrUnauthorized):
		code = fiber.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		code = fiber.StatusForbidden
	case errors.Is(err, ErrTooManyRequests):
		code = fiber.StatusTooManyRequests
	case errors.Is(err, ErrNotFound):
		code = fiber.StatusNotFound
	case errors.Is(err, sql.ErrNoRows):
		code = fiber.StatusNotFound
		message = "Not found"
	case errors.Is(err, worker.ErrQueueFull):
		code = fiber.StatusServiceUnavailable
	}

	return c.Status(code).JSON(response.CommonResponse{Status: false, Message: message})
}
