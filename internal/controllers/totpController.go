package controllers

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type TOTPController struct {
	service services.TOTPService
	users   services.UserService
	audit   services.AuditService
}

func NewTOTPController(service services.TOTPService, users services.UserService, audit services.AuditService) *TOTPController {
	return &TOTPController{service: service, users: users, audit: audit}
}

func (h *TOTPController) GetStatus(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.Status(ctx, claims.UId)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *TOTPController) BeginEnrollment(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}
	user, err := h.users.GetUserByID(ctx, claims.UId)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	res, err := h.service.BeginEnrollment(ctx, claims.UId, user.Email)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *TOTPController) ConfirmEnrollment(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	dto := &request.TOTPCodeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.ConfirmEnrollment(ctx, claims.UId, dto.Code)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	h.audit.Record(ctx, services.AuditActionTOTPEnable, "user", claims.UId, "")
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *TOTPController) Disable(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	dto := &request.TOTPCodeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	if err := h.service.Disable(ctx, claims.UId, dto.Code); err != nil {
		return apperrors.HandleError(c, err)
	}
	h.audit.Record(ctx, services.AuditActionTOTPDisable, "user", claims.UId, "")
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "Two-factor authentication disabled"})
}

func (h *TOTPController) RegenerateRecoveryCodes(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	dto := &request.TOTPCodeDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	res, err := h.service.RegenerateRecoveryCodes(ctx, claims.UId, dto.Code)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}
