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

type CalibreController struct {
	calibreService services.CalibreSyncService
}

func NewCalibreController(calibreService services.CalibreSyncService) *CalibreController {
	return &CalibreController{calibreService: calibreService}
}

func (c *CalibreController) ImportCalibre(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dto := &request.ImportCalibreDto{}
	if errs := validator.ValidateBodyDto(ctx, dto); errs != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: errs,
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
