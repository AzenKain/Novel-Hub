package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

func (h *BookController) BulkDeleteBooks(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 30*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.BulkDeleteBooksDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := h.bookService.BulkDeleteBooks(ctx, dto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	h.audit.Record(ctx, services.AuditActionBookBulkDelete, "book", "", strconv.Itoa(result.SuccessCount)+" books")

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (h *BookController) BulkMoveBooks(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 30*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.BulkMoveBooksDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := h.bookService.BulkMoveBooks(ctx, dto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	h.audit.Record(ctx, services.AuditActionBookBulkMove, "book", "", strconv.Itoa(result.SuccessCount)+" books")

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (h *BookController) BulkAssignCollections(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.BulkAssignCollectionsDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := h.bookService.BulkAssignCollections(ctx, dto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (h *BookController) BulkAddTags(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.BulkAddTagsDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := h.bookService.BulkAddTags(ctx, dto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (h *BookController) BulkUpdateMetadata(c fiber.Ctx) error {
	ctx, cancel := auditContext(c, 30*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	dto := &request.BulkUpdateMetadataDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	result, err := h.bookService.BulkUpdateMetadata(ctx, dto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	h.audit.Record(ctx, services.AuditActionBookBulkUpdateMetadata, "book", "", strconv.Itoa(result.SuccessCount)+" books")

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

