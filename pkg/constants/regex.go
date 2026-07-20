package constants

import (
	"errors"
	"regexp"
)

var (
	hasUpper  = regexp.MustCompile(`[A-Z]`)
	hasLower  = regexp.MustCompile(`[a-z]`)
	hasNumber = regexp.MustCompile(`\d`)
	hasSymbol = regexp.MustCompile(`[!@#$%^&*()_+{}|:<>?~-]`)

	EMAIL_REGEX = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
)

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if !hasUpper.MatchString(password) {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower.MatchString(password) {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber.MatchString(password) {
		return errors.New("password must contain at least one number")
	}
	if !hasSymbol.MatchString(password) {
		return errors.New("password must contain at least one special character")
	}
	return nil
}
