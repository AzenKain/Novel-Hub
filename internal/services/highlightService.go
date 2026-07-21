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
	"novelhub/pkg/convert"
)

type HighlightService interface {
	CreateHighlight(ctx context.Context, userID int64, dto *request.CreateHighlightDto) (*response.HighlightResponse, error)
	GetHighlights(ctx context.Context, userID int64, chapterID string) ([]*response.HighlightResponse, error)
	UpdateHighlightNote(ctx context.Context, userID int64, id string, dto *request.UpdateHighlightNoteDto) (*response.HighlightResponse, error)
	DeleteHighlight(ctx context.Context, userID int64, id string) error
}

type highlightService struct {
	repo repositories.HighlightRepository
}

func NewHighlightService(repo repositories.HighlightRepository) HighlightService {
	return &highlightService{repo: repo}
}

func (s *highlightService) CreateHighlight(ctx context.Context, userID int64, dto *request.CreateHighlightDto) (*response.HighlightResponse, error) {
	if dto.StartIndex >= dto.EndIndex {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid text selection range")
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

func (s *highlightService) GetHighlights(ctx context.Context, userID int64, chapterID string) ([]*response.HighlightResponse, error) {
	highlights, err := s.repo.GetByChapter(ctx, userID, chapterID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get highlights")
	}

	return models.HighlightEntitiesToResponse(highlights), nil
}

func (s *highlightService) UpdateHighlightNote(ctx context.Context, userID int64, id string, dto *request.UpdateHighlightNoteDto) (*response.HighlightResponse, error) {
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

func (s *highlightService) DeleteHighlight(ctx context.Context, userID int64, id string) error {
	err := s.repo.Delete(ctx, sqlc.DeleteHighlightParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete highlight")
	}
	return nil
}
