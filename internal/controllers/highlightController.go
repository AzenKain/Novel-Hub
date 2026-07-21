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

type HighlightController struct {
	service services.HighlightService
}

func NewHighlightController(service services.HighlightService) *HighlightController {
	return &HighlightController{service: service}
}

func (c *HighlightController) CreateHighlight(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.CreateHighlightDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := c.service.CreateHighlight(reqCtx, userID, dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (c *HighlightController) GetHighlights(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	chapterID := ctx.Query("chapterId")
	if chapterID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "chapterId query parameter is required",
		})
	}

	res, err := c.service.GetHighlights(reqCtx, userID, chapterID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (c *HighlightController) UpdateHighlightNote(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	id := ctx.Params("id")
	dto := &request.UpdateHighlightNoteDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := c.service.UpdateHighlightNote(reqCtx, userID, id, dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (c *HighlightController) DeleteHighlight(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	id := ctx.Params("id")
	err := c.service.DeleteHighlight(reqCtx, userID, id)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
	})
}
