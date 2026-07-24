package services

import (
	"context"

	"github.com/google/uuid"
	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type HighlightService interface {
	CreateHighlight(ctx context.Context, userID int64, dto *request.CreateHighlightDto, claims *response.JWTClaims) (*response.HighlightResponse, error)
	GetHighlights(ctx context.Context, userID int64, chapterID string, claims *response.JWTClaims) ([]*response.HighlightResponse, error)
	UpdateHighlightNote(ctx context.Context, userID int64, id string, dto *request.UpdateHighlightNoteDto, claims *response.JWTClaims) (*response.HighlightResponse, error)
	DeleteHighlight(ctx context.Context, userID int64, id string, claims *response.JWTClaims) error
}

type highlightService struct {
	repo        repositories.HighlightRepository
	bookRepo    repositories.BookDBRepository
	permissions PermissionCache
}

func NewHighlightService(repo repositories.HighlightRepository, bookRepo repositories.BookDBRepository, permissions PermissionCache) HighlightService {
	return &highlightService{repo: repo, bookRepo: bookRepo, permissions: permissions}
}

func (s *highlightService) canHighlight(ctx context.Context, bookID, chapterID string, claims *response.JWTClaims) bool {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return false
	}
	chapter, err := s.bookRepo.GetChapter(ctx, chapterID)
	if err != nil || chapter == nil || chapter.BookID != bookID {
		return false
	}
	resolved := resolveClaims(claims)
	attrs := map[string]any{"library_id": book.LibraryID}
	return s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermBookRead, attrs) &&
		s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermBookHighlight, attrs)
}

func (s *highlightService) CreateHighlight(ctx context.Context, userID int64, dto *request.CreateHighlightDto, claims *response.JWTClaims) (*response.HighlightResponse, error) {
	if dto.StartIndex >= dto.EndIndex {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid text selection range")
	}
	if !s.canHighlight(ctx, dto.BookID, dto.ChapterID, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Highlights are not allowed for this book")
	}

	id := uuid.Must(uuid.NewV7()).String()
	h, err := s.repo.Create(ctx, sqlc.CreateHighlightParams{
		ID:          id,
		UserID:      userID,
		BookID:      dto.BookID,
		ChapterID:   dto.ChapterID,
		TextContent: dto.TextContent,
		StartIndex:  int64(dto.StartIndex),
		EndIndex:    int64(dto.EndIndex),
		Color:       dto.Color,
		Note:        convert.StrPtrToNullString(dto.Note),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create highlight")
	}

	return h.ToResponse(), nil
}

func (s *highlightService) GetHighlights(ctx context.Context, userID int64, chapterID string, claims *response.JWTClaims) ([]*response.HighlightResponse, error) {
	chapter, err := s.bookRepo.GetChapter(ctx, chapterID)
	if err != nil || chapter == nil || !s.canHighlight(ctx, chapter.BookID, chapterID, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Highlights are not allowed for this chapter")
	}
	highlights, err := s.repo.GetByChapter(ctx, userID, chapterID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get highlights")
	}

	return models.HighlightEntitiesToResponse(highlights), nil
}

func (s *highlightService) UpdateHighlightNote(ctx context.Context, userID int64, id string, dto *request.UpdateHighlightNoteDto, claims *response.JWTClaims) (*response.HighlightResponse, error) {
	items, err := s.repo.GetHighlightsByIDs(ctx, []string{id})
	if err != nil || len(items) != 1 || items[0].UserID != userID || !s.canHighlight(ctx, items[0].BookID, items[0].ChapterID, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Highlight is not accessible")
	}
	h, err := s.repo.UpdateNote(ctx, sqlc.UpdateHighlightNoteParams{
		Note:   convert.StrPtrToNullString(dto.Note),
		Color:  dto.Color,
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update highlight note")
	}

	return h.ToResponse(), nil
}

func (s *highlightService) DeleteHighlight(ctx context.Context, userID int64, id string, claims *response.JWTClaims) error {
	items, err := s.repo.GetHighlightsByIDs(ctx, []string{id})
	if err != nil || len(items) != 1 || items[0].UserID != userID || !s.canHighlight(ctx, items[0].BookID, items[0].ChapterID, claims) {
		return apperrors.New(apperrors.ErrForbidden, "Highlight is not accessible")
	}
	err = s.repo.Delete(ctx, sqlc.DeleteHighlightParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete highlight")
	}
	return nil
}
