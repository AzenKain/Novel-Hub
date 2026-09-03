package controllers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type CalibreServerController struct {
	service services.CalibreServerService
}

func NewCalibreServerController(service services.CalibreServerService) *CalibreServerController {
	return &CalibreServerController{service: service}
}

func (h *CalibreServerController) GetLibraryInfo(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := h.service.GetLibraryInfo(ctx, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(info)
}

func (h *CalibreServerController) GetCategories(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	libraryID := c.Params("library_id")
	categories, err := h.service.GetCategories(ctx, libraryID, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(categories)
}

func (h *CalibreServerController) GetCategory(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var dto request.CalibrePaginationDto
	if err := validator.ValidateQueryDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	libraryID := c.Params("library_id")
	encodedName := c.Params("encoded_name")

	res, err := h.service.GetCategory(ctx, libraryID, encodedName, dto.Num, dto.Offset, dto.Sort, dto.SortOrder, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *CalibreServerController) GetBooksInCategory(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var dto request.CalibrePaginationDto
	if err := validator.ValidateQueryDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	libraryID := c.Params("library_id")
	encodedCategory := c.Params("encoded_category")
	encodedItem := c.Params("encoded_item")

	res, err := h.service.GetBooksInCategory(ctx, libraryID, encodedCategory, encodedItem, dto.Num, dto.Offset, dto.Sort, dto.SortOrder, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *CalibreServerController) Search(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var dto request.CalibreSearchQueryDto
	if err := validator.ValidateQueryDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	libraryID := c.Params("library_id")
	res, err := h.service.SearchBooks(ctx, libraryID, dto.Query, dto.Num, dto.Offset, dto.Sort, dto.SortOrder, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *CalibreServerController) GetBooks(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var dto request.CalibreBooksQueryDto
	if err := validator.ValidateQueryDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var bookIDs []string
	if strings.TrimSpace(dto.IDs) != "" && dto.IDs != "all" {
		for _, id := range strings.Split(dto.IDs, ",") {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				bookIDs = append(bookIDs, trimmed)
			}
		}
	}

	if len(bookIDs) > 100 {
		bookIDs = bookIDs[:100]
	}

	libraryID := c.Params("library_id")
	res, err := h.service.GetBooksMetadata(ctx, libraryID, bookIDs, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *CalibreServerController) GetBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("book_id")
	libraryID := c.Params("library_id")

	res, err := h.service.GetBookMetadata(ctx, libraryID, bookID, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (h *CalibreServerController) GetContent(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	what := strings.ToLower(c.Params("what"))
	rawBookID := c.Params("book_id")
	bookID, _, _ := strings.Cut(rawBookID, "_")

	claims := getOptionalClaims(c)

	if what == "cover" || what == "thumb" {
		filePath, err := h.service.GetBookCover(ctx, bookID, what == "thumb", claims)
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		c.Set("Cache-Control", "public, max-age=86400")
		return c.SendFile(filePath)
	}

	filePath, filename, err := h.service.GetBookFile(ctx, bookID, what, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.Download(filePath, filename)
}
