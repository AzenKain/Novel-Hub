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
	ListAuthors(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
	ListSeries(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
	ListPublishers(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
	ListLanguages(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
	ListTags(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
	ListFormats(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
	ListRatings(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error)
}

type metadataService struct {
	bookRepo  repositories.BookDBRepository
	libraries LibraryService
}

func NewMetadataService(bookRepo repositories.BookDBRepository, libraries LibraryService) MetadataService {
	return &metadataService{
		bookRepo:  bookRepo,
		libraries: libraries,
	}
}

func (s *metadataService) listFacet(
	ctx context.Context,
	q *request.MetadataFacetDto,
	claims *response.JWTClaims,
	fetch func(context.Context, repositories.MetadataFacetFilter) ([]*models.MetadataCountEntity, error),
) (*response.PaginatedResponse, error) {
	if q == nil {
		q = &request.MetadataFacetDto{}
	}
	limit := int64(q.Limit)
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 20
	}
	var libraryIDs []string
	if s.libraries != nil {
		var err error
		libraryIDs, err = s.libraries.ReadableLibraryIDs(ctx, claims)
		if err != nil {
			return nil, err
		}
		if len(libraryIDs) == 0 {
			return response.BuildCursorPaginatedResponse([]*response.MetadataCountResponse{}, 0, int(limit), ""), nil
		}
	}
	items, err := fetch(ctx, repositories.MetadataFacetFilter{
		Cursor:     q.Cursor,
		Limit:      limit,
		Search:     strings.TrimSpace(q.Search),
		Alpha:      strings.TrimSpace(q.Alpha),
		LibraryIDs: libraryIDs,
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

func (s *metadataService) ListAuthors(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListAuthorsWithCount)
}

func (s *metadataService) ListSeries(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListSeriesWithCount)
}

func (s *metadataService) ListPublishers(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListPublishersWithCount)
}

func (s *metadataService) ListLanguages(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListLanguagesWithCount)
}

func (s *metadataService) ListTags(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListTagsWithCount)
}

func (s *metadataService) ListFormats(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListFormatsWithCount)
}

func (s *metadataService) ListRatings(ctx context.Context, q *request.MetadataFacetDto, claims *response.JWTClaims) (*response.PaginatedResponse, error) {
	return s.listFacet(ctx, q, claims, s.bookRepo.ListRatingsWithCount)
}
