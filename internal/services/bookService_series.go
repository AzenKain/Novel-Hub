package services

import (
	"context"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
)

func (s *bookService) GetBookSeriesContext(ctx context.Context, bookID string, claims *response.JWTClaims) (*response.BookSeriesContextResponse, error) {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load book")
	}
	if !s.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "You do not have access to this book")
	}

	entries, err := s.bookRepo.GetBookSeries(ctx, bookID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load series")
	}

	out := &response.BookSeriesContextResponse{Series: models.BookSeriesToResponse(entries)}
	for _, entry := range entries {
		next, err := s.nextReadableInSeries(ctx, entry, bookID, claims)
		if err != nil {
			return nil, err
		}
		if next != nil {
			out.Next = next
			break
		}
	}
	return out, nil
}

func (s *bookService) nextReadableInSeries(ctx context.Context, entry *models.BookSeriesEntity, bookID string, claims *response.JWTClaims) (*response.NextInSeriesResponse, error) {
	if entry == nil {
		return nil, nil
	}
	next, err := s.bookRepo.GetNextBookInSeries(ctx, entry.SeriesID, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load the next book")
	}
	candidate, err := s.bookRepo.GetBook(ctx, next.BookID)
	if err != nil || !s.CanReadBook(ctx, candidate, claims) {
		return nil, nil
	}
	next.SeriesName = entry.SeriesName
	return next.ToResponse(), nil
}
