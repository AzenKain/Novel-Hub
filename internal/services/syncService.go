package services

import (
	"context"
	"fmt"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
)

type SyncService interface {
	GetProgress(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (*response.ReadingProgressResponse, error)
	UpdateProgress(ctx context.Context, userID string, dto *request.ProgressSyncDto, claims *response.JWTClaims) (*response.ReadingProgressResponse, error)
	KosyncGetProgress(ctx context.Context, userID string, document string, claims *response.JWTClaims) (map[string]any, error)
	KosyncPushProgress(ctx context.Context, userID string, dto *request.KosyncPushProgressDto, claims *response.JWTClaims) (map[string]any, error)
}

type syncService struct {
	featureService FeatureService
	bookService    BookService
	permissions    PermissionCache
}

func NewSyncService(featureService FeatureService, bookService BookService, permissions PermissionCache) SyncService {
	return &syncService{
		featureService: featureService,
		bookService:    bookService,
		permissions:    permissions,
	}
}

func (s *syncService) checkAccess(ctx context.Context, bookID string, claims *response.JWTClaims) error {
	resolved := resolveClaims(claims)
	book, err := s.bookService.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	if !s.bookService.CanReadBook(ctx, book, resolved) {
		return apperrors.New(apperrors.ErrForbidden, "Book is not accessible")
	}
	if !s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermTrackerSync, map[string]any{"library_id": book.LibraryID}) {
		return apperrors.New(apperrors.ErrForbidden, "Permission denied to sync reading progress")
	}
	return nil
}

func (s *syncService) GetProgress(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (*response.ReadingProgressResponse, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}
	if err := s.checkAccess(ctx, bookID, claims); err != nil {
		return nil, err
	}
	return s.featureService.GetReadingProgress(ctx, userID, bookID)
}

func (s *syncService) UpdateProgress(ctx context.Context, userID string, dto *request.ProgressSyncDto, claims *response.JWTClaims) (*response.ReadingProgressResponse, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}
	if err := s.checkAccess(ctx, dto.BookID, claims); err != nil {
		return nil, err
	}
	chapterIndex := int64(0)
	if dto.ChapterIndex != nil {
		chapterIndex = *dto.ChapterIndex
	}
	chapterTitle := ""
	if dto.ChapterTitle != nil {
		chapterTitle = *dto.ChapterTitle
	}

	input := models.ReadingActivityInput{
		UserID:          userID,
		BookID:          dto.BookID,
		ChapterID:       dto.BookID,
		ChapterTitle:    chapterTitle,
		ChapterIndex:    chapterIndex,
		ProgressPercent: dto.ProgressPercent,
		LocationCfi:     dto.LocationCfi,
		LocationType:    dto.LocationType,
		EventType:       "progress_sync",
	}

	res, err := s.featureService.RecordReadingActivity(ctx, input, claims)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Progress == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to record reading progress")
	}
	return res.Progress, nil
}

func (s *syncService) KosyncGetProgress(ctx context.Context, userID string, document string, claims *response.JWTClaims) (map[string]any, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}
	if err := s.checkAccess(ctx, document, claims); err != nil {
		return nil, err
	}
	progress, err := s.featureService.GetReadingProgress(ctx, userID, document)
	if err != nil || progress == nil {
		return map[string]any{
			"document":   document,
			"progress":   0,
			"percentage": 0,
			"timestamp":  time.Now().Unix(),
			"device":     "NovelHub Server",
		}, nil
	}
	percent := float64(0)
	if progress.ProgressPercent != nil {
		percent = *progress.ProgressPercent
	}
	return map[string]any{
		"document":   document,
		"progress":   percent / 100.0,
		"percentage": percent,
		"timestamp":  progress.UpdatedAt.Unix(),
		"device":     "NovelHub Server",
	}, nil
}

func (s *syncService) KosyncPushProgress(ctx context.Context, userID string, dto *request.KosyncPushProgressDto, claims *response.JWTClaims) (map[string]any, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}
	if err := s.checkAccess(ctx, dto.Document, claims); err != nil {
		return nil, err
	}
	percent := dto.Percentage
	if percent <= 0 && dto.Progress > 0 {
		percent = dto.Progress * 100.0
	}
	cfi := fmt.Sprintf("%.4f", dto.Progress)
	locType := "koreader"

	input := models.ReadingActivityInput{
		UserID:          userID,
		BookID:          dto.Document,
		ChapterID:       dto.Document,
		ChapterTitle:    "KOReader Sync",
		ProgressPercent: &percent,
		LocationCfi:     &cfi,
		LocationType:    &locType,
		EventType:       "koreader_sync",
	}

	_, err := s.featureService.RecordReadingActivity(ctx, input, claims)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"document":  dto.Document,
		"timestamp": time.Now().Unix(),
		"status":    "ok",
	}, nil
}
