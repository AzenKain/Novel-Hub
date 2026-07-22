package controllers

import (
	"context"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"

	"github.com/gofiber/fiber/v3"
)

type KoboController struct {
	koboService services.KoboService
}

func NewKoboController(koboService services.KoboService) *KoboController {
	return &KoboController{
		koboService: koboService,
	}
}

func (ctrl *KoboController) GetInitialization(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := ctrl.koboService.GetInitialization(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) GetUserProfile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := ctrl.koboService.GetUserProfile(ctx)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) GetSyncList(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token := c.Query("SyncToken", "")
	res, err := ctrl.koboService.GetSyncList(ctx, token)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) DownloadKePub(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bookID := c.Params("id")
	if bookID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid book ID"})
	}

	c.Set("Content-Type", "application/epub+zip")
	c.Set("Content-Disposition", `attachment; filename="book.kepub.epub"`)

	return ctrl.koboService.GetBookKePubStream(ctx, bookID, c.Response().BodyWriter())
}

func (ctrl *KoboController) SyncState(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stateData map[string]any
	if err := c.Bind().Body(&stateData); err != nil {
		stateData = make(map[string]any)
	}

	if err := ctrl.koboService.SyncState(ctx, 0, stateData); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
