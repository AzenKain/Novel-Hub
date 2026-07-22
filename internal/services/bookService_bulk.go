package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
)

func (s *bookService) BulkDeleteBooks(ctx context.Context, dto *request.BulkDeleteBooksDto, claims *response.JWTClaims) (*response.BulkOperationResponse, error) {
	if dto == nil || len(dto.BookIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No book IDs provided")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		var found *models.BookEntity
		for _, b := range books {
			if b != nil && b.ID == id {
				found = b
				break
			}
		}

		if found == nil {
			res.FailedCount++
			res.Errors[id] = "Book not found"
			continue
		}

		if !s.CanDeleteBook(ctx, found, claims) {
			res.FailedCount++
			res.Errors[id] = "Permission denied"
			continue
		}

		allowedIDs = append(allowedIDs, id)
	}

	if len(allowedIDs) > 0 {
		tx, err := s.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		txRepo := s.bookRepo.WithTx(tx)

		for _, id := range allowedIDs {
			_ = txRepo.DeleteFTSBook(ctx, id)
			if err := s.fileRepo.RemoveBookDir(ctx, id); err != nil {
				log.Warn().Err(err).Str("book_id", id).Msg("failed to remove book dir during bulk delete")
			}
		}

		if err := txRepo.BulkDeleteBooks(ctx, allowedIDs); err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		res.SuccessCount = len(allowedIDs)

		for _, b := range books {
			if b != nil && containsString(allowedIDs, b.ID) && s.webhookService != nil {
				s.webhookService.DispatchEvent(ctx, "book.deleted", BuildBookWebhookPayload(b))
			}
		}
	}

	return res, nil
}

func (s *bookService) BulkMoveBooks(ctx context.Context, dto *request.BulkMoveBooksDto, claims *response.JWTClaims) (*response.BulkOperationResponse, error) {
	if dto == nil || len(dto.BookIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No book IDs provided")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		var found *models.BookEntity
		for _, b := range books {
			if b != nil && b.ID == id {
				found = b
				break
			}
		}

		if found == nil {
			res.FailedCount++
			res.Errors[id] = "Book not found"
			continue
		}

		if !s.CanUpdateBook(ctx, found, claims) {
			res.FailedCount++
			res.Errors[id] = "Permission denied"
			continue
		}

		allowedIDs = append(allowedIDs, id)
	}

	if len(allowedIDs) > 0 {
		if err := s.bookRepo.BulkUpdateBookLibrary(ctx, allowedIDs, dto.TargetLibraryID); err != nil {
			return nil, err
		}
		res.SuccessCount = len(allowedIDs)
	}

	return res, nil
}

func (s *bookService) BulkAssignCollections(ctx context.Context, dto *request.BulkAssignCollectionsDto, claims *response.JWTClaims) (*response.BulkOperationResponse, error) {
	if dto == nil || len(dto.BookIDs) == 0 || len(dto.CollectionIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No book IDs or collection IDs provided")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		var found *models.BookEntity
		for _, b := range books {
			if b != nil && b.ID == id {
				found = b
				break
			}
		}

		if found == nil {
			res.FailedCount++
			res.Errors[id] = "Book not found"
			continue
		}

		if !s.CanUpdateBook(ctx, found, claims) {
			res.FailedCount++
			res.Errors[id] = "Permission denied"
			continue
		}

		allowedIDs = append(allowedIDs, id)
	}

	if len(allowedIDs) > 0 {
		res.SuccessCount = len(allowedIDs)
	}

	return res, nil
}

func (s *bookService) BulkAddTags(ctx context.Context, dto *request.BulkAddTagsDto, claims *response.JWTClaims) (*response.BulkOperationResponse, error) {
	if dto == nil || len(dto.BookIDs) == 0 || len(dto.TagNames) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No book IDs or tag names provided")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		var found *models.BookEntity
		for _, b := range books {
			if b != nil && b.ID == id {
				found = b
				break
			}
		}

		if found == nil {
			res.FailedCount++
			res.Errors[id] = "Book not found"
			continue
		}

		if !s.CanUpdateBook(ctx, found, claims) {
			res.FailedCount++
			res.Errors[id] = "Permission denied"
			continue
		}

		allowedIDs = append(allowedIDs, id)
	}

	if len(allowedIDs) > 0 {
		tx, err := s.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		txRepo := s.bookRepo.WithTx(tx)

		tagIDs := make([]string, 0, len(dto.TagNames))
		for _, tagName := range dto.TagNames {
			if tagID, err := ensureTagHelper(ctx, txRepo, tagName); err == nil && tagID != "" {
				tagIDs = append(tagIDs, tagID)
			}
		}

		for _, bookID := range allowedIDs {
			for _, tagID := range tagIDs {
				_ = txRepo.AddBookTag(ctx, bookID, tagID)
			}
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		res.SuccessCount = len(allowedIDs)
	}

	return res, nil
}

func ensureTagHelper(ctx context.Context, repo repositories.BookMetadataRepository, name string) (string, error) {
	tag, err := repo.GetTagByName(ctx, name)
	if err == nil && tag != nil && tag.ID != "" {
		return tag.ID, nil
	}
	newID := uuid.Must(uuid.NewV7()).String()
	err = repo.CreateTag(ctx, &models.TagEntity{
		ID:   newID,
		Name: name,
	})
	if err != nil {
		return "", err
	}
	return newID, nil
}

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
