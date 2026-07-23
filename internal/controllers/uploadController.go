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

type UploadController struct {
	uploadService services.UploadService
}

func NewUploadController(uploadService services.UploadService) *UploadController {
	return &UploadController{uploadService: uploadService}
}

func (h *UploadController) InitUpload(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	claims, ok := getUserClaims(c)
	if !ok {
		return fiber.ErrUnauthorized
	}
	var dto request.InitUploadDto
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	uploadID, err := h.uploadService.InitUploadSession(ctx, &dto, claims)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: map[string]string{"upload_id": uploadID}})
}

func (h *UploadController) UploadChunk(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	claims, ok := getUserClaims(c)
	if !ok {
		return fiber.ErrUnauthorized
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Missing file chunk"})
	}
	if err := h.uploadService.SaveChunk(ctx, c.Params("uploadId"), c.FormValue("chunk_index"), file, claims); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true})
}

func (h *UploadController) CommitUpload(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	claims, ok := getUserClaims(c)
	if !ok {
		return fiber.ErrUnauthorized
	}
	var dto request.CommitUploadDto
	if err := validator.ValidateBodyDto(c, &dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	if err := h.uploadService.CommitUpload(ctx, c.Params("uploadId"), &dto, claims); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "File uploaded successfully"})
}
