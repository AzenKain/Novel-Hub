package controllers

import (
	"context"
	"time"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"novelhub/pkg/apperrors"


	"novelhub/internal/services"
)

type MetadataController struct {
	service services.MetadataService
}

func NewMetadataController(service services.MetadataService) *MetadataController {
	return &MetadataController{
		service: service,
	}
}

func (h *MetadataController) ListAuthors(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor := c.Query("cursor", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 { limit = 20 }
	res, err := h.service.ListAuthors(ctx, cursor, int64(limit))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *MetadataController) ListSeries(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor := c.Query("cursor", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 { limit = 20 }
	res, err := h.service.ListSeries(ctx, cursor, int64(limit))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *MetadataController) ListPublishers(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor := c.Query("cursor", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 { limit = 20 }
	res, err := h.service.ListPublishers(ctx, cursor, int64(limit))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *MetadataController) ListLanguages(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor := c.Query("cursor", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 { limit = 20 }
	res, err := h.service.ListLanguages(ctx, cursor, int64(limit))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *MetadataController) ListTags(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor := c.Query("cursor", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 { limit = 20 }
	res, err := h.service.ListTags(ctx, cursor, int64(limit))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *MetadataController) ListFormats(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor := c.Query("cursor", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 { limit = 20 }
	res, err := h.service.ListFormats(ctx, cursor, int64(limit))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}
