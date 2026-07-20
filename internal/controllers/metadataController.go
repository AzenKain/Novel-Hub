package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
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

	authors, err := h.service.ListAuthors(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: authors})
}

func (h *MetadataController) ListSeries(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	series, err := h.service.ListSeries(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: series})
}

func (h *MetadataController) ListPublishers(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	publishers, err := h.service.ListPublishers(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: publishers})
}

func (h *MetadataController) ListLanguages(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	languages, err := h.service.ListLanguages(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: languages})
}

func (h *MetadataController) ListTags(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tags, err := h.service.ListTags(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: tags})
}

func (h *MetadataController) ListFormats(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	formats, err := h.service.ListFormats(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}
	return c.JSON(response.CommonResponse{Status: true, Data: formats})
}
