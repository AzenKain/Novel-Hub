package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/services"
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
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch library stats",
			
		})
	}
	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func getUserIdFromLocals(ctx fiber.Ctx) (int64, bool) {
	uidRaw := ctx.Locals("uid")
	if uidRaw == nil {
		return 0, false
	}
	uidStr, ok := uidRaw.(string)
	if !ok {
		return 0, false
	}
	userID, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return userID, true
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

	collections, err := c.service.GetUserCollections(reqCtx, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch collections",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   collections,
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
	if !c.policyAllowsNoBook(ctx, "collection") {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Collections are not allowed"})
	}

	col, err := c.service.CreateCollection(reqCtx, dto.Name, userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to create collection",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   col,
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

	dto := &request.LimitDto{Limit: 10}
	if err := validator.ValidateQueryDto(ctx, dto); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{Status: false, Errors: err})
	}

	history, err := c.service.GetRecentReadingHistory(reqCtx, userID, int64(dto.Limit))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch reading history",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   history,
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
		EventType:       dto.EventType,
	})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to record reading activity",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   result,
	})
}

func (c *FeatureController) GetBookReadStats(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.service.GetBookReadStats(reqCtx, ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch read stats",
			
		})
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
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch download stats",
			
		})
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
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch engagement stats",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *FeatureController) RecordBookShare(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.RecordShareDto{}
	_ = ctx.Bind().Body(dto)
	stats, err := c.service.RecordShare(reqCtx, models.ShareInput{
		BookID:     ctx.Params("id"),
		ActorKey:   shareActorKey(dto.ClientID, ctx.IP(), ctx.Get("User-Agent")),
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to record share",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func shareActorKey(clientID string, ip string, userAgent string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = "anonymous"
	}
	sum := sha256.Sum256([]byte(clientID + "|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
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
	if !c.policyAllowsBook(reqCtx, ctx, "bookmark", ctx.Params("id")) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Bookmarking is not allowed"})
	}

	bookmark, err := c.service.SetBookmark(reqCtx, userID, ctx.Params("id"), dto.Bookmarked)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to update bookmark",
			
		})
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

	books, err := c.service.GetBookmarkedBooks(reqCtx, userID, int64(dto.Limit), int64(dto.Offset))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch bookmarked books",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   books,
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

	state, err := c.service.GetBookUserState(reqCtx, userID, ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch book state",
			
		})
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
	if !c.policyAllowsBook(reqCtx, ctx, "review", ctx.Params("id")) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Reviews are not allowed"})
	}

	review, err := c.service.UpsertBookReview(reqCtx, userID, ctx.Params("id"), dto.Rating, dto.Review)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to save review",
			
		})
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
	if !c.policyAllowsBook(reqCtx, ctx, "review", ctx.Params("id")) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Reviews are not allowed"})
	}

	if err := c.service.DeleteBookReview(reqCtx, userID, ctx.Params("id")); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to delete review",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status:  true,
		Message: "review deleted",
	})
}

func (c *FeatureController) AdminDeleteReview(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	targetUserID, err := strconv.ParseInt(ctx.Params("userId"), 10, 64)
	if err != nil || targetUserID < 1 {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "invalid user ID",
		})
	}

	bookID := ctx.Params("bookId")
	if err := c.service.DeleteReviewByAdmin(reqCtx, targetUserID, bookID); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to delete review",
			
		})
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
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch reviews",
			
		})
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

	reviews, err := c.service.ListBookReviews(reqCtx, ctx.Params("id"), int64(dto.Limit), int64(dto.Offset))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch reviews",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
		Data:   reviews,
	})
}

func (c *FeatureController) GetBookRatingSummary(ctx fiber.Ctx) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	summary, err := c.service.GetBookRatingSummary(reqCtx, ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to fetch rating summary",
			
		})
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
	if !c.policyAllowsBook(reqCtx, ctx, "collection", dto.BookID) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Collections are not allowed for this book"})
	}

	collectionID := ctx.Params("id")
	err := c.service.AddBookToCollection(reqCtx, userID, collectionID, dto.BookID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to add book to collection",
			
		})
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
	if !c.policyAllowsBook(reqCtx, ctx, "collection", bookID) {
		return ctx.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "Collections are not allowed for this book"})
	}
	err := c.service.RemoveBookFromCollection(reqCtx, userID, collectionID, bookID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: "failed to remove book from collection",
			
		})
	}

	return ctx.JSON(response.CommonResponse{
		Status: true,
	})
}

func (c *FeatureController) policyAllowsBook(reqCtx context.Context, ctx fiber.Ctx, policy string, bookID string) bool {
	book, err := c.books.GetBook(reqCtx, bookID)
	if err != nil || book == nil {
		return false
	}
	claims, _ := ctx.Locals("user_claims").(*response.JWTClaims)
	admin := claims != nil && c.permissions.IsAdmin(claims.RoleIDs, claims.Roles)
	return c.settings.PolicyAllows(policy, book.LibraryID, admin)
}

func (c *FeatureController) policyAllowsNoBook(ctx fiber.Ctx, policy string) bool {
	claims, _ := ctx.Locals("user_claims").(*response.JWTClaims)
	admin := claims != nil && c.permissions.IsAdmin(claims.RoleIDs, claims.Roles)
	if admin {
		return true
	}
	settings, err := c.settings.Public(ctx.Context())
	if err != nil {
		return false
	}
	switch policy {
	case "collection":
		return settings.Collection.Mode != "disabled"
	default:
		return false
	}
}
