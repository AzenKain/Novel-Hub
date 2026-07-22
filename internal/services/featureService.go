package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

type FeatureService interface {
	GetLibraryStats(ctx context.Context) (*response.LibraryStatsResponse, error)
	CreateCollection(ctx context.Context, name string, userID int64) (*response.CollectionResponse, error)
	UpdateCollection(ctx context.Context, id, name string, userID int64) (*response.CollectionResponse, error)
	DeleteCollection(ctx context.Context, id string, userID int64) error
	GetUserCollections(ctx context.Context, userID int64, cursorCreatedAt *time.Time, limit int64) ([]*response.CollectionResponse, error)
	GetRecentReadingHistory(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*response.ReadingHistoryResponse, error)
	GetReadingProgress(ctx context.Context, userID int64, bookID string) (*response.ReadingProgressResponse, error)
	RecordReadingActivity(ctx context.Context, input models.ReadingActivityInput) (*response.ReadingActivityResponse, error)
	GetBookReadStats(ctx context.Context, bookID string) (*response.BookReadStatsResponse, error)
	RecordDownload(ctx context.Context, bookID string) error
	GetBookDownloadStats(ctx context.Context, bookID string) (*response.BookDownloadStatsResponse, error)
	GetBookSocialStats(ctx context.Context, bookID string) (*response.BookSocialStatsResponse, error)
	GetBookEngagementStats(ctx context.Context, bookID string) (*response.BookEngagementStatsResponse, error)
	RecordShare(ctx context.Context, input models.ShareInput) (*response.BookSocialStatsResponse, error)
	SetBookmark(ctx context.Context, userID int64, bookID string, bookmarked bool) (*response.BookmarkResponse, error)
	GetBookmarkedBooks(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*models.BookEntity, error)
	GetBookUserState(ctx context.Context, userID int64, bookID string) (*response.BookUserStateResponse, error)
	UpsertBookReview(ctx context.Context, userID int64, bookID string, rating int64, review string) (*response.BookReviewResponse, error)
	DeleteBookReview(ctx context.Context, userID int64, bookID string) error
	DeleteReviewByAdmin(ctx context.Context, targetUserID int64, bookID string) error
	ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, limit int64) ([]*response.BookReviewResponse, error)
	ListAllReviews(ctx context.Context, limit, offset int64) ([]*response.BookReviewResponse, error)
	GetBookRatingSummary(ctx context.Context, bookID string) (*response.BookRatingSummaryResponse, error)
	AddBookToCollection(ctx context.Context, userID int64, collectionID string, bookID string) error
	RemoveBookFromCollection(ctx context.Context, userID int64, collectionID string, bookID string) error

	PolicyAllowsBook(ctx context.Context, policy string, bookID string, claims *response.JWTClaims) bool
	PolicyAllowsNoBook(ctx context.Context, policy string, claims *response.JWTClaims) bool
	ShareActorKey(clientID string, ip string, userAgent string) string
	RecordReadingSession(ctx context.Context, userID int64, bookID string, duration int64, words int64) error
	GetReadingHeatmap(ctx context.Context, userID int64) (map[string]map[string]int64, error)
}

type featureService struct {
	repo        repositories.FeatureRepository
	bookRepo    repositories.BookDBRepository
	settings    SettingsService
	permissions PermissionCache
	txManager   database.TxManager
	activityMu  sync.Mutex
}

const (
	readCountCooldown = 12 * time.Hour
	shareCountWindow  = time.Hour
)

func NewFeatureService(repo repositories.FeatureRepository, bookRepo repositories.BookDBRepository, settings SettingsService, permissions PermissionCache, txManager database.TxManager) FeatureService {
	return &featureService{repo: repo, bookRepo: bookRepo, settings: settings, permissions: permissions, txManager: txManager}
}

func (s *featureService) GetLibraryStats(ctx context.Context) (*response.LibraryStatsResponse, error) {
	stats, err := s.repo.GetLibraryStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) CreateCollection(ctx context.Context, name string, userID int64) (*response.CollectionResponse, error) {
	id := uuid.Must(uuid.NewV7()).String()
	col, err := s.repo.CreateCollection(ctx, id, name, userID)
	if err != nil {
		return nil, err
	}
	return col.ToResponse(), nil
}

func (s *featureService) UpdateCollection(ctx context.Context, id, name string, userID int64) (*response.CollectionResponse, error) {
	col, err := s.repo.UpdateCollection(ctx, id, name, userID)
	if err != nil {
		return nil, err
	}
	return col.ToResponse(), nil
}

func (s *featureService) DeleteCollection(ctx context.Context, id string, userID int64) error {
	return s.repo.DeleteCollection(ctx, id, userID)
}

func (s *featureService) GetUserCollections(ctx context.Context, userID int64, cursorCreatedAt *time.Time, limit int64) ([]*response.CollectionResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	cols, err := s.repo.GetUserCollections(ctx, userID, cursorCreatedAt, limit)
	if err != nil {
		return nil, err
	}
	return models.CollectionEntitiesToResponse(cols), nil
}

func (s *featureService) GetRecentReadingHistory(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*response.ReadingHistoryResponse, error) {
	history, err := s.repo.GetRecentReadingHistory(ctx, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	return models.ReadingHistoryEntitiesToResponse(history), nil
}

func (s *featureService) GetReadingProgress(ctx context.Context, userID int64, bookID string) (*response.ReadingProgressResponse, error) {
	entity, err := s.repo.GetReadingProgress(ctx, userID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.New(apperrors.ErrNotFound, "No reading progress found")
		}
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *featureService) RecordReadingActivity(ctx context.Context, input models.ReadingActivityInput) (*response.ReadingActivityResponse, error) {
	if strings.TrimSpace(input.BookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	if strings.TrimSpace(input.ChapterID) == "" {
		return nil, fmt.Errorf("chapterId is required")
	}

	s.activityMu.Lock()
	defer s.activityMu.Unlock()

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.repo.WithTx(tx)

	now := time.Now().UTC()
	existing, err := txRepo.GetReadingProgress(ctx, input.UserID, input.BookID)
	if err != nil {
		return nil, err
	}

	openedCount := int64(1)
	qualifiedReadCount := int64(0)
	var lastCountedAt *time.Time
	if existing != nil {
		openedCount = existing.OpenedCount + 1
		qualifiedReadCount = existing.QualifiedReadCount
		lastCountedAt = existing.LastCountedAt
	}

	counted := shouldCountQualifiedRead(existing, now)
	qualifiedDelta := int64(0)
	if counted {
		qualifiedReadCount++
		qualifiedDelta = 1
		lastCountedAt = &now
	}

	progressPercent := normalizeProgress(input.ProgressPercent)
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
		OpenedCount:        openedCount,
		QualifiedReadCount: qualifiedReadCount,
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
	return res.ToResponse(), nil
}

func (s *featureService) GetBookReadStats(ctx context.Context, bookID string) (*response.BookReadStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	stats, err := s.repo.GetBookReadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) RecordDownload(ctx context.Context, bookID string) error {
	if strings.TrimSpace(bookID) == "" {
		return fmt.Errorf("bookId is required")
	}
	return s.repo.UpsertBookDownloadStats(ctx, bookID, 1)
}

func (s *featureService) GetBookDownloadStats(ctx context.Context, bookID string) (*response.BookDownloadStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	stats, err := s.repo.GetBookDownloadStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) GetBookSocialStats(ctx context.Context, bookID string) (*response.BookSocialStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	stats, err := s.repo.GetBookSocialStats(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *featureService) GetBookEngagementStats(ctx context.Context, bookID string) (*response.BookEngagementStatsResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
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
		return nil, fmt.Errorf("bookId is required")
	}
	actorKey := strings.TrimSpace(input.ActorKey)
	if actorKey == "" {
		return nil, fmt.Errorf("share actor is required")
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

func (s *featureService) SetBookmark(ctx context.Context, userID int64, bookID string, bookmarked bool) (*response.BookmarkResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	bm, err := s.repo.SetBookmark(ctx, userID, bookID, bookmarked)
	if err != nil {
		return nil, err
	}
	return bm.ToResponse(), nil
}

func (s *featureService) GetBookmarkedBooks(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*models.BookEntity, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("userId is required")
	}
	if limit <= 0 {
		limit = 20
	}
	ids, err := s.repo.GetBookmarkedBookIDs(ctx, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	books, err := s.bookRepo.GetBooksByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, book := range books {
		if files, err := s.bookRepo.GetFilesByBookId(ctx, book.ID); err == nil {
			book.Files = files
		}
	}
	return books, nil
}

func (s *featureService) GetBookUserState(ctx context.Context, userID int64, bookID string) (*response.BookUserStateResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
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

func (s *featureService) UpsertBookReview(ctx context.Context, userID int64, bookID string, rating int64, review string) (*response.BookReviewResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
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

func (s *featureService) DeleteBookReview(ctx context.Context, userID int64, bookID string) error {
	if userID <= 0 {
		return fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return fmt.Errorf("bookId is required")
	}
	return s.repo.DeleteBookReview(ctx, userID, bookID)
}

func (s *featureService) DeleteReviewByAdmin(ctx context.Context, targetUserID int64, bookID string) error {
	if targetUserID <= 0 {
		return fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return fmt.Errorf("bookId is required")
	}
	return s.repo.DeleteBookReview(ctx, targetUserID, bookID)
}

func (s *featureService) ListAllReviews(ctx context.Context, limit, offset int64) ([]*response.BookReviewResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	reviews, err := s.repo.ListAllReviews(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return models.BookReviewEntitiesToResponse(reviews), nil
}

func (s *featureService) ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, limit int64) ([]*response.BookReviewResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	if limit <= 0 {
		limit = 20
	}
	reviews, err := s.repo.ListBookReviews(ctx, bookID, cursor, limit)
	if err != nil {
		return nil, err
	}
	return models.BookReviewEntitiesToResponse(reviews), nil
}

func (s *featureService) GetBookRatingSummary(ctx context.Context, bookID string) (*response.BookRatingSummaryResponse, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
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

func (s *featureService) AddBookToCollection(ctx context.Context, userID int64, collectionID string, bookID string) error {
	cols, err := s.repo.GetUserCollections(ctx, userID, nil, 1000)
	if err != nil {
		return err
	}
	found := false
	for _, c := range cols {
		if c.ID == collectionID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("collection not found or does not belong to user")
	}
	return s.repo.AddBookToCollection(ctx, collectionID, bookID)
}

func (s *featureService) RemoveBookFromCollection(ctx context.Context, userID int64, collectionID string, bookID string) error {
	cols, err := s.repo.GetUserCollections(ctx, userID, nil, 1000)
	if err != nil {
		return err
	}
	found := false
	for _, c := range cols {
		if c.ID == collectionID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("collection not found or does not belong to user")
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

func (s *featureService) RecordReadingSession(ctx context.Context, userID int64, bookID string, duration int64, words int64) error {
	_, err := s.repo.UpsertReadingSession(ctx, sqlc.UpsertReadingSessionParams{
		UserID:          userID,
		BookID:          bookID,
		DurationSeconds: duration,
		WordsRead:       words,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to record reading session")
	}
	return nil
}

func (s *featureService) GetReadingHeatmap(ctx context.Context, userID int64) (map[string]map[string]int64, error) {
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
