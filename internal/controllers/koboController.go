package controllers

import (
	"context"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/middlewares"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/kobo"

	"github.com/gofiber/fiber/v3"
)

type KoboController struct {
	koboService     services.KoboService
	authService     services.KoboAuthService
	settingsService services.SettingsService
}

func NewKoboController(koboService services.KoboService, authService services.KoboAuthService, settingsService services.SettingsService) *KoboController {
	return &KoboController{koboService: koboService, authService: authService, settingsService: settingsService}
}

func (ctrl *KoboController) GetInitialization(c fiber.Ctx) error {
	c.Set(kobo.APITokenHeader, kobo.APITokenValue)
	return c.JSON(ctrl.koboService.GetInitialization(getBaseURL(c, ctrl.settingsService) + "/kobo/" + middlewares.KoboAuthTokenFrom(c)))
}

func (ctrl *KoboController) AuthDevice(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var payload request.KoboAuthDeviceDto
	_ = jsonx.Unmarshal(c.Body(), &payload)

	res, err := ctrl.koboService.AuthDevice(ctx, payload.UserKey)
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) GetUserProfile(c fiber.Ctx) error {
	return c.JSON(ctrl.koboService.GetUserProfile(getOptionalClaims(c)))
}

func (ctrl *KoboController) GetSyncList(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	userID, _ := getUserIdFromLocals(c)
	res, err := ctrl.koboService.GetSyncList(ctx, request.KoboSyncDto{
		UserID:      userID,
		SyncToken:   c.Get(kobo.SyncTokenHeader),
		EndpointURL: getBaseURL(c, ctrl.settingsService) + "/kobo/" + middlewares.KoboAuthTokenFrom(c),
	}, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	c.Set(kobo.SyncTokenHeader, res.SyncToken)
	if res.Continue {
		c.Set(kobo.SyncContinueHeader, kobo.SyncContinueValue)
	}
	if res.Items == nil {
		return c.JSON([]response.KoboSyncItemResponse{})
	}
	return c.JSON(res.Items)
}

func (ctrl *KoboController) GetBookMetadata(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := ctrl.koboService.GetBookMetadata(ctx, c.Params("uuid"), getBaseURL(c, ctrl.settingsService)+"/kobo/"+middlewares.KoboAuthTokenFrom(c), getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) GetReadingState(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	userID, _ := getUserIdFromLocals(c)
	res, err := ctrl.koboService.GetReadingState(ctx, userID, c.Params("uuid"), getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) PutReadingState(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var dto request.PutKoboStateDto
	if err := jsonx.Unmarshal(c.Body(), &dto); err != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Malformed reading state request"))
	}

	userID, _ := getUserIdFromLocals(c)
	res, err := ctrl.koboService.PutReadingState(ctx, userID, c.Params("uuid"), dto, getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(res)
}

func (ctrl *KoboController) GetCoverImage(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path, err := ctrl.koboService.GetCoverPath(ctx, c.Params("uuid"), getOptionalClaims(c))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	c.Set("X-Content-Type-Options", "nosniff")
	return c.SendFile(path)
}

func (ctrl *KoboController) DownloadKePub(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	bookID := c.Params("id")
	if bookID == "" {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Invalid book ID"))
	}

	c.Set("Content-Type", "application/epub+zip")
	c.Set("Content-Disposition", `attachment; filename="book.kepub.epub"`)

	return ctrl.koboService.GetBookKePubStream(ctx, bookID, getOptionalClaims(c), c.Response().BodyWriter())
}

func (ctrl *KoboController) ArchiveBook(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, _ := getUserIdFromLocals(c)
	if err := ctrl.koboService.ArchiveBook(ctx, userID, c.Params("uuid"), getOptionalClaims(c)); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{Status: true})
}

func (ctrl *KoboController) GetSetup(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}
	res, err := ctrl.authService.EnsureSetup(ctx, claims.UId, getBaseURL(c, ctrl.settingsService))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: res})
}

func (ctrl *KoboController) RegenerateSetup(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}
	res, err := ctrl.authService.RegenerateSetup(ctx, claims.UId, getBaseURL(c, ctrl.settingsService))
	if err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "Kobo endpoint regenerated", Data: res})
}

func (ctrl *KoboController) RevokeSetup(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	claims, ok := getUserClaims(c)
	if !ok {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}
	if err := ctrl.authService.RevokeToken(ctx, claims.UId); err != nil {
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Message: "Kobo endpoint revoked"})
}
