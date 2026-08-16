package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"

	"github.com/rs/zerolog/log"
)

type FeatureService interface {
	GetLibraryStats(ctx context.Context, claims *response.JWTClaims) (*response.LibraryStatsResponse, error)
	CreateCollection(ctx context.Context, name string, userID string) (*response.CollectionResponse, error)
	UpdateCollection(ctx context.Context, id, name string, userID string) (*response.CollectionResponse, error)
	DeleteCollection(ctx context.Context, id string, userID string) error
	GetUserCollections(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*response.CollectionResponse, error)
	GetRecentReadingHistory(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*response.ReadingHistoryResponse, error)
	GetReadingProgress(ctx context.Context, userID string, bookID string) (*response.ReadingProgressResponse, error)
	RecordReadingActivity(ctx context.Context, input models.ReadingActivityInput, claims *response.JWTClaims) (*response.ReadingActivityResponse, error)
	GetBookReadStats(ctx context.Context, bookID string) (*response.BookReadStatsResponse, error)
	RecordDownload(ctx context.Context, bookID string) error
	GetBookDownloadStats(ctx context.Context, bookID string) (*response.BookDownloadStatsResponse, error)
	GetBookSocialStats(ctx context.Context, bookID string) (*response.BookSocialStatsResponse, error)
	GetBookEngagementStats(ctx context.Context, bookID string) (*response.BookEngagementStatsResponse, error)
	RecordShare(ctx context.Context, input models.ShareInput) (*response.BookSocialStatsResponse, error)
	SetBookmark(ctx context.Context, userID string, bookID string, bookmarked bool) (*response.BookmarkResponse, error)
	GetBookmarkedBooks(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) (*response.BookmarkedBooksPageResponse, error)
	GetBookUserState(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (*response.BookUserStateResponse, error)
	UpsertBookReview(ctx context.Context, userID string, bookID string, rating int64, review string) (*response.BookReviewResponse, error)
	DeleteBookReview(ctx context.Context, userID string, bookID string) error
	DeleteReviewByAdmin(ctx context.Context, targetUserID string, bookID string) error
	ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, cursorID string, limit int64) ([]*response.BookReviewResponse, error)
	ListAllReviews(ctx context.Context, limit, offset int64) ([]*response.BookReviewResponse, error)
	GetBookRatingSummary(ctx context.Context, bookID string) (*response.BookRatingSummaryResponse, error)
	AddBookToCollection(ctx context.Context, userID string, collectionID string, bookID string) error
	RemoveBookFromCollection(ctx context.Context, userID string, collectionID string, bookID string) error

	PolicyAllowsBook(ctx context.Context, policy string, bookID string, claims *response.JWTClaims) bool
	PolicyAllowsNoBook(ctx context.Context, policy string, claims *response.JWTClaims) bool
	ShareActorKey(clientID string, ip string, userAgent string) string
	RecordReadingSession(ctx context.Context, userID string, bookID string, duration int64, words int64, sessionDate string, claims *response.JWTClaims) error
	GetReadingHeatmap(ctx context.Context, userID string) (map[string]map[string]int64, error)
	GetReadingStatsSummary(ctx context.Context, userID string) (*response.ReadingStatsSummaryResponse, error)
	GetReaderETA(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (*response.ReadingETAResponse, error)
	GetLibraryBreakdown(ctx context.Context, userID string) (*response.LibraryBreakdownResponse, error)
	GetReadingGoal(ctx context.Context, userID string) (*response.ReadingGoalResponse, error)
	UpsertReadingGoal(ctx context.Context, userID string, wordsPerDay int64, booksPerYear int64) (*response.ReadingGoalResponse, error)
	ListSmartCollections(ctx context.Context, userID string) ([]*response.SmartCollectionResponse, error)
	CreateSmartCollection(ctx context.Context, userID string, dto request.UpsertSmartCollectionDto) (*response.SmartCollectionResponse, error)
	UpdateSmartCollection(ctx context.Context, id string, userID string, dto request.UpsertSmartCollectionDto) (*response.SmartCollectionResponse, error)
	DeleteSmartCollection(ctx context.Context, id string, userID string) error
	ListSmartFilters(ctx context.Context, userID string) ([]*response.SmartFilterResponse, error)
	GetSmartFilter(ctx context.Context, id string, userID string) (*response.SmartFilterResponse, error)
	CreateSmartFilter(ctx context.Context, userID string, dto request.UpsertSmartFilterDto) (*response.SmartFilterResponse, error)
	UpdateSmartFilter(ctx context.Context, id string, userID string, dto request.UpsertSmartFilterDto) (*response.SmartFilterResponse, error)
	DeleteSmartFilter(ctx context.Context, id string, userID string) error
	UpdateSmartFilterPinSidebar(ctx context.Context, id string, userID string, isPinned bool) (*response.SmartFilterResponse, error)
	UpdateSmartFilterPinHome(ctx context.Context, id string, userID string, isPinned bool) (*response.SmartFilterResponse, error)
	ReorderSmartFiltersHome(ctx context.Context, userID string, dto request.ReorderHomeShelvesDto) error
	SetWebhookService(webhook WebhookService)
}

type featureService struct {
	repo           repositories.FeatureRepository
	bookRepo       repositories.BookDBRepository
	settings       SettingsService
	permissions    PermissionCache
	txManager      database.TxManager
	webhookService WebhookService
}

const (
	readCountCooldown = 12 * time.Hour
	shareCountWindow  = time.Hour
	// Must match the "read"/"reading" filters in db/query/books.sql.
	bookCompletedPercent = 99.5
)

func NewFeatureService(repo repositories.FeatureRepository, bookRepo repositories.BookDBRepository, settings SettingsService, permissions PermissionCache, txManager database.TxManager) FeatureService {
	return &featureService{repo: repo, bookRepo: bookRepo, settings: settings, permissions: permissions, txManager: txManager}
}

func (s *featureService) SetWebhookService(webhook WebhookService) {
	s.webhookService = webhook
}

func (s *featureService) GetLibraryStats(ctx context.Context, claims *response.JWTClaims) (*response.LibraryStatsResponse, error) {
	if isGuestClaims(claims) {
		settings, err := s.settings.Public(ctx)
		if err == nil && settings.GuestLoginRequired {
			return nil, apperrors.New(apperrors.ErrUnauthorized, "Guest access is disabled")
		}
	}
	stats, err := s.repo.GetLibraryStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) CreateCollection(ctx context.Context, name string, userID string) (*response.CollectionResponse, error) {
	id := uuid.Must(uuid.NewV7()).String()
	col, err := s.repo.CreateCollection(ctx, id, name, userID)
	if err != nil {
		return nil, err
	}
	return col.ToResponse(), nil
}

func (s *featureService) UpdateCollection(ctx context.Context, id, name string, userID string) (*response.CollectionResponse, error) {
	col, err := s.repo.UpdateCollection(ctx, id, name, userID)
	if err != nil {
		return nil, err
	}
	return col.ToResponse(), nil
}

func (s *featureService) DeleteCollection(ctx context.Context, id string, userID string) error {
	return s.repo.DeleteCollection(ctx, id, userID)
}

func (s *featureService) GetUserCollections(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*response.CollectionResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	cols, err := s.repo.GetUserCollections(ctx, userID, cursorCreatedAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return models.CollectionEntitiesToResponse(cols), nil
}

func (s *featureService) GetRecentReadingHistory(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*response.ReadingHistoryResponse, error) {
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 20
	}
	history, err := s.repo.GetRecentReadingHistory(ctx, userID, cursor, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return models.ReadingHistoryEntitiesToResponse(history), nil
}

func (s *featureService) GetReadingProgress(ctx context.Context, userID string, bookID string) (*response.ReadingProgressResponse, error) {
	entity, err := s.repo.GetReadingProgress(ctx, userID, bookID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "No reading progress found")
		}
		return nil, err
	}
	return entity.ToResponse(), nil
}

// Announces reading.completed only on the crossing into finished, so re-opening does not re-fire.
func (s *featureService) RecordReadingActivity(ctx context.Context, input models.ReadingActivityInput, claims *response.JWTClaims) (*response.ReadingActivityResponse, error) {
	book, err := s.bookRepo.GetBook(ctx, input.BookID)
	resolved := resolveClaims(claims)
	if err != nil || book == nil || !s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermBookRead, map[string]any{"library_id": book.LibraryID}) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Book is not accessible")
	}
	if strings.TrimSpace(input.BookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	if strings.TrimSpace(input.ChapterID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "chapterId is required")
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.repo.WithTx(tx)

	now := time.Now().UTC()
	existing, err := txRepo.GetReadingProgress(ctx, input.UserID, input.BookID)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, err
	}

	var lastCountedAt *time.Time
	if existing != nil {
		lastCountedAt = existing.LastCountedAt
	}

	counted := shouldCountQualifiedRead(existing, now)
	qualifiedDelta := int64(0)
	if counted {
		qualifiedDelta = 1
		lastCountedAt = &now
	}

	progressPercent := normalizeProgress(input.ProgressPercent)
	justFinished := progressPercent != nil && *progressPercent >= bookCompletedPercent &&
		!(existing != nil && existing.ProgressPercent != nil && *existing.ProgressPercent >= bookCompletedPercent)
	chapterTitle := strings.TrimSpace(input.ChapterTitle)
	if chapterTitle == "" {
		chapterTitle = fmt.Sprintf("Chapter %d", input.ChapterIndex+1)
	}

	progress := &models.ReadingProgressEntity{
		UserID:             input.UserID,
		BookID:             input.BookID,
		FileID:             cleanStringPtr(input.FileID),
		ChapterID:          input.ChapterID,
		ChapterTitle:       chapterTitle,
		ChapterIndex:       input.ChapterIndex,
		ProgressPercent:    progressPercent,
		LocationCfi:        input.LocationCfi,
		LocationType:       input.LocationType,
		OpenedCount:        1,
		QualifiedReadCount: qualifiedDelta,
		LastCountedAt:      lastCountedAt,
	}

	saved, err := txRepo.UpsertReadingProgress(ctx, progress)
	if err != nil {
		return nil, err
	}
	if err := txRepo.UpsertBookReadStats(ctx, input.BookID, 1, qualifiedDelta, lastCountedAt); err != nil {
		return nil, err
	}
	stats, err := txRepo.GetBookReadStats(ctx, input.BookID)
	if err != nil {
		return nil, err
	}
	res := &models.ReadingActivityEntity{
		Progress:        saved,
		Stats:           stats,
		Counted:         counted,
		CooldownSeconds: int64(readCountCooldown.Seconds()),
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if justFinished && s.webhookService != nil {
		payload := BuildBookWebhookPayload(book)
		payload["user_id"] = input.UserID
		s.webhookService.DispatchEvent(ctx, "reading.completed", payload)
	}
	return res.ToResponse(), nil
}

func (s *featureService) GetBookReadStats(ctx context.Context, bookID string) (*response.BookReadStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	stats, err := s.repo.GetBookReadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) RecordDownload(ctx context.Context, bookID string) error {
	if strings.TrimSpace(bookID) == "" {
		return apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	return s.repo.UpsertBookDownloadStats(ctx, bookID, 1)
}

func (s *featureService) GetBookDownloadStats(ctx context.Context, bookID string) (*response.BookDownloadStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	stats, err := s.repo.GetBookDownloadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) GetBookSocialStats(ctx context.Context, bookID string) (*response.BookSocialStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	stats, err := s.repo.GetBookSocialStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) GetBookEngagementStats(ctx context.Context, bookID string) (*response.BookEngagementStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	socialStats, err := s.repo.GetBookSocialStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	downloadStats, err := s.repo.GetBookDownloadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	readStats, err := s.repo.GetBookReadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	entity := &models.BookEngagementStatsEntity{
		BookID:        bookID,
		SocialStats:   socialStats,
		DownloadStats: downloadStats,
		ReadStats:     readStats,
	}
	return entity.ToResponse(), nil
}

func (s *featureService) RecordShare(ctx context.Context, input models.ShareInput) (*response.BookSocialStatsResponse, error) {
	bookID := strings.TrimSpace(input.BookID)
	if bookID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	actorKey := strings.TrimSpace(input.ActorKey)
	if actorKey == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "share actor is required")
	}
	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	windowBucket := occurredAt.Unix() / int64(shareCountWindow.Seconds())

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.repo.WithTx(tx)

	created, err := txRepo.CreateBookShareEvent(ctx, bookID, actorKey, windowBucket)
	if err != nil {
		return nil, err
	}
	if created {
		if err := txRepo.UpsertBookShareStats(ctx, bookID, 1); err != nil {
			return nil, err
		}
	}
	stats, err := txRepo.GetBookSocialStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) SetBookmark(ctx context.Context, userID string, bookID string, bookmarked bool) (*response.BookmarkResponse, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	bm, err := s.repo.SetBookmark(ctx, userID, bookID, bookmarked)
	if err != nil {
		return nil, err
	}
	return bm.ToResponse(), nil
}

func (s *featureService) GetBookmarkedBooks(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) (*response.BookmarkedBooksPageResponse, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "userId is required")
	}
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 20
	}
	bookmarkRows, err := s.repo.GetBookmarkedBooks(ctx, userID, cursor, cursorID, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(bookmarkRows))
	for i, row := range bookmarkRows {
		ids[i] = row.BookID
	}
	books, err := s.bookRepo.GetBooksByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	files, err := s.bookRepo.GetFilesByBookIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	filesByBookID := make(map[string][]*models.BookFileEntity, len(books))
	for _, file := range files {
		if file != nil {
			filesByBookID[file.BookID] = append(filesByBookID[file.BookID], file)
		}
	}
	for _, book := range books {
		if book != nil {
			book.Files = filesByBookID[book.ID]
		}
	}
	result := &models.BookmarkedBooksPage{Books: books}
	if len(bookmarkRows) >= int(limit) && len(bookmarkRows) > 0 {
		last := bookmarkRows[len(bookmarkRows)-1]
		result.NextCursor = last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.BookID
	}
	return result.ToResponse(), nil
}

func (s *featureService) GetBookUserState(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (*response.BookUserStateResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	if !s.PolicyAllowsBook(ctx, "read", bookID, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Book is not accessible")
	}
	bookmark, err := s.repo.GetBookmark(ctx, userID, bookID)
	if err != nil {
		return nil, err
	}
	myReview, err := s.repo.GetBookReview(ctx, userID, bookID)
	if err != nil {
		return nil, err
	}
	ratingSummary, err := s.repo.GetBookRatingSummary(ctx, bookID)
	if err != nil {
		return nil, err
	}
	socialStats, err := s.repo.GetBookSocialStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	downloadStats, err := s.repo.GetBookDownloadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	readStats, err := s.repo.GetBookReadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	collections, err := s.repo.GetBookCollectionIDs(ctx, userID, bookID)
	if err != nil {
		return nil, err
	}
	if collections == nil {
		collections = []string{}
	}
	entity := &models.BookUserStateEntity{
		BookID:        bookID,
		Bookmarked:    bookmark != nil,
		MyReview:      myReview,
		RatingSummary: ratingSummary,
		SocialStats:   socialStats,
		DownloadStats: downloadStats,
		ReadStats:     readStats,
		Collections:   collections,
	}
	return entity.ToResponse(), nil
}

func (s *featureService) UpsertBookReview(ctx context.Context, userID string, bookID string, rating int64, review string) (*response.BookReviewResponse, error) {
	if userID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	if rating < 1 || rating > 5 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "rating must be between 1 and 5")
	}
	var reviewPtr *string
	if value := strings.TrimSpace(review); value != "" {
		reviewPtr = &value
	}
	res, err := s.repo.UpsertBookReview(ctx, userID, bookID, rating, reviewPtr)
	if err != nil {
		return nil, err
	}
	return res.ToResponse(), nil
}

func (s *featureService) DeleteBookReview(ctx context.Context, userID string, bookID string) error {
	if userID == "" {
		return apperrors.New(apperrors.ErrBadRequest, "userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	return s.repo.DeleteBookReview(ctx, userID, bookID)
}

func (s *featureService) DeleteReviewByAdmin(ctx context.Context, targetUserID string, bookID string) error {
	if targetUserID == "" {
		return apperrors.New(apperrors.ErrBadRequest, "userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	return s.repo.DeleteBookReview(ctx, targetUserID, bookID)
}

func (s *featureService) ListAllReviews(ctx context.Context, limit, offset int64) ([]*response.BookReviewResponse, error) {
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 50
	}
	reviews, err := s.repo.ListAllReviews(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return models.BookReviewEntitiesToResponse(reviews), nil
}

func (s *featureService) ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, cursorID string, limit int64) ([]*response.BookReviewResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 20
	}
	reviews, err := s.repo.ListBookReviews(ctx, bookID, cursor, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return models.BookReviewEntitiesToResponse(reviews), nil
}

func (s *featureService) GetBookRatingSummary(ctx context.Context, bookID string) (*response.BookRatingSummaryResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "bookId is required")
	}
	summary, err := s.repo.GetBookRatingSummary(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return summary.ToResponse(), nil
}

func shouldCountQualifiedRead(existing *models.ReadingProgressEntity, now time.Time) bool {
	if existing == nil || existing.LastCountedAt == nil {
		return true
	}
	return now.Sub(*existing.LastCountedAt) >= readCountCooldown
}

func normalizeProgress(value *float64) *float64 {
	if value == nil {
		zero := 0.0
		return &zero
	}
	next := *value
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	return &next
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	next := strings.TrimSpace(*value)
	if next == "" {
		return nil
	}
	return &next
}

func (s *featureService) AddBookToCollection(ctx context.Context, userID string, collectionID string, bookID string) error {
	owned, err := s.repo.CollectionOwnedByUser(ctx, collectionID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return apperrors.New(apperrors.ErrForbidden, "Collection permission denied")
	}
	return s.repo.AddBookToCollection(ctx, collectionID, bookID)
}

func (s *featureService) RemoveBookFromCollection(ctx context.Context, userID string, collectionID string, bookID string) error {
	owned, err := s.repo.CollectionOwnedByUser(ctx, collectionID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return apperrors.New(apperrors.ErrForbidden, "Collection permission denied")
	}
	return s.repo.RemoveBookFromCollection(ctx, collectionID, bookID)
}

func (s *featureService) PolicyAllowsBook(ctx context.Context, policy string, bookID string, claims *response.JWTClaims) bool {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return false
	}
	c := resolveClaims(claims)
	if isGuestClaims(c) && !s.settings.GuestAllows(book.LibraryID) {
		return false
	}
	if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
		return true
	}
	permKey := "book." + policy
	if policy == "review" {
		permKey = constants.PermBookReviewCreate
	}
	return s.permissions.CanRoles(c.RoleIDs, c.Roles, permKey, map[string]any{"library_id": book.LibraryID})
}

func (s *featureService) PolicyAllowsNoBook(ctx context.Context, policy string, claims *response.JWTClaims) bool {
	c := resolveClaims(claims)
	if s.permissions.IsAdmin(c.RoleIDs, c.Roles) {
		return true
	}
	permKey := "book." + policy
	return s.permissions.CanRoles(c.RoleIDs, c.Roles, permKey, nil)
}

func (s *featureService) ShareActorKey(clientID string, ip string, userAgent string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = "anonymous"
	}
	sum := sha256.Sum256([]byte(clientID + "|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}

func readingSessionBookAccessible(book *models.BookEntity, err error, permissions PermissionCache, claims *response.JWTClaims) bool {
	if err != nil || book == nil || permissions == nil {
		return false
	}
	return permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermBookRead, map[string]any{"library_id": book.LibraryID})
}

func sessionDateOrServerDate(raw string) string {
	const layout = "2006-01-02"
	now := time.Now().UTC()
	serverDate := now.Format(layout)
	parsed, err := time.Parse(layout, strings.TrimSpace(raw))
	if err != nil {
		return serverDate
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if delta := parsed.Sub(today); delta < -24*time.Hour || delta > 24*time.Hour {
		return serverDate
	}
	return parsed.Format(layout)
}

func (s *featureService) RecordReadingSession(ctx context.Context, userID string, bookID string, duration int64, words int64, sessionDate string, claims *response.JWTClaims) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	resolved := resolveClaims(claims)
	if !readingSessionBookAccessible(book, err, s.permissions, resolved) {
		return apperrors.New(apperrors.ErrForbidden, "Book is not accessible")
	}
	_, err = s.repo.UpsertReadingSession(ctx, sqlc.UpsertReadingSessionParams{
		ID:              uuid.Must(uuid.NewV7()).String(),
		UserID:          userID,
		BookID:          bookID,
		SessionDate:     sessionDateOrServerDate(sessionDate),
		DurationSeconds: duration,
		WordsRead:       words,
	})
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Str("book_id", bookID).Msg("failed to record reading session")
		return apperrors.New(errors.Join(apperrors.ErrInternalError, err), "Failed to record reading session")
	}
	return nil
}

func (s *featureService) GetReadingHeatmap(ctx context.Context, userID string) (map[string]map[string]int64, error) {
	rows, err := s.repo.GetReadingHeatmap(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get reading heatmap")
	}

	result := make(map[string]map[string]int64)
	for _, r := range rows {
		dateStr := r.Date.Format("2006-01-02")
		result[dateStr] = map[string]int64{
			"duration": r.DurationSeconds,
			"words":    r.WordsRead,
		}
	}
	return result, nil
}

func computeStreaks(active map[string]struct{}, today time.Time) (current, longest int64) {
	day := func(t time.Time) string { return t.Format("2006-01-02") }

	start := today
	if _, ok := active[day(today)]; !ok {
		start = today.AddDate(0, 0, -1)
	}
	for {
		if _, ok := active[day(start)]; !ok {
			break
		}
		current++
		start = start.AddDate(0, 0, -1)
	}

	earliest := today.AddDate(0, 0, -364)
	for d := earliest; !d.After(today); d = d.AddDate(0, 0, 1) {
		if _, ok := active[day(d)]; !ok {
			continue
		}
		run := int64(1)
		for {
			next := d.AddDate(0, 0, 1)
			if _, ok := active[day(next)]; !ok {
				break
			}
			run++
			d = next
		}
		if run > longest {
			longest = run
		}
	}
	return current, longest
}

func (s *featureService) GetReadingStatsSummary(ctx context.Context, userID string) (*response.ReadingStatsSummaryResponse, error) {
	rows, err := s.repo.GetReadingHeatmap(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get reading stats")
	}

	active := make(map[string]struct{}, len(rows))
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	var totalWords, totalMinutes, wordsToday int64
	for _, r := range rows {
		dateStr := r.Date.Format("2006-01-02")
		active[dateStr] = struct{}{}
		totalWords += r.WordsRead
		totalMinutes += r.DurationSeconds / 60
		if dateStr == todayStr {
			wordsToday += r.WordsRead
		}
	}

	current, longest := computeStreaks(active, now)

	goal, err := s.repo.GetReadingGoal(ctx, userID)
	if err != nil {
		if !apperrors.IsNotFound(err) {
			return nil, err
		}
		goal = nil
	}
	targetWords := int64(defaultTargetWordsPerDay)
	targetBooks := int64(defaultTargetBooksPerYear)
	if goal != nil {
		targetWords = goal.TargetWordsPerDay
		targetBooks = goal.TargetBooksPerYear
	}

	return &response.ReadingStatsSummaryResponse{
		CurrentStreakDays:  current,
		LongestStreakDays:  longest,
		TotalActiveDays:    int64(len(active)),
		TotalWords:         totalWords,
		TotalMinutes:       totalMinutes,
		WordsToday:         wordsToday,
		WordsTodayTarget:   targetWords,
		BooksPerYearTarget: targetBooks,
	}, nil
}

func (s *featureService) GetReaderETA(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (*response.ReadingETAResponse, error) {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	resolved := resolveClaims(claims)
	if !readingSessionBookAccessible(book, err, s.permissions, resolved) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Book is not accessible")
	}

	stats, err := s.repo.GetReadingStatsByBook(ctx, userID, bookID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get reading stats")
	}

	wordsRead := int64(0)
	totalMinutes := int64(0)
	if stats != nil {
		wordsRead = stats.TotalWords
		totalMinutes = stats.TotalDuration / 60
	}

	pace := float64(0)
	if stats == nil || totalMinutes == 0 {
		global, gerr := s.repo.GetReadingStatsSince(ctx, userID, time.Now().AddDate(0, 0, -30))
		if gerr != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get reading stats")
		}
		if global.TotalDuration > 0 {
			pace = float64(global.TotalWords) / (float64(global.TotalDuration) / 60)
		}
	} else {
		pace = float64(wordsRead) / float64(totalMinutes)
	}

	res := &response.ReadingETAResponse{
		PaceWordsPerMin: pace,
		WordsRead:       wordsRead,
	}

	progress, err := s.repo.GetReadingProgress(ctx, userID, bookID)
	if err == nil && progress != nil && progress.ProgressPercent != nil && *progress.ProgressPercent > 0 {
		percent := *progress.ProgressPercent
		estTotal := float64(wordsRead) / (percent / 100)
		remaining := estTotal * (100 - percent) / 100
		if remaining < 0 {
			remaining = 0
		}
		res.Percent = percent
		res.RemainingWords = int64(remaining)
		if pace > 0 {
			res.EtaMinutes = int64(remaining / pace)
		}
	}
	return res, nil
}

func nameCountsToResponse(entities []*models.NameCountEntity) []response.NameCountResponse {
	out := make([]response.NameCountResponse, 0, len(entities))
	for _, e := range entities {
		if e == nil {
			continue
		}
		out = append(out, response.NameCountResponse{Name: e.Name, Count: e.Count})
	}
	return out
}

func (s *featureService) GetLibraryBreakdown(ctx context.Context, userID string) (*response.LibraryBreakdownResponse, error) {
	breakdown, err := s.repo.GetLibraryBreakdown(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get library breakdown")
	}

	listening, err := s.repo.GetListeningHistory(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get listening history")
	}

	listeningStats, err := s.repo.GetListeningStats(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get listening stats")
	}

	res := &response.LibraryBreakdownResponse{
		Formats:    nameCountsToResponse(breakdown.Formats),
		Tags:       nameCountsToResponse(breakdown.Tags),
		Authors:    nameCountsToResponse(breakdown.Authors),
		Publishers: nameCountsToResponse(breakdown.Publishers),
		Languages:  nameCountsToResponse(breakdown.Languages),
		Listening:  make([]response.ListeningMonthCount, 0, len(listening)),
	}
	if listeningStats != nil && listeningStats.TotalDuration > 0 {
		res.AvgSpeedWpm = float64(listeningStats.TotalWords) / (float64(listeningStats.TotalDuration) / 60)
	}
	for _, l := range listening {
		if l == nil {
			continue
		}
		res.Listening = append(res.Listening, response.ListeningMonthCount{
			Month: l.Month,
			Hours: l.TotalSeconds / 3600,
		})
	}
	return res, nil
}

// Defaults mirror the reading_goals table (db/schema/58_reading_goals.sql). A user
// who never set a goal is a normal state, not a 404 — the analytics page always
// needs something to divide by.
const (
	defaultTargetWordsPerDay  = 1000
	defaultTargetBooksPerYear = 12
)

func (s *featureService) GetReadingGoal(ctx context.Context, userID string) (*response.ReadingGoalResponse, error) {
	goal, err := s.repo.GetReadingGoal(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return &response.ReadingGoalResponse{
				UserID:             userID,
				TargetWordsPerDay:  defaultTargetWordsPerDay,
				TargetBooksPerYear: defaultTargetBooksPerYear,
			}, nil
		}
		return nil, err
	}
	return goal.ToResponse(), nil
}

func (s *featureService) UpsertReadingGoal(ctx context.Context, userID string, wordsPerDay int64, booksPerYear int64) (*response.ReadingGoalResponse, error) {
	goal, err := s.repo.UpsertReadingGoal(ctx, userID, wordsPerDay, booksPerYear)
	if err != nil {
		return nil, err
	}
	return goal.ToResponse(), nil
}

func smartCollectionToResponse(entity *models.SmartCollectionEntity) *response.SmartCollectionResponse {
	res := entity.ToResponse()
	if res == nil {
		return nil
	}
	var rule request.SmartCollectionRuleDto
	if err := jsonx.Unmarshal([]byte(entity.RuleJson), &rule); err == nil {
		res.Rule = rule
	}
	return res
}

func (s *featureService) ListSmartCollections(ctx context.Context, userID string) ([]*response.SmartCollectionResponse, error) {
	entities, err := s.repo.ListSmartCollections(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SmartCollectionResponse, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		out = append(out, smartCollectionToResponse(entity))
	}
	return out, nil
}

func (s *featureService) CreateSmartCollection(ctx context.Context, userID string, dto request.UpsertSmartCollectionDto) (*response.SmartCollectionResponse, error) {
	ruleJson, err := jsonx.Marshal(dto.Rule)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid smart collection rule")
	}
	id := uuid.Must(uuid.NewV7()).String()
	entity, err := s.repo.CreateSmartCollection(ctx, id, userID, dto.Name, string(ruleJson))
	if err != nil {
		return nil, err
	}
	return smartCollectionToResponse(entity), nil
}

func (s *featureService) UpdateSmartCollection(ctx context.Context, id string, userID string, dto request.UpsertSmartCollectionDto) (*response.SmartCollectionResponse, error) {
	ruleJson, err := jsonx.Marshal(dto.Rule)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid smart collection rule")
	}
	entity, err := s.repo.UpdateSmartCollection(ctx, id, userID, dto.Name, string(ruleJson))
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Smart collection not found")
		}
		return nil, err
	}
	return smartCollectionToResponse(entity), nil
}

func (s *featureService) DeleteSmartCollection(ctx context.Context, id string, userID string) error {
	return s.repo.DeleteSmartCollection(ctx, id, userID)
}

func (s *featureService) ListSmartFilters(ctx context.Context, userID string) ([]*response.SmartFilterResponse, error) {
	entities, err := s.repo.ListSmartFilters(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SmartFilterResponse, len(entities))
	for i, entity := range entities {
		out[i] = entity.ToResponse()
	}
	return out, nil
}

func (s *featureService) GetSmartFilter(ctx context.Context, id string, userID string) (*response.SmartFilterResponse, error) {
	entity, err := s.repo.GetSmartFilter(ctx, id, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Smart filter not found")
		}
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *featureService) CreateSmartFilter(ctx context.Context, userID string, dto request.UpsertSmartFilterDto) (*response.SmartFilterResponse, error) {
	rulesJson, err := jsonx.Marshal(dto.Rules)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid smart filter rules")
	}
	id := uuid.Must(uuid.NewV7()).String()

	// Get current list of smart filters to compute next position
	filters, err := s.repo.ListSmartFilters(ctx, userID)
	var nextPos int64 = 0
	if err == nil {
		for _, f := range filters {
			if f.HomePosition >= nextPos {
				nextPos = f.HomePosition + 1
			}
		}
	}

	entity, err := s.repo.CreateSmartFilter(ctx, id, userID, dto.Name, string(rulesJson), dto.IsPinnedSidebar, dto.IsPinnedHome, nextPos)
	if err != nil {
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *featureService) UpdateSmartFilter(ctx context.Context, id string, userID string, dto request.UpsertSmartFilterDto) (*response.SmartFilterResponse, error) {
	rulesJson, err := jsonx.Marshal(dto.Rules)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid smart filter rules")
	}
	entity, err := s.repo.UpdateSmartFilter(ctx, id, userID, dto.Name, string(rulesJson), dto.IsPinnedSidebar, dto.IsPinnedHome)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Smart filter not found")
		}
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *featureService) DeleteSmartFilter(ctx context.Context, id string, userID string) error {
	_, err := s.repo.GetSmartFilter(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.New(apperrors.ErrNotFound, "Smart filter not found")
		}
		return err
	}
	return s.repo.DeleteSmartFilter(ctx, id, userID)
}

func (s *featureService) UpdateSmartFilterPinSidebar(ctx context.Context, id string, userID string, isPinned bool) (*response.SmartFilterResponse, error) {
	entity, err := s.repo.UpdateSmartFilterPinSidebar(ctx, id, userID, isPinned)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Smart filter not found")
		}
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *featureService) UpdateSmartFilterPinHome(ctx context.Context, id string, userID string, isPinned bool) (*response.SmartFilterResponse, error) {
	entity, err := s.repo.UpdateSmartFilterPinHome(ctx, id, userID, isPinned)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Smart filter not found")
		}
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *featureService) ReorderSmartFiltersHome(ctx context.Context, userID string, dto request.ReorderHomeShelvesDto) error {
	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txRepo := s.repo.WithTx(tx)
	for _, shelf := range dto.Shelves {
		_, err := txRepo.UpdateSmartFilterHomePosition(ctx, shelf.ID, userID, shelf.Position)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
