package services

import (
	"context"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/convert"
)

type MetadataService interface {
	ListAuthors(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error)
	ListSeries(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error)
	ListPublishers(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error)
	ListLanguages(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error)
	ListTags(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error)
	ListFormats(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error)
}

type metadataService struct {
	bookRepo repositories.BookDBRepository
}

func NewMetadataService(bookRepo repositories.BookDBRepository) MetadataService {
	return &metadataService{
		bookRepo: bookRepo,
	}
}

func (s *metadataService) ListAuthors(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error) {
	items, err := s.bookRepo.ListAuthorsWithCount(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}

func (s *metadataService) ListSeries(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error) {
	items, err := s.bookRepo.ListSeriesWithCount(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}

func (s *metadataService) ListPublishers(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error) {
	items, err := s.bookRepo.ListPublishersWithCount(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}

func (s *metadataService) ListLanguages(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error) {
	items, err := s.bookRepo.ListLanguagesWithCount(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}

func (s *metadataService) ListTags(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error) {
	items, err := s.bookRepo.ListTagsWithCount(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}

func (s *metadataService) ListFormats(ctx context.Context, cursor string, limit int64) (*response.PaginatedResponse, error) {
	items, err := s.bookRepo.ListFormatsWithCount(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}
