package controllers

import (
	"context"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/convert"
	"novelhub/pkg/validator"
)

type CustomizationController struct {
	service  services.CustomizationService
	settings services.SettingsService
}

func NewCustomizationController(service services.CustomizationService, settings services.SettingsService) *CustomizationController {
	return &CustomizationController{service: service, settings: settings}
}


func (h *CustomizationController) ListSoundscapes(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queryDto := &request.ListCustomizationQueryDto{Limit: 50}
	if err := validator.ValidateQueryDto(c, queryDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursor *time.Time
	var cursorID string
	if queryDto.Cursor != "" {
		if parts := convert.DecodeCursor(queryDto.Cursor); len(parts) == 2 {
			if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursor = &parsedTime
				cursorID = parts[1]
			}
		}
	}

	claims := getOptionalClaims(c)
	list, nextCursor, err := h.service.ListSoundscapes(ctx, claims, cursor, cursorID, queryDto.Limit)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CursorPaginatedResponse{
		Status:     true,
		Data:       list,
		NextCursor: nextCursor,
	})
}

func (h *CustomizationController) UploadSoundscape(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.UploadSoundscapeDto{
		Name:     strings.TrimSpace(c.FormValue("name")),
		Category: strings.TrimSpace(c.FormValue("category")),
		Icon:     strings.TrimSpace(c.FormValue("icon")),
		AudioURL: strings.TrimSpace(c.FormValue("audio_url")),
		IsSystem: c.FormValue("is_system") == "true" || c.FormValue("is_system") == "1",
	}

	if dto.Name == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Soundscape name is required"))
	}

	file, err := c.FormFile("audio")
	if file == nil && dto.AudioURL == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Audio file or audio URL is required"))
	}

	var fileData []byte
	var filename string
	if file != nil {
		f, err := file.Open()
		if err != nil {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrInternalError, "Failed to open uploaded file"))
		}
		defer f.Close()

		limit := int64(50 * 1024 * 1024)
		if h.settings != nil {
			limit = h.settings.Limits().SoundscapeBytes
		}
		fileData, err = io.ReadAll(io.LimitReader(f, limit+1))
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		if int64(len(fileData)) > limit {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.CommonResponse{Status: false, Message: "Audio file exceeds configured size limit"})
		}
		filename = file.Filename
	}

	res, err := h.service.CreateSoundscape(ctx, claims, dto, fileData, filename)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *CustomizationController) DeleteSoundscape(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Soundscape ID required"))
	}

	if err := h.service.DeleteSoundscape(ctx, id, claims); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Soundscape deleted"})
}

func (h *CustomizationController) StreamSoundscape(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Soundscape ID required"))
	}

	claims := getOptionalClaims(c)
	filePath, _, err := h.service.GetSoundscapeFilePath(ctx, id, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}

	c.Set("Content-Type", mimeType)
	c.Set("Cache-Control", "public, max-age=86400")

	return c.SendFile(filePath, fiber.SendFile{ByteRange: true})
}

func (h *CustomizationController) ListCustomFonts(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queryDto := &request.ListCustomizationQueryDto{Limit: 50}
	if err := validator.ValidateQueryDto(c, queryDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursor *time.Time
	var cursorID string
	if queryDto.Cursor != "" {
		if parts := convert.DecodeCursor(queryDto.Cursor); len(parts) == 2 {
			if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursor = &parsedTime
				cursorID = parts[1]
			}
		}
	}

	claims := getOptionalClaims(c)
	list, nextCursor, err := h.service.ListCustomFonts(ctx, claims, cursor, cursorID, queryDto.Limit)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CursorPaginatedResponse{
		Status:     true,
		Data:       list,
		NextCursor: nextCursor,
	})
}

func (h *CustomizationController) UploadCustomFont(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.UploadFontDto{
		Name:       strings.TrimSpace(c.FormValue("name")),
		FontFamily: strings.TrimSpace(c.FormValue("font_family")),
		SourceType: strings.TrimSpace(c.FormValue("source_type")),
		FontURL:    strings.TrimSpace(c.FormValue("font_url")),
		IsSystem:   c.FormValue("is_system") == "true" || c.FormValue("is_system") == "1",
	}

	if dto.Name == "" || dto.FontFamily == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Font name and font family are required"))
	}
	if dto.SourceType != "file" && dto.SourceType != "url" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Invalid source type (must be 'file' or 'url')"))
	}

	file, err := c.FormFile("font")
	if dto.SourceType == "file" && file == nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Font file is required for file source type"))
	}
	if dto.SourceType == "url" && dto.FontURL == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Font URL is required for url source type"))
	}

	var fileData []byte
	var filename string
	if file != nil {
		f, err := file.Open()
		if err != nil {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrInternalError, "Failed to open uploaded file"))
		}
		defer f.Close()

		limit := int64(20 * 1024 * 1024)
		if h.settings != nil {
			limit = h.settings.Limits().FontBytes
		}
		fileData, err = io.ReadAll(io.LimitReader(f, limit+1))
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		if int64(len(fileData)) > limit {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.CommonResponse{Status: false, Message: "Font file exceeds configured size limit"})
		}
		filename = file.Filename
	}

	res, err := h.service.CreateCustomFont(ctx, claims, dto, fileData, filename)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *CustomizationController) DeleteCustomFont(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Font ID required"))
	}

	if err := h.service.DeleteCustomFont(ctx, id, claims); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Custom font deleted"})
}

func (h *CustomizationController) ServeFontFile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Font ID required"))
	}

	claims := getOptionalClaims(c)
	filePath, _, err := h.service.GetFontFilePath(ctx, id, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var mimeType string
	switch ext {
	case ".woff2":
		mimeType = "font/woff2"
	case ".woff":
		mimeType = "font/woff"
	case ".ttf":
		mimeType = "font/ttf"
	case ".otf":
		mimeType = "font/otf"
	default:
		mimeType = "application/octet-stream"
	}

	c.Set("Content-Type", mimeType)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("Access-Control-Allow-Origin", "*")

	return c.SendFile(filePath, fiber.SendFile{ByteRange: true})
}


func (h *CustomizationController) ListCustomThemes(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queryDto := &request.ListCustomizationQueryDto{Limit: 50}
	if err := validator.ValidateQueryDto(c, queryDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursor *time.Time
	var cursorID string
	if queryDto.Cursor != "" {
		if parts := convert.DecodeCursor(queryDto.Cursor); len(parts) == 2 {
			if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursor = &parsedTime
				cursorID = parts[1]
			}
		}
	}

	claims := getOptionalClaims(c)
	list, nextCursor, err := h.service.ListCustomThemes(ctx, claims, cursor, cursorID, queryDto.Limit)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CursorPaginatedResponse{
		Status:     true,
		Data:       list,
		NextCursor: nextCursor,
	})
}

func (h *CustomizationController) GetCustomTheme(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Theme ID required"))
	}

	claims := getOptionalClaims(c)
	theme, err := h.service.GetCustomTheme(ctx, id, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: theme})
}

func (h *CustomizationController) CreateCustomTheme(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.CreateCustomThemeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.CreateCustomTheme(ctx, claims, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *CustomizationController) UpdateCustomTheme(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Theme ID required"))
	}

	dto := &request.UpdateCustomThemeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.UpdateCustomTheme(ctx, id, claims, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *CustomizationController) DeleteCustomTheme(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims := getOptionalClaims(c)
	if claims == nil || claims.UId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Theme ID required"))
	}

	if err := h.service.DeleteCustomTheme(ctx, id, claims); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Custom theme deleted"})
}
