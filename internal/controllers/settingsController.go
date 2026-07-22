package controllers

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

type SettingsController struct {
	service services.SettingsService
}

func NewSettingsController(service services.SettingsService) *SettingsController {
	return &SettingsController{service: service}
}

func (h *SettingsController) PublicSettings(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := h.service.Public(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: settings})
}

func (h *SettingsController) AdminSettings(c fiber.Ctx) error {
	return h.PublicSettings(c)
}

func (h *SettingsController) UpdateSettings(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := map[string]any{}
	if err := c.Bind().Body(&dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid settings payload"})
	}
	settings, err := h.service.UpdateSettings(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: settings})
}

func (h *SettingsController) SetupStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data: map[string]bool{
			"required": h.service.SetupRequired(ctx),
		},
	})
}

func (h *SettingsController) UploadSetupLogo(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if !h.service.SetupRequired(ctx) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Setup already completed"})
	}

	return h.handleAssetUpload(ctx, c)
}

func (h *SettingsController) UploadAdminLogo(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return h.handleAssetUpload(ctx, c)
}

func (h *SettingsController) handleAssetUpload(ctx context.Context, c fiber.Ctx) error {
	target := c.FormValue("target", "logo")
	urlStr := c.FormValue("url")

	var fileData []byte
	var fileName string

	fileHeader, err := c.FormFile("file")
	if err == nil && fileHeader != nil {
		f, err := fileHeader.Open()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Failed to open uploaded file"})
		}
		defer f.Close()
		
		fileData, err = io.ReadAll(io.LimitReader(f, 5<<20))
		if err != nil {
			return apperrors.HandleError(c, err)
		}

		contentType := http.DetectContentType(fileData)
		if !strings.HasPrefix(contentType, "image/") {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Uploaded file must be a valid image"})
		}

		fileName = filepath.Base(fileHeader.Filename)
	}

	assetURL, err := h.service.SaveAsset(ctx, target, fileData, fileName, urlStr)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   map[string]string{"url": assetURL},
	})
}
