package services

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

func (s *bookService) BulkDeleteBooks(ctx context.Context, dto *request.BulkDeleteBooksDto, claims *response.JWTClaims) (*response.BulkOperationResponse, error) {
	if dto == nil || len(dto.BookIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No book IDs provided")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	bookByID := make(map[string]*models.BookEntity, len(books))
	for _, book := range books {
		if book != nil {
			bookByID[book.ID] = book
		}
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		found := bookByID[id]

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
		}

		if err := txRepo.BulkDeleteBooks(ctx, allowedIDs); err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}
		txRepo.FlushCache(ctx)
		for _, id := range allowedIDs {
			if err := s.fileRepo.RemoveBookDir(ctx, id); err != nil {
				log.Warn().Err(err).Str("book_id", id).Msg("failed to remove deleted book directory")
			}
		}

		res.SuccessCount = len(allowedIDs)

		allowed := make(map[string]struct{}, len(allowedIDs))
		for _, id := range allowedIDs {
			allowed[id] = struct{}{}
		}
		for _, b := range books {
			if b == nil {
				continue
			}
			_, wasDeleted := allowed[b.ID]
			if wasDeleted && s.webhookService != nil {
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
	if s.libraryRepo == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Library repository not configured")
	}
	if _, err := s.libraryRepo.GetLibrary(ctx, dto.TargetLibraryID); err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Target library not found")
	}
	if claims == nil || !s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermBookUpload, map[string]any{"library_id": dto.TargetLibraryID}) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Target library permission denied")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	bookByID := make(map[string]*models.BookEntity, len(books))
	for _, book := range books {
		if book != nil {
			bookByID[book.ID] = book
		}
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		found := bookByID[id]

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

	bookByID := make(map[string]*models.BookEntity, len(books))
	for _, book := range books {
		if book != nil {
			bookByID[book.ID] = book
		}
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		found := bookByID[id]

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
		if s.featureRepo == nil || claims == nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Collection repository not configured")
		}
		userID, err := convert.ParseID(claims.UId)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid user")
		}
		collections, err := s.featureRepo.GetCollectionsByIDs(ctx, dto.CollectionIDs)
		if err != nil {
			return nil, err
		}
		owned := make(map[string]struct{}, len(collections))
		for _, collection := range collections {
			if collection != nil && collection.UserID == userID {
				owned[collection.ID] = struct{}{}
			}
		}
		if len(owned) != len(dto.CollectionIDs) {
			return nil, apperrors.New(apperrors.ErrForbidden, "Collection permission denied")
		}
		tx, err := s.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		txRepo := s.featureRepo.WithTx(tx)
		for _, bookID := range allowedIDs {
			for _, collectionID := range dto.CollectionIDs {
				if err := txRepo.AddBookToCollection(ctx, collectionID, bookID); err != nil {
					return nil, err
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
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

	bookByID := make(map[string]*models.BookEntity, len(books))
	for _, book := range books {
		if book != nil {
			bookByID[book.ID] = book
		}
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		found := bookByID[id]

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
		txRepo.FlushCache(ctx)

		res.SuccessCount = len(allowedIDs)
	}

	return res, nil
}

func (s *bookService) BulkUpdateMetadata(ctx context.Context, dto *request.BulkUpdateMetadataDto, claims *response.JWTClaims) (*response.BulkOperationResponse, error) {
	if dto == nil || len(dto.BookIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "No book IDs provided")
	}

	books, err := s.bookRepo.GetBooksByIDs(ctx, dto.BookIDs)
	if err != nil {
		return nil, err
	}

	bookByID := make(map[string]*models.BookEntity, len(books))
	for _, book := range books {
		if book != nil {
			bookByID[book.ID] = book
		}
	}

	res := &response.BulkOperationResponse{
		Errors: make(map[string]string),
	}

	allowedIDs := make([]string, 0, len(books))
	for _, id := range dto.BookIDs {
		found := bookByID[id]

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

	if len(allowedIDs) == 0 {
		return res, nil
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	txRepo := s.bookRepo.WithTx(tx)

	// Pre-resolve common Author
	var commonAuthorID *string
	if dto.Author != nil && strings.TrimSpace(*dto.Author) != "" {
		if aID, err := ensureAuthor(ctx, txRepo, *dto.Author); err == nil && aID != "" {
			commonAuthorID = &aID
		}
	}

	// Pre-resolve common Series
	var commonSeriesID string
	if dto.Series != nil && strings.TrimSpace(*dto.Series) != "" {
		seriesName := strings.TrimSpace(*dto.Series)
		sEntity, err := txRepo.GetSeriesByName(ctx, seriesName)
		if err != nil {
			commonSeriesID = uuid.Must(uuid.NewV7()).String()
			_ = txRepo.CreateSeries(ctx, &models.SeriesEntity{ID: commonSeriesID, Name: seriesName})
		} else if sEntity != nil {
			commonSeriesID = sEntity.ID
		}
	}

	// Pre-resolve common Publisher
	var commonPublisherID string
	if dto.Publisher != nil && strings.TrimSpace(*dto.Publisher) != "" {
		pubName := strings.TrimSpace(*dto.Publisher)
		pEntity, err := txRepo.GetPublisherByName(ctx, pubName)
		if err != nil {
			commonPublisherID = uuid.Must(uuid.NewV7()).String()
			_ = txRepo.CreatePublisher(ctx, &models.PublisherEntity{ID: commonPublisherID, Name: pubName})
		} else if pEntity != nil {
			commonPublisherID = pEntity.ID
		}
	}

	// Pre-resolve common Language
	var commonLanguageID string
	if dto.Language != nil && strings.TrimSpace(*dto.Language) != "" {
		langName := strings.TrimSpace(*dto.Language)
		lEntity, err := txRepo.GetLanguageByName(ctx, langName)
		if err != nil {
			commonLanguageID = uuid.Must(uuid.NewV7()).String()
			_ = txRepo.CreateLanguage(ctx, &models.LanguageEntity{ID: commonLanguageID, Name: langName})
		} else if lEntity != nil {
			commonLanguageID = lEntity.ID
		}
	}

	// Pre-resolve AddTags
	addTagIDs := make([]string, 0, len(dto.AddTags))
	for _, tName := range dto.AddTags {
		tName = strings.TrimSpace(tName)
		if tName == "" {
			continue
		}
		if tID, err := ensureTagHelper(ctx, txRepo, tName); err == nil && tID != "" {
			addTagIDs = append(addTagIDs, tID)
		}
	}

	// Pre-resolve RemoveTags
	removeTagIDs := make([]string, 0, len(dto.RemoveTags))
	for _, tName := range dto.RemoveTags {
		tName = strings.TrimSpace(tName)
		if tName == "" {
			continue
		}
		if t, err := txRepo.GetTagByName(ctx, tName); err == nil && t != nil && t.ID != "" {
			removeTagIDs = append(removeTagIDs, t.ID)
		}
	}

	// Build map for item overrides
	itemMap := make(map[string]request.BulkUpdateMetadataItemDto, len(dto.Items))
	for _, it := range dto.Items {
		itemMap[it.BookID] = it
	}

	for idx, bookID := range allowedIDs {
		book := bookByID[bookID]
		if book == nil {
			continue
		}

		item, hasItem := itemMap[bookID]
		bookChanged := false

		// Title update
		if hasItem && item.Title != nil && strings.TrimSpace(*item.Title) != "" {
			book.Title = strings.TrimSpace(*item.Title)
			bookChanged = true
		}

		// Description update
		if hasItem && item.Description != nil {
			d := strings.TrimSpace(*item.Description)
			if d != "" {
				book.Description = &d
			} else {
				book.Description = nil
			}
			bookChanged = true
		}

		// Author update: per-item takes precedence over common
		if hasItem && item.Author != nil && strings.TrimSpace(*item.Author) != "" {
			if aID, err := ensureAuthor(ctx, txRepo, *item.Author); err == nil && aID != "" {
				book.AuthorID = &aID
				bookChanged = true
			}
		} else if commonAuthorID != nil {
			book.AuthorID = commonAuthorID
			bookChanged = true
		}

		if bookChanged {
			if err := txRepo.UpdateBook(ctx, book); err != nil {
				res.FailedCount++
				res.Errors[bookID] = err.Error()
				continue
			}
		}

		// Series linkage
		if commonSeriesID != "" {
			var seriesIndex *string
			if hasItem && item.SeriesIndex != nil && strings.TrimSpace(*item.SeriesIndex) != "" {
				sIdx := strings.TrimSpace(*item.SeriesIndex)
				seriesIndex = &sIdx
			} else if dto.AutoIndexSeries {
				sIdx := strconv.Itoa(idx + 1)
				seriesIndex = &sIdx
			}
			_ = txRepo.LinkBookSeries(ctx, bookID, commonSeriesID, seriesIndex)
		}

		// Publisher linkage
		if commonPublisherID != "" {
			_ = txRepo.ClearBookPublishers(ctx, bookID)
			_ = txRepo.LinkBookPublisher(ctx, bookID, commonPublisherID)
		}

		// Language linkage
		if commonLanguageID != "" {
			_ = txRepo.ClearBookLanguages(ctx, bookID)
			_ = txRepo.LinkBookLanguage(ctx, bookID, commonLanguageID)
		}

		// Add tags
		for _, tagID := range addTagIDs {
			_ = txRepo.AddBookTag(ctx, bookID, tagID)
		}

		// Remove tags
		for _, tagID := range removeTagIDs {
			_ = txRepo.RemoveBookTag(ctx, bookID, tagID)
		}

		res.SuccessCount++
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	txRepo.FlushCache(ctx)

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
