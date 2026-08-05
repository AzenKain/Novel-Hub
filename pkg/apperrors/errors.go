package apperrors

import (
	"database/sql"
	"errors"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrBadRequest      = errors.New("bad request")
	ErrTooManyRequests = errors.New("too many requests")
	ErrInternalError   = errors.New("internal server error")
)

type AppError struct {
	Err     error
	Message string
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(err error, message string) error {
	return &AppError{Err: err, Message: message}
}

// Repositories are inconsistent: some return sql.ErrNoRows raw, some wrap it.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound)
}
