package controllers

import (
	"context"
	"io"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type SettingsController struct {
	service services.SettingsService
	audit   services.AuditService
}

func NewSettingsController(service services.SettingsService, audit services.AuditService) *SettingsController {
	return &SettingsController{service: service, audit: audit}
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := h.service.Admin(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: settings})
}

func (h *SettingsController) UpdateSettings(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	dto := &request.UpdateSettingsDto{}
	if errs := validator.ValidateBodyDto(c, dto); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}
	values := dto.Values()
	settings, err := h.service.UpdateSettings(ctx, values)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	h.audit.Record(ctx, services.AuditActionSettingsUpdate, "settings", "", services.SettingsAuditLabel(values))
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: settings})
}

func (h *SettingsController) TestSMTP(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	dto := &request.SMTPTestDto{}
	if errs := validator.ValidateBodyDto(c, dto); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}
	if err := h.service.TestSMTP(ctx, dto); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "SMTP connection succeeded"})
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if !h.service.SetupRequired(ctx) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Setup already completed"})
	}

	return h.handleAssetUpload(ctx, c)
}

func (h *SettingsController) UploadAdminLogo(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
			// The client's part parsed fine; failing to open the spooled copy is our side.
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrInternalError, "Failed to open uploaded file"))
		}
		defer f.Close()

		limit := h.service.Limits().SiteAssetBytes
		fileData, err = io.ReadAll(io.LimitReader(f, limit+1))
		if err != nil {
			return apperrors.HandleError(c, err)
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
