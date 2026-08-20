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

type SmartFilterController struct {
	service     services.FeatureService
	bookService services.BookService
}

func NewSmartFilterController(service services.FeatureService, bookService services.BookService) *SmartFilterController {
	return &SmartFilterController{
		service:     service,
		bookService: bookService,
	}
}

func (h *SmartFilterController) ListSmartFilters(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	filters, err := h.service.ListSmartFilters(ctx, claims.UId)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status: true,
		Data:   filters,
	})
}

func (h *SmartFilterController) GetSmartFilter(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	id := c.Params("id")
	filter, err := h.service.GetSmartFilter(ctx, id, claims.UId)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status: true,
		Data:   filter,
	})
}

func (h *SmartFilterController) CreateSmartFilter(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	dto := request.UpsertSmartFilterDto{}
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.CreateSmartFilter(ctx, claims.UId, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{
		Status:  true,
		Data:    res,
		Message: "Smart filter created successfully",
	})
}

func (h *SmartFilterController) UpdateSmartFilter(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	id := c.Params("id")
	dto := request.UpsertSmartFilterDto{}
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.UpdateSmartFilter(ctx, id, claims.UId, dto)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status:  true,
		Data:    res,
		Message: "Smart filter updated successfully",
	})
}

func (h *SmartFilterController) DeleteSmartFilter(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	id := c.Params("id")
	if err := h.service.DeleteSmartFilter(ctx, id, claims.UId); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status:  true,
		Message: "Smart filter deleted successfully",
	})
}

func (h *SmartFilterController) PinSidebar(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	id := c.Params("id")
	dto := request.PinSmartFilterDto{}
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.UpdateSmartFilterPinSidebar(ctx, id, claims.UId, dto.IsPinned)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status:  true,
		Data:    res,
		Message: "Smart filter sidebar pin updated successfully",
	})
}

func (h *SmartFilterController) PinHome(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	id := c.Params("id")
	dto := request.PinSmartFilterDto{}
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.service.UpdateSmartFilterPinHome(ctx, id, claims.UId, dto.IsPinned)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status:  true,
		Data:    res,
		Message: "Smart filter homepage pin updated successfully",
	})
}

func (h *SmartFilterController) ReorderHome(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
	}

	dto := request.ReorderHomeShelvesDto{}
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	if err := h.service.ReorderSmartFiltersHome(ctx, claims.UId, dto); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status:  true,
		Message: "Homepage shelves reordered successfully",
	})
}

func (h *SmartFilterController) GetSmartFilterBooks(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, _ := getUserClaims(c)
	var userUId string
	if claims != nil {
		userUId = claims.UId
	}

	id := c.Params("id")

	queryDto := &request.SearchBookDto{}
	queryDto.Limit = 20
	queryDto.Offset = 0
	if err := validator.ValidateQueryDto(c, queryDto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	res, err := h.bookService.SearchSmartFilterBooksByFilter(ctx, id, userUId, queryDto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(res)
}
