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

type PodcastController struct {
	service services.PodcastService
}

func NewPodcastController(service services.PodcastService) *PodcastController {
	return &PodcastController{service: service}
}

func (c *PodcastController) ListPodcasts(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	podcasts, err := c.service.ListPodcasts(reqCtx)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: podcasts})
}

func (c *PodcastController) Subscribe(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dto := &request.SubscribePodcastDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	podcast, err := c.service.Subscribe(reqCtx, dto.FeedURL, dto.LibraryID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: podcast})
}

func (c *PodcastController) UpdatePodcast(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.UpdatePodcastDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	podcast, err := c.service.UpdatePodcast(reqCtx, ctx.Params("id"), dto.AutoDownload)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: podcast})
}

func (c *PodcastController) DeletePodcast(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.service.DeletePodcast(reqCtx, ctx.Params("id")); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true})
}

func (c *PodcastController) ListEpisodes(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	episodes, err := c.service.ListEpisodes(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: episodes})
}

func (c *PodcastController) RefreshPodcast(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobID, err := c.service.QueueRefreshPodcast(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: map[string]string{"job_id": jobID}})
}

func (c *PodcastController) DownloadEpisode(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobID, err := c.service.DownloadEpisode(reqCtx, ctx.Params("id"), ctx.Params("episode_id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: map[string]string{"job_id": jobID}})
}
