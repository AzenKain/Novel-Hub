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

type AuditController struct {
	service services.AuditService
}

func NewAuditController(service services.AuditService) *AuditController {
	return &AuditController{service: service}
}

func (h *AuditController) ListAuditLogs(c fiber.Ctx) error {
	var req request.ListAuditLogsDto
	if errs := validator.ValidateQueryDto(c, &req); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	res, err := h.service.List(ctx, &req)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

func (h *AuditController) ListAuditActions(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 10*time.Second)
	defer cancel()

	actions, err := h.service.ListActions(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: actions})
}
