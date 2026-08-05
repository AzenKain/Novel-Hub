package controllers

import (
	"context"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/convert"
	"novelhub/pkg/validator"

	"github.com/gofiber/fiber/v3"
)

type TrackerController struct {
	trackerService services.TrackerService
	featureService services.FeatureService
}

func NewTrackerController(trackerService services.TrackerService, featureService services.FeatureService) *TrackerController {
	return &TrackerController{
		trackerService: trackerService,
		featureService: featureService,
	}
}

func (ctrl *TrackerController) ConnectTracker(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.ConnectTrackerDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	userID, err := convert.ParseID(claims.UId)
	if err != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Invalid user ID"))
	}

	if err := ctrl.trackerService.SaveUserTracker(ctx, userID, dto.Provider, dto.AccessToken); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Tracker connected successfully"})
}

func (ctrl *TrackerController) bookReadable(ctx context.Context, bookID string, claims *response.JWTClaims) bool {
	return ctrl.featureService != nil && ctrl.featureService.PolicyAllowsBook(ctx, "read", bookID, claims)
}

func (ctrl *TrackerController) MapBookTracker(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.MapTrackerDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}
	if !ctrl.bookReadable(ctx, dto.BookID, claims) {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrForbidden, "Book is not accessible"))
	}

	if err := ctrl.trackerService.SaveBookMapping(ctx, claims.UId, dto.BookID, dto.Provider, dto.ExternalSeriesID); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Book mapped successfully"})
}

func (ctrl *TrackerController) SearchAniList(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := c.Query("title")
	if title == "" {
		title = c.Query("query")
	}
	if title == "" {
		title = c.Query("q")
	}
	if title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Title query is required"})
	}

	results, err := ctrl.trackerService.SearchAniListMedia(ctx, title)
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{
		Status: true,
		Data:   fiber.Map{"results": results},
	})
}

func (ctrl *TrackerController) SyncProgress(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dto := &request.SyncProgressDto{}
	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	claims, ok := getUserClaims(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}

	userID, err := convert.ParseID(claims.UId)
	if err != nil {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Invalid user ID"))
	}

	if !ctrl.bookReadable(ctx, dto.BookID, claims) {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrForbidden, "Book is not accessible"))
	}

	progress := dto.Progress
	if progress <= 0 {
		if ctrl.featureService != nil {
			readingProgress, _ := ctrl.featureService.GetReadingProgress(ctx, userID, dto.BookID)
			if readingProgress != nil && readingProgress.ChapterIndex >= 0 {
				progress = int(readingProgress.ChapterIndex) + 1
			}
		}
	}

	if progress <= 0 {
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrBadRequest, "Progress must be 1 or greater"))
	}

	mediaID, err := ctrl.trackerService.GetOrMapBookTrackerID(ctx, userID, dto.BookID, dto.Title, "anilist")
	if err != nil {
		return apperrors.HandleError(c, err)
	}

	if err := ctrl.trackerService.SyncAniListProgress(ctx, userID, mediaID, progress); err != nil {
		return apperrors.HandleError(c, err)
	}

	return c.JSON(response.CommonResponse{Status: true, Message: "Progress synced to AniList"})
}
