package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type IntegrationsController struct {
	service services.IntegrationsService
}

func NewIntegrationsController(service services.IntegrationsService) *IntegrationsController {
	return &IntegrationsController{service: service}
}

func (c *IntegrationsController) ExportHighlightsToReadwise(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.ExportHighlightsDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	count, err := c.service.ExportHighlightsToReadwise(reqCtx, userID, dto.BookID, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   map[string]any{"exported": count},
	})
}

func (c *IntegrationsController) ExportHighlightsMarkdown(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	markdown, err := c.service.ExportHighlightsMarkdown(reqCtx, userID, ctx.Params("book_id"), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	ctx.Set(fiber.HeaderContentType, "text/markdown; charset=utf-8")
	ctx.Set(fiber.HeaderContentDisposition, `attachment; filename="highlights.md"`)
	return ctx.SendString(markdown)
}

func (c *IntegrationsController) ExportHighlightsAnki(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	apkgBytes, bookTitle, err := c.service.ExportHighlightsAnki(reqCtx, userID, ctx.Params("book_id"), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	filename := "highlights.apkg"
	if bookTitle != "" {
		filename = fmt.Sprintf("%s_highlights.apkg", sanitizeFilename(bookTitle))
	}

	ctx.Set(fiber.HeaderContentType, "application/apkg")
	ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
	return ctx.Send(apkgBytes)
}

func (c *IntegrationsController) ExportHighlightsCSV(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	csvStr, err := c.service.ExportHighlightsCSV(reqCtx, userID, ctx.Params("book_id"), getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	ctx.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	ctx.Set(fiber.HeaderContentDisposition, `attachment; filename="highlights.csv"`)
	return ctx.SendString(csvStr)
}

func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, name)
}