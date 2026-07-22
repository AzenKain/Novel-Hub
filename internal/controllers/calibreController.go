package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

type CalibreController struct {
	calibreService services.CalibreSyncService
}

func NewCalibreController(calibreService services.CalibreSyncService) *CalibreController {
	return &CalibreController{calibreService: calibreService}
}

type ImportCalibreDto struct {
	Path      string `json:"path"`
	LibraryID string `json:"library_id"`
}

func (c *CalibreController) ImportCalibre(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var dto ImportCalibreDto
	if err := ctx.Bind().Body(&dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "invalid payload",
		})
	}

	if dto.Path == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "path is required",
		})
	}

	count, err := c.calibreService.ImportCalibreLibrary(reqCtx, dto.Path, dto.LibraryID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status:  true,
		Message: "Calibre library imported successfully",
		Data:    map[string]any{"imported_count": count},
	})
}
