package controllers

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"novelhub/pkg/apperrors"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/services"
	"novelhub/pkg/convert"
	"novelhub/pkg/validator"
)

type FeatureController struct {
	service     services.FeatureService
	books       services.BookService
	settings    services.SettingsService
	permissions services.PermissionCache
}

func NewFeatureController(service services.FeatureService, books services.BookService, settings services.SettingsService, permissions services.PermissionCache) *FeatureController {
	return &FeatureController{service: service, books: books, settings: settings, permissions: permissions}
}

func (c *FeatureController) GetLibraryStats(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.service.GetLibraryStats(reqCtx)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *FeatureController) GetCollections(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	limit := int64(50)
	if l, err := strconv.ParseInt(ctx.Query("limit"), 10, 64); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100
	}
	var cursorCreatedAt *time.Time
	var cursorID string
	if cursorStr := ctx.Query("cursor"); cursorStr != "" {
		if parts := strings.SplitN(cursorStr, "|", 2); len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorCreatedAt = &t
				cursorID = parts[1]
			}
		} else if t, err := time.Parse(time.RFC3339Nano, cursorStr); err == nil {
			cursorCreatedAt = &t
		}
	}

	collections, err := c.service.GetUserCollections(reqCtx, userID, cursorCreatedAt, cursorID, limit)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	var nextCursor *string
	if len(collections) >= int(limit) && len(collections) > 0 {
		last := collections[len(collections)-1]
		next := last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID
		nextCursor = &next
	}
	return ctx.JSON(fiber.Map{
		"status":      true,
		"data":        collections,
		"next_cursor": nextCursor,
	})
}

func (c *FeatureController) CreateCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.CreateCollectionDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, _ := getUserClaims(ctx)
	if !c.service.PolicyAllowsNoBook(reqCtx, "collection", claims) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Collections are not allowed"})
	}

	col, err := c.service.CreateCollection(reqCtx, dto.Name, userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   col,
	})
}

func (c *FeatureController) UpdateCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "missing id"})
	}

	dto := &request.UpdateCollectionDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	col, err := c.service.UpdateCollection(reqCtx, id, dto.Name, userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   col,
	})
}

func (c *FeatureController) DeleteCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "missing id"})
	}

	err := c.service.DeleteCollection(reqCtx, id, userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
	})
}

func (c *FeatureController) GetRecentReadingHistory(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.PaginationDto{Limit: 10}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursorTime *time.Time
	var cursorID string
	if dto.Cursor != "" {
		if parts := strings.SplitN(dto.Cursor, "|", 2); len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
				cursorID = parts[1]
			}
		} else if t, err := time.Parse(time.RFC3339Nano, dto.Cursor); err == nil {
			cursorTime = &t
		}
	}

	history, err := c.service.GetRecentReadingHistory(reqCtx, userID, cursorTime, cursorID, int64(dto.Limit))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	var nextCursor *string
	if len(history) >= int(dto.Limit) && len(history) > 0 {
		last := history[len(history)-1]
		c := last.UpdatedAt.Format(time.RFC3339Nano) + "|" + last.BookID
		nextCursor = &c
	}
	return ctx.JSON(fiber.Map{
		"status":      true,
		"data":        history,
		"next_cursor": nextCursor,
	})
}

func (c *FeatureController) RecordReadingActivity(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.RecordReadingActivityDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var fileID *string
	if value := strings.TrimSpace(dto.FileID); value != "" {
		fileID = &value
	}

	result, err := c.service.RecordReadingActivity(reqCtx, models.ReadingActivityInput{
		UserID:          userID,
		BookID:          dto.BookID,
		FileID:          fileID,
		ChapterID:       dto.ChapterID,
		ChapterTitle:    dto.ChapterTitle,
		ChapterIndex:    dto.ChapterIndex,
		ProgressPercent: dto.ProgressPercent,
		LocationCfi:     dto.LocationCfi,
		LocationType:    dto.LocationType,
		EventType:       dto.EventType,
	}, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (h *FeatureController) GetReadingProgress(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookID := c.Params("id")
	claims, ok := c.Locals("user_claims").(*response.JWTClaims)
	if !ok || claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Unauthorized"})
	}
	userID, err := convert.ParseID(claims.UId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "Invalid user ID"})
	}

	progress, err := h.service.GetReadingProgress(ctx, userID, bookID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(response.CommonResponse{Status: false, Message: "No reading progress found"})
		}
		return apperrors.HandleError(c, err)
	}
	return c.JSON(response.CommonResponse{Status: true, Data: progress})
}

func (c *FeatureController) GetBookReadStats(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.service.GetBookReadStats(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *FeatureController) GetBookDownloadStats(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.service.GetBookDownloadStats(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *FeatureController) GetBookEngagementStats(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.service.GetBookEngagementStats(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *FeatureController) RecordBookShare(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !c.service.PolicyAllowsBook(reqCtx, "share", ctx.Params("id"), getOptionalClaims(ctx)) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Sharing is disabled"})
	}

	dto := &request.RecordShareDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	stats, err := c.service.RecordShare(reqCtx, models.ShareInput{
		BookID:     ctx.Params("id"),
		ActorKey:   c.service.ShareActorKey(dto.ClientID, ctx.IP(), ctx.Get("User-Agent")),
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *FeatureController) SetBookmark(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.SetBookmarkDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, _ := getUserClaims(ctx)
	if !c.service.PolicyAllowsBook(reqCtx, "bookmark", ctx.Params("id"), claims) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Bookmarking is not allowed"})
	}

	bookmark, err := c.service.SetBookmark(reqCtx, userID, ctx.Params("id"), dto.Bookmarked)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   bookmark,
	})
}

func (c *FeatureController) GetBookmarkedBooks(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.PaginationDto{Limit: 20}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursorTime *time.Time
	var cursorID string
	if dto.Cursor != "" {
		if parts := strings.SplitN(dto.Cursor, "|", 2); len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
				cursorID = parts[1]
			}
		} else if t, err := time.Parse(time.RFC3339Nano, dto.Cursor); err == nil {
			cursorTime = &t
		}
	}
	page, err := c.service.GetBookmarkedBooks(reqCtx, userID, cursorTime, cursorID, int64(dto.Limit))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}
	return ctx.JSON(fiber.Map{
		"status":      true,
		"data":        models.BookEntitiesToResponse(page.Books),
		"next_cursor": nextCursor,
	})
}

func (c *FeatureController) GetBookUserState(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	claims, _ := getUserClaims(ctx)
	state, err := c.service.GetBookUserState(reqCtx, userID, ctx.Params("id"), claims)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   state,
	})
}

func (c *FeatureController) UpsertBookReview(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.UpsertBookReviewDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, _ := getUserClaims(ctx)
	if !c.service.PolicyAllowsBook(reqCtx, "review", ctx.Params("id"), claims) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Reviews are not allowed"})
	}

	review, err := c.service.UpsertBookReview(reqCtx, userID, ctx.Params("id"), dto.Rating, dto.Review)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   review,
	})
}

func (c *FeatureController) DeleteBookReview(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}
	claims, _ := getUserClaims(ctx)
	if !c.service.PolicyAllowsBook(reqCtx, "review", ctx.Params("id"), claims) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Reviews are not allowed"})
	}

	if err := c.service.DeleteBookReview(reqCtx, userID, ctx.Params("id")); err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status:  true,
		Message: "review deleted",
	})
}

func (c *FeatureController) AdminDeleteReview(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	targetUserID, err := convert.ParseID(ctx.Params("userId"))
	if err != nil {
		return apperrors.HandleError(ctx, apperrors.New(apperrors.ErrBadRequest, "invalid user ID"))
	}

	bookID := ctx.Params("bookId")
	if err := c.service.DeleteReviewByAdmin(reqCtx, targetUserID, bookID); err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status:  true,
		Message: "review deleted",
	})
}

func (c *FeatureController) ListAllReviews(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.PaginationDto{Limit: 50}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	reviews, err := c.service.ListAllReviews(reqCtx, int64(dto.Limit), int64(dto.Offset))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   reviews,
	})
}

func (c *FeatureController) ListBookReviews(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.PaginationDto{Limit: 20}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	var cursorTime *time.Time
	var cursorID string
	if dto.Cursor != "" {
		if parts := strings.SplitN(dto.Cursor, "|", 2); len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
				cursorID = parts[1]
			}
		} else if t, err := time.Parse(time.RFC3339Nano, dto.Cursor); err == nil {
			cursorTime = &t
		}
	}

	// Reviews name their author and quote free text, so they are scoped to the book they hang
	// off. This route used to carry no check at all: anyone who guessed a book id read the
	// reviewers of a library they had no access to.
	if !c.service.PolicyAllowsBook(reqCtx, "read", ctx.Params("id"), getOptionalClaims(ctx)) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	reviews, err := c.service.ListBookReviews(reqCtx, ctx.Params("id"), cursorTime, cursorID, int64(dto.Limit))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	var nextCursor *string
	if len(reviews) >= int(dto.Limit) && len(reviews) > 0 {
		last := reviews[len(reviews)-1]
		if last.UpdatedAt != nil {
			c := last.UpdatedAt.Format(time.RFC3339Nano) + "|" + last.UserID
			nextCursor = &c
		}
	}
	return ctx.JSON(fiber.Map{
		"status":      true,
		"data":        reviews,
		"next_cursor": nextCursor,
	})
}

func (c *FeatureController) GetBookRatingSummary(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !c.service.PolicyAllowsBook(reqCtx, "read", ctx.Params("id"), getOptionalClaims(ctx)) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "You do not have access to this book"})
	}

	summary, err := c.service.GetBookRatingSummary(reqCtx, ctx.Params("id"))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   summary,
	})
}

func (c *FeatureController) AddBookToCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.CollectionBookDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}
	claims, _ := getUserClaims(ctx)
	if !c.service.PolicyAllowsBook(reqCtx, "collection", dto.BookID, claims) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Collections are not allowed for this book"})
	}

	collectionID := ctx.Params("id")
	err := c.service.AddBookToCollection(reqCtx, userID, collectionID, dto.BookID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
	})
}

func (c *FeatureController) RemoveBookFromCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	collectionID := ctx.Params("id")
	bookID := ctx.Params("bookId")
	claims, _ := getUserClaims(ctx)
	if !c.service.PolicyAllowsBook(reqCtx, "collection", bookID, claims) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Collections are not allowed for this book"})
	}
	err := c.service.RemoveBookFromCollection(reqCtx, userID, collectionID, bookID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
	})
}

func (c *FeatureController) RecordReadingSession(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.RecordReadingSessionDto{}
	if errs := validator.ValidateBodyDto(ctx, dto); errs != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: errs})
	}

	err := c.service.RecordReadingSession(reqCtx, userID, dto.BookID, dto.Duration, dto.Words, dto.SessionDate, getOptionalClaims(ctx))
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{Status: true})
}

func (c *FeatureController) GetReadingHeatmap(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	res, err := c.service.GetReadingHeatmap(reqCtx, userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}

	return ctx.JSON(response.CommonResponse{Status: true, Data: res})
}

func (c *FeatureController) GetReadingGoal(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	goal, err := c.service.GetReadingGoal(reqCtx, userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: goal})
}

func (c *FeatureController) UpsertReadingGoal(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.UpsertReadingGoalDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	goal, err := c.service.UpsertReadingGoal(reqCtx, userID, dto.TargetWordsPerDay, dto.TargetBooksPerYear)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: goal})
}

func (c *FeatureController) ListSmartCollections(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	items, err := c.service.ListSmartCollections(reqCtx, userID)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: items})
}

func (c *FeatureController) CreateSmartCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	dto := &request.UpsertSmartCollectionDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	item, err := c.service.CreateSmartCollection(reqCtx, userID, *dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: item})
}

func (c *FeatureController) UpdateSmartCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "id is required"})
	}

	dto := &request.UpsertSmartCollectionDto{}
	if err := validator.ValidateBodyDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	item, err := c.service.UpdateSmartCollection(reqCtx, id, userID, *dto)
	if err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true, Data: item})
}

func (c *FeatureController) DeleteSmartCollection(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, ok := getUserIdFromLocals(ctx)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
	}

	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Message: "id is required"})
	}

	if err := c.service.DeleteSmartCollection(reqCtx, id, userID); err != nil {
		return apperrors.HandleError(ctx, err)
	}
	return ctx.JSON(response.CommonResponse{Status: true})
}
