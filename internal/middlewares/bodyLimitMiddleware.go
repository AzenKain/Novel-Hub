package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func RequestBodyLimit(settings services.SettingsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		limit := int64(constants.MaxDefaultRequestBody)
		path := c.Path()
		limits := settings.Limits()
		switch {
		case strings.HasPrefix(path, "/api/v1/upload/") && strings.HasSuffix(path, "/chunk"):
			limit = limits.UploadChunkBytes + constants.MultipartBodyOverhead
		case path == "/api/v1/settings/logo" || path == "/api/v1/setup/logo":
			limit = limits.SiteAssetBytes + constants.MultipartBodyOverhead
		case strings.HasPrefix(path, "/api/v1/reader/") && strings.HasSuffix(path, "/cover"):
			limit = limits.CoverBytes + constants.MultipartBodyOverhead
		case path == "/api/v1/soundscapes/upload":
			limit = limits.SoundscapeBytes + constants.MultipartBodyOverhead
		case path == "/api/v1/fonts/upload":
			limit = limits.FontBytes + constants.MultipartBodyOverhead
		}
		length := int64(c.RequestCtx().Request.Header.ContentLength())
		if actual := int64(len(c.Body())); actual > length {
			length = actual
		}
		if length > limit {
			return fiber.ErrRequestEntityTooLarge
		}
		return c.Next()
	}
}
