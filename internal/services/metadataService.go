package services

import (
	"context"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
)

type MetadataService interface {
	ListAuthors(ctx context.Context) ([]*models.MetadataCountEntity, error)
	ListSeries(ctx context.Context) ([]*models.MetadataCountEntity, error)
	ListPublishers(ctx context.Context) ([]*models.MetadataCountEntity, error)
	ListLanguages(ctx context.Context) ([]*models.MetadataCountEntity, error)
	ListTags(ctx context.Context) ([]*models.MetadataCountEntity, error)
	ListFormats(ctx context.Context) ([]*models.MetadataCountEntity, error)
}

type metadataService struct {
	bookRepo repositories.BookDBRepository
}

func NewMetadataService(bookRepo repositories.BookDBRepository) MetadataService {
	return &metadataService{
		bookRepo: bookRepo,
	}
}

func (s *metadataService) ListAuthors(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	return s.bookRepo.ListAuthorsWithCount(ctx)
}

func (s *metadataService) ListSeries(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	return s.bookRepo.ListSeriesWithCount(ctx)
}

func (s *metadataService) ListPublishers(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	return s.bookRepo.ListPublishersWithCount(ctx)
}

func (s *metadataService) ListLanguages(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	return s.bookRepo.ListLanguagesWithCount(ctx)
}

func (s *metadataService) ListTags(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	return s.bookRepo.ListTagsWithCount(ctx)
}

func (s *metadataService) ListFormats(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	return s.bookRepo.ListFormatsWithCount(ctx)
}
