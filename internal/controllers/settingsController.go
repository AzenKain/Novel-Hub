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
	"novelhub/pkg/apperrors"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/netx"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !h.service.SetupRequired(ctx) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Setup already completed"})
	}

	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	os.MkdirAll(publicDir, 0755)

	target := c.FormValue("target", "logo")
	filename := "logo.png"
	if target == "favicon" {
		filename = "favicon.png"
	}
	destPath := filepath.Join(publicDir, filename)

	// Try file upload
	file, err := c.FormFile("file")
	if err == nil {
		if err := c.SaveFile(file, destPath); err != nil {
			return apperrors.HandleError(c, err)
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/" + filename}})
	}

	// Try URL
	urlStr := c.FormValue("url")
	if urlStr != "" {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if reqErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid URL"})
		}
		client := netx.NewSafeHTTPClient(15 * time.Second)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Failed to fetch URL or URL blocked for security"})
		}
		defer resp.Body.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		defer out.Close()

		if _, err := io.Copy(out, io.LimitReader(resp.Body, 10<<20)); err != nil {
			return apperrors.HandleError(c, err)
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/" + filename}})
	}

	return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Provide a file or URL"})
}

func (h *SettingsController) UploadAdminLogo(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	os.MkdirAll(publicDir, 0755)

	target := c.FormValue("target", "logo")
	filename := "logo.png"
	if target == "favicon" {
		filename = "favicon.png"
	}
	destPath := filepath.Join(publicDir, filename)

	// Try file upload
	file, err := c.FormFile("file")
	if err == nil {
		if err := c.SaveFile(file, destPath); err != nil {
			return apperrors.HandleError(c, err)
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/" + filename}})
	}

	// Try URL
	urlStr := c.FormValue("url")
	if urlStr != "" {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if reqErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid URL"})
		}
		client := netx.NewSafeHTTPClient(15 * time.Second)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Failed to fetch URL or URL blocked for security"})
		}
		defer resp.Body.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		defer out.Close()

		if _, err := io.Copy(out, io.LimitReader(resp.Body, 10<<20)); err != nil {
			return apperrors.HandleError(c, err)
		}
		return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: map[string]string{"url": "/public/" + filename}})
	}

	return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Provide a file or URL"})
}
