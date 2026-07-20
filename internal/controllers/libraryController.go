package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/validator"
)

type LibraryController struct {
	libraryService services.LibraryService
}

func NewLibraryController(libraryService services.LibraryService) *LibraryController {
	return &LibraryController{
		libraryService: libraryService,
	}
}

func (h *LibraryController) CreateLibrary(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.CreateLibraryDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	lib, err := h.libraryService.CreateLibrary(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: lib})
}

func (h *LibraryController) GetLibrary(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	lib, err := h.libraryService.GetLibrary(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Library not found"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: lib})
}

func (h *LibraryController) ListLibraries(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	libs, err := h.libraryService.ListLibraries(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: libs})
}

func (h *LibraryController) UpdateLibrary(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	dto := &request.UpdateLibraryDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	lib, err := h.libraryService.UpdateLibrary(ctx, id, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}

	return c.JSON(response.CommonResponse{Status: true, Data: lib})
}

func (h *LibraryController) DeleteLibrary(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	err := h.libraryService.DeleteLibrary(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "An internal error occurred"})
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Library deleted successfully"})
}

func (h *LibraryController) UploadFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid multipart form"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "No files provided"})
	}

	result, err := h.libraryService.UploadFiles(ctx, id, files)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return c.Status(fiber.StatusRequestTimeout).JSON(response.CommonResponse{Status: false, Message: "Upload timed out"})
		}
		return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "Library not found"})
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Uploaded and queued files successfully", Data: result})
}
