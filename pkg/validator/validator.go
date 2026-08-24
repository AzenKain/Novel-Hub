package validator

import (
	"errors"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			name = strings.SplitN(fld.Tag.Get("query"), ",", 2)[0]
		}
		return name
	})

	_ = validate.RegisterValidation("image_url", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}
		return isImageURL(value)
	})

	_ = validate.RegisterValidation("server_url", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}
		return isServerURL(value)
	})

	_ = validate.RegisterValidation("readlist_cursor", func(fl validator.FieldLevel) bool {
		return isReadListCursor(fl.Field().String())
	})
}

func isServerURL(value string) bool {
	text := strings.TrimSpace(value)
	if text == "" {
		return true
	}
	if strings.ContainsAny(text, "\r\n") {
		return false
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if strings.Trim(parsed.Path, "/") != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return true
}

func isImageURL(value string) bool {
	if strings.HasPrefix(value, "/public/") {
		switch strings.ToLower(path.Ext(value)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			return true
		default:
			return false
		}
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func isReadListCursor(value string) bool {
	if value == "" {
		return true
	}
	parts := strings.SplitN(value, "|", 2)
	_, err := time.Parse(time.RFC3339Nano, parts[0])
	return err == nil
}

type ErrorResponse struct {
	FailedField string `json:"failed_field,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Value       string `json:"value,omitempty"`
	Message     string `json:"message"`
}

func formatValidationError(err error) []*ErrorResponse {
	var validationErrors validator.ValidationErrors
	errorsList := []*ErrorResponse{}

	if errors.As(err, &validationErrors) {
		for _, fieldError := range validationErrors {
			message := ""
			switch fieldError.Tag() {
			case "required":
				message = fieldError.Field() + " is mandatory"
			case "email":
				message = "The email address is invalid"
			case "min":
				message = fieldError.Field() + " is too short (min " + fieldError.Param() + ")"
			case "max":
				message = fieldError.Field() + " is too long (max " + fieldError.Param() + ")"
			case "url":
				message = fieldError.Field() + " must be a valid URL"
			case "server_url":
				message = fieldError.Field() + " must be a valid http or https URL"
			case "image_url":
				message = fieldError.Field() + " must be a link to an image"
			case "readlist_cursor":
				message = fieldError.Field() + " is an invalid cursor format"
			case "oneof":
				message = fieldError.Field() + " must be one of: " + fieldError.Param()
			case "uuid":
				message = fieldError.Field() + " must be a valid UUID"
			case "numeric":
				message = fieldError.Field() + " must be numeric"
			case "len":
				message = fieldError.Field() + " must be exactly " + fieldError.Param() + " characters"
			case "gte":
				message = fieldError.Field() + " must be at least " + fieldError.Param()
			case "lte":
				message = fieldError.Field() + " must be at most " + fieldError.Param()
			default:
				message = "Field " + fieldError.Field() + " failed validation: " + fieldError.Tag()
			}
			errorsList = append(errorsList, &ErrorResponse{
				FailedField: fieldError.Field(),
				Tag:         fieldError.Tag(),
				Value:       fieldError.Param(),
				Message:     message,
			})
		}
	} else {
		errorsList = append(errorsList, &ErrorResponse{
			Message: "Invalid request payload: " + err.Error(),
		})
	}

	return errorsList
}

func ValidateQueryDto(c fiber.Ctx, dto any) []*ErrorResponse {
	if err := c.Bind().Query(dto); err != nil {
		return formatValidationError(err)
	}
	if err := validate.Struct(dto); err != nil {
		return formatValidationError(err)
	}
	return nil
}

func ValidateBodyDto(c fiber.Ctx, dto any) []*ErrorResponse {
	if err := c.Bind().Body(dto); err != nil {
		return formatValidationError(err)
	}
	if err := validate.Struct(dto); err != nil {
		return formatValidationError(err)
	}
	return nil
}
