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

type MetadataController struct {
	service services.MetadataService
}

func NewMetadataController(service services.MetadataService) *MetadataController {
	return &MetadataController{
		service: service,
	}
}

func (h *MetadataController) listFacet(c fiber.Ctx, fetch func(context.Context, *request.MetadataFacetDto, *response.JWTClaims) (*response.PaginatedResponse, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.MetadataFacetDto{Limit: 20}
	if errs := validator.ValidateQueryDto(c, dto); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	res, err := fetch(ctx, dto, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *MetadataController) ListAuthors(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListAuthors)
}

func (h *MetadataController) ListSeries(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListSeries)
}

func (h *MetadataController) ListPublishers(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListPublishers)
}

func (h *MetadataController) ListLanguages(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListLanguages)
}

func (h *MetadataController) ListTags(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListTags)
}

func (h *MetadataController) ListFormats(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListFormats)
}

func (h *MetadataController) ListRatings(c fiber.Ctx) error {
	return h.listFacet(c, h.service.ListRatings)
}
