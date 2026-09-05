package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type MagicCodeController struct {
	service         services.MagicCodeService
	settingsService services.SettingsService
}

func NewMagicCodeController(service services.MagicCodeService, settingsService services.SettingsService) *MagicCodeController {
	return &MagicCodeController{
		service:         service,
		settingsService: settingsService,
	}
}

func (h *MagicCodeController) RequestCode(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.RequestMagicCodeDto{}
	_ = validator.ValidateBodyDto(c, dto)

	baseURL := getBaseURL(c, h.settingsService)
	res, err := h.service.RequestMagicCode(ctx, dto, baseURL)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (h *MagicCodeController) ActivateCode(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ActivateMagicCodeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.ActivateMagicCode(ctx, claims.UId, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Data:    res,
		Message: "Device login activated successfully",
	})
}

func (h *MagicCodeController) PollCode(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.PollMagicCodeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.PollMagicCode(ctx, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}
