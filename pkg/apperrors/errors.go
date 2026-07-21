package apperrors

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrBadRequest    = errors.New("bad request")
	ErrInternalError = errors.New("internal server error")
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
