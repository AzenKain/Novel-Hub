package apperrors

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/worker"
)

// HandleError is the single place a domain error becomes an HTTP status.
//
// Matching is by errors.Is, not by comparing AppError.Err, so a kind stays matchable after being
// wrapped with %w. The sql.ErrNoRows case is what stops a missing row from becoming a 500:
// repositories return it raw by convention (no repository imports this package), so without it
// every "book not found" surfaced as 500 with "sql: no rows in result set" as the message — a
// storage detail on the wire, and the wrong status for a client to act on.
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
		// Raw driver text names the storage engine and says nothing a client can use.
		message = "Not found"
	case errors.Is(err, worker.ErrQueueFull):
		// A saturated worker pool is transient, so the client needs a status it can retry on
		// rather than the 500 an unmatched error would otherwise get.
		code = fiber.StatusServiceUnavailable
	}

	return c.Status(code).JSON(response.CommonResponse{Status: false, Message: message})
}
