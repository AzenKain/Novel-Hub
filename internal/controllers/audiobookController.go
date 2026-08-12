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

type AudiobookController struct {
	service services.AudiobookService
}

func NewAudiobookController(service services.AudiobookService) *AudiobookController {
	return &AudiobookController{service: service}
}

func (c *AudiobookController) ListChapters(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chapters, err := c.service.ListChapters(reqCtx, ctx.Params("book_id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: chapters})
}

func (c *AudiobookController) UpsertChapter(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpsertAudiobookChapterDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	chapter, err := c.service.UpsertChapter(reqCtx, ctx.Params("book_id"), services.UpsertAudiobookChapterInput{
		ID:           ctx.Params("id"),
		FileID:       dto.FileID,
		ChapterIndex: dto.ChapterIndex,
		Title:        dto.Title,
		StartSec:     dto.StartSec,
		EndSec:       dto.EndSec,
	})
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: chapter})
}

func (c *AudiobookController) DeleteChapter(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.service.DeleteChapter(reqCtx, ctx.Params("book_id"), ctx.Params("id")); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true})
}

func (c *AudiobookController) DeleteAllChapters(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.service.DeleteChaptersForBook(reqCtx, ctx.Params("book_id")); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true})
}

func (c *AudiobookController) LookupChapters(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dto := &request.LookupAudiobookChaptersDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	chapters, err := c.service.LookupChaptersFromAudnexus(reqCtx, ctx.Params("book_id"), dto.ASIN)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: chapters})
}

func (c *AudiobookController) MergeAudio(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.MergeAudioDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	jobID, err := c.service.MergeAudio(reqCtx, ctx.Params("book_id"), dto.Title, dto.FileIDs)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: map[string]string{"job_id": jobID}})
}