package apperrors

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
)

func HandleError(c fiber.Ctx, err error) error {
	var appErr *AppError
	code := fiber.StatusInternalServerError
	if errors.As(err, &appErr) {
		switch appErr.Err {
		case ErrBadRequest:
			code = fiber.StatusBadRequest
		case ErrNotFound:
			code = fiber.StatusNotFound
		case ErrConflict:
			code = fiber.StatusConflict
		case ErrUnauthorized:
			code = fiber.StatusUnauthorized
		case ErrForbidden:
			code = fiber.StatusForbidden
		}
	}
	return c.Status(code).JSON(response.CommonResponse{Status: false, Message: err.Error()})
}
