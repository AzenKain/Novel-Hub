package controllers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"novelhub/pkg/config"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
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
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to load settings"})
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
	settings, ferr := h.service.UpdateSettings(ctx, dto)
	if ferr != nil {
		return c.Status(ferr.Code).JSON(response.CommonResponse{Status: false, Message: ferr.Message})
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !h.service.SetupRequired(ctx) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Setup already completed"})
	}

	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	os.MkdirAll(publicDir, 0755)
	destPath := filepath.Join(publicDir, "logo.png")

	// Try file upload
	file, err := c.FormFile("file")
	if err == nil {
		if err := c.SaveFile(file, destPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to save file"})
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/logo.png"}})
	}

	// Try URL
	urlStr := c.FormValue("url")
	if urlStr != "" {
		resp, err := http.Get(urlStr)
		if err != nil || resp.StatusCode != 200 {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Failed to fetch URL"})
		}
		defer resp.Body.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to save file"})
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to copy data"})
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/logo.png"}})
	}

	return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Provide a file or URL"})
}

func (h *SettingsController) UploadAdminLogo(c fiber.Ctx) error {
	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	os.MkdirAll(publicDir, 0755)
	
	// Create a unique name to prevent cache issues, or just use logo.png and rely on cache busting
	destPath := filepath.Join(publicDir, "logo.png")

	// Try file upload
	file, err := c.FormFile("file")
	if err == nil {
		if err := c.SaveFile(file, destPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "Failed to save file"})
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/logo.png"}})
	}

	return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Provide a file"})
}

