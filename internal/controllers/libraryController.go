package controllers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"novelhub/pkg/apperrors"

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
		return apperrors.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.CommonResponse{Status: true, Data: lib})
}

func (h *LibraryController) GetLibrary(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	lib, err := h.libraryService.GetLibrary(ctx, id, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: lib})
}

func (h *LibraryController) ListLibraries(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	libs, err := h.libraryService.ListLibraries(ctx, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
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
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Data: lib})
}

func (h *LibraryController) DeleteLibrary(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")
	err := h.libraryService.DeleteLibrary(ctx, id)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Library deleted successfully"})
}

func (h *LibraryController) UploadFiles(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Uploaded and queued files successfully", Data: result})
}

func (h *LibraryController) DownloadLibraryZip(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	libraryID := c.Params("id")
	if libraryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Library ID is required"})
	}

	_, err := h.libraryService.GetLibrary(ctx, libraryID, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", `attachment; filename="library_`+libraryID+`.zip"`)

	pr, pw := io.Pipe()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				pw.CloseWithError(fmt.Errorf("panic: %v", r))
			}
		}()
		streamCtx, cancelStream := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancelStream()
		err := h.libraryService.StreamLibraryZip(streamCtx, libraryID, pw)
		pw.CloseWithError(err)
	}()

	return c.SendStream(pr)
}
