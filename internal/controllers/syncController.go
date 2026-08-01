package controllers

import (
	"context"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/validator"

	"github.com/gofiber/fiber/v3"
)

type SyncController struct {
	syncService services.SyncService
}

func NewSyncController(syncService services.SyncService) *SyncController {
	return &SyncController{syncService: syncService}
}

func getSyncUserID(c fiber.Ctx) string {
	uidRaw := c.Locals("uid")
	if uidRaw != nil {
		if uidStr, ok := uidRaw.(string); ok {
			return uidStr
		}
	}
	return ""
}

func (ctrl *SyncController) GetProgress(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("bookId")
	userID := getSyncUserID(c)

	res, err := ctrl.syncService.GetProgress(ctx, userID, bookID, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: res})
}

func (ctrl *SyncController) UpdateProgress(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ProgressSyncDto{}
	if errs := validator.ValidateBodyDto(c, dto); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	userID := getSyncUserID(c)
	res, err := ctrl.syncService.UpdateProgress(ctx, userID, dto, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: res})
}

func (ctrl *SyncController) KosyncGetProgress(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	document := c.Params("document")
	userID := getSyncUserID(c)

	res, err := ctrl.syncService.KosyncGetProgress(ctx, userID, document, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *SyncController) KosyncPushProgress(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.KosyncPushProgressDto{}
	if errs := validator.ValidateBodyDto(c, dto); errs != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	userID := getSyncUserID(c)
	res, err := ctrl.syncService.KosyncPushProgress(ctx, userID, dto, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}
