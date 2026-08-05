package services

import (
	"context"
	"strings"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type MetadataService interface {
	ListAuthors(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error)
	ListSeries(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error)
	ListPublishers(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error)
	ListLanguages(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error)
	ListTags(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error)
	ListFormats(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error)
}

type metadataService struct {
	bookRepo repositories.BookDBRepository
}

func NewMetadataService(bookRepo repositories.BookDBRepository) MetadataService {
	return &metadataService{
		bookRepo: bookRepo,
	}
}

func (s *metadataService) listFacet(
	ctx context.Context,
	q *request.MetadataFacetDto,
	fetch func(context.Context, repositories.MetadataFacetFilter) ([]*models.MetadataCountEntity, error),
) (*response.PaginatedResponse, error) {
	if q == nil {
		q = &request.MetadataFacetDto{}
	}
	limit := int64(q.Limit)
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 20
	}
	items, err := fetch(ctx, repositories.MetadataFacetFilter{
		Cursor: q.Cursor,
		Limit:  limit,
		Search: strings.TrimSpace(q.Search),
		Alpha:  strings.TrimSpace(q.Alpha),
	})
	if err != nil {
		return nil, err
	}
	var nextCursor string
	if int64(len(items)) == limit && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = convert.EncodeCursor(last.Name, last.ID)
	}
	dtos := models.MetadataCountEntitiesToResponse(items)
	return response.BuildCursorPaginatedResponse(dtos, 0, int(limit), nextCursor), nil
}

func (s *metadataService) ListAuthors(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, s.bookRepo.ListAuthorsWithCount)
}

func (s *metadataService) ListSeries(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, s.bookRepo.ListSeriesWithCount)
}

func (s *metadataService) ListPublishers(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, s.bookRepo.ListPublishersWithCount)
}

func (s *metadataService) ListLanguages(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, s.bookRepo.ListLanguagesWithCount)
}

func (s *metadataService) ListTags(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, s.bookRepo.ListTagsWithCount)
}

func (s *metadataService) ListFormats(ctx context.Context, q *request.MetadataFacetDto) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, s.bookRepo.ListFormatsWithCount)
}
