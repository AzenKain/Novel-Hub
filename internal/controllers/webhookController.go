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

type WebhookController struct {
	service services.WebhookService
}

func NewWebhookController(service services.WebhookService) *WebhookController {
	return &WebhookController{service: service}
}

func (h *WebhookController) ListWebhooks(c fiber.Ctx) error {
	dto := request.PaginationDto{Limit: 20}
	if errs := validator.ValidateQueryDto(c, &dto); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, total, err := h.service.ListAll(ctx, int64(dto.Limit), int64(dto.Offset))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	page := int(dto.Offset/dto.Limit) + 1
	return c.Status(fiber.StatusOK).JSON(response.BuildPaginatedResponse(list, total, page, int(dto.Limit)))
}

func (h *WebhookController) CreateWebhook(c fiber.Ctx) error {
	var req request.CreateWebhookDto
	if errs := validator.ValidateBodyDto(c, &req); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.Create(ctx, &req)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *WebhookController) GetWebhook(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "missing webhook id"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetByID(ctx, id)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *WebhookController) UpdateWebhook(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "missing webhook id"))
	}

	var req request.UpdateWebhookDto
	if errs := validator.ValidateBodyDto(c, &req); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.Update(ctx, id, &req)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Data: res})
}

func (h *WebhookController) DeleteWebhook(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "missing webhook id"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.service.Delete(ctx, id); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "webhook deleted successfully"})
}

func (h *WebhookController) TestPing(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "missing webhook id"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := h.service.TestPing(ctx, id); err != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true, Message: "ping test sent successfully"})
}
