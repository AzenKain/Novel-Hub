package validator

import (
	"errors"
	"net/url"
	"path"
	"reflect"
	"strings"

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
}

func isImageURL(value string) bool {
	if strings.HasPrefix(value, "data:image/") {
		return true
	}
	
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
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
			case "image_url":
				message = fieldError.Field() + " must be a link to an image"
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
