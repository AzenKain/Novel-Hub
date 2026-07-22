package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"
)

type UploadController struct {
	uploadService services.UploadService
}

func NewUploadController(uploadService services.UploadService) *UploadController {
	return &UploadController{
		uploadService: uploadService,
	}
}

type InitUploadRequest struct {
	// Add fields if needed
}

func (h *UploadController) InitUpload(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uploadID, err := h.uploadService.InitUploadSession(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status: true,
		Data: map[string]string{
			"upload_id": uploadID,
		},
	})
}

func (h *UploadController) UploadChunk(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	uploadID := c.Params("uploadId")
	if uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Missing upload ID"})
	}

	chunkIndexStr := c.FormValue("chunk_index")
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Missing file chunk"})
	}

	err = h.uploadService.SaveChunk(ctx, uploadID, chunkIndexStr, file)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true})
}

type CommitRequest struct {
	Target      string `json:"target"`
	LibraryID   string `json:"library_id"`
	BookID      string `json:"book_id"`
	Filename    string `json:"filename"`
	TotalChunks int    `json:"total_chunks"`
}

func (h *UploadController) CommitUpload(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	uploadID := c.Params("uploadId")
	if uploadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Missing upload ID"})
	}

	var req CommitRequest
	if err := validator.ValidateBodyDto(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	err := h.uploadService.CommitUpload(ctx, uploadID, req.Target, req.LibraryID, req.BookID, req.Filename, req.TotalChunks)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "File uploaded successfully"})
}
