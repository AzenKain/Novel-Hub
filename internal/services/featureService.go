package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
)

type FeatureService interface {
	GetLibraryStats(ctx context.Context) (*models.LibraryStatsEntity, error)
	CreateCollection(ctx context.Context, name string, userID int64) (*models.CollectionEntity, error)
	UpdateCollection(ctx context.Context, id, name string, userID int64) (*models.CollectionEntity, error)
	DeleteCollection(ctx context.Context, id string, userID int64) error
	GetUserCollections(ctx context.Context, userID int64, cursorCreatedAt *time.Time, limit int64) ([]*models.CollectionEntity, error)
	GetRecentReadingHistory(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*models.ReadingHistoryEntity, error)
	RecordReadingActivity(ctx context.Context, input models.ReadingActivityInput) (*models.ReadingActivityEntity, error)
	GetBookReadStats(ctx context.Context, bookID string) (*models.BookReadStatsEntity, error)
	RecordDownload(ctx context.Context, bookID string) error
	GetBookDownloadStats(ctx context.Context, bookID string) (*models.BookDownloadStatsEntity, error)
	GetBookSocialStats(ctx context.Context, bookID string) (*models.BookSocialStatsEntity, error)
	GetBookEngagementStats(ctx context.Context, bookID string) (*models.BookEngagementStatsEntity, error)
	RecordShare(ctx context.Context, input models.ShareInput) (*models.BookSocialStatsEntity, error)
	SetBookmark(ctx context.Context, userID int64, bookID string, bookmarked bool) (*models.BookmarkEntity, error)
	GetBookmarkedBooks(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*models.BookEntity, error)
	GetBookUserState(ctx context.Context, userID int64, bookID string) (*models.BookUserStateEntity, error)
	UpsertBookReview(ctx context.Context, userID int64, bookID string, rating int64, review string) (*models.BookReviewEntity, error)
	DeleteBookReview(ctx context.Context, userID int64, bookID string) error
	DeleteReviewByAdmin(ctx context.Context, targetUserID int64, bookID string) error
	ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, limit int64) ([]*models.BookReviewEntity, error)
	ListAllReviews(ctx context.Context, limit, offset int64) ([]*models.BookReviewEntity, error)
	GetBookRatingSummary(ctx context.Context, bookID string) (*models.BookRatingSummaryEntity, error)
	AddBookToCollection(ctx context.Context, userID int64, collectionID string, bookID string) error
	RemoveBookFromCollection(ctx context.Context, userID int64, collectionID string, bookID string) error

	PolicyAllowsBook(ctx context.Context, policy string, bookID string, claims *response.JWTClaims) bool
	PolicyAllowsNoBook(ctx context.Context, policy string, claims *response.JWTClaims) bool
	ShareActorKey(clientID string, ip string, userAgent string) string
}

type featureService struct {
	repo        repositories.FeatureRepository
	bookRepo    repositories.BookDBRepository
	settings    SettingsService
	permissions PermissionCache
	activityMu  sync.Mutex
}

const (
	readCountCooldown = 12 * time.Hour
	shareCountWindow  = time.Hour
)

func NewFeatureService(repo repositories.FeatureRepository, bookRepo repositories.BookDBRepository, settings SettingsService, permissions PermissionCache) FeatureService {
	return &featureService{repo: repo, bookRepo: bookRepo, settings: settings, permissions: permissions}
}

func (s *featureService) GetLibraryStats(ctx context.Context) (*models.LibraryStatsEntity, error) {
	return s.repo.GetLibraryStats(ctx)
}

func (s *featureService) CreateCollection(ctx context.Context, name string, userID int64) (*models.CollectionEntity, error) {
	id := uuid.Must(uuid.NewV7()).String()
	return s.repo.CreateCollection(ctx, id, name, userID)
}

func (s *featureService) UpdateCollection(ctx context.Context, id, name string, userID int64) (*models.CollectionEntity, error) {
	return s.repo.UpdateCollection(ctx, id, name, userID)
}

func (s *featureService) DeleteCollection(ctx context.Context, id string, userID int64) error {
	return s.repo.DeleteCollection(ctx, id, userID)
}

func (s *featureService) GetUserCollections(ctx context.Context, userID int64, cursorCreatedAt *time.Time, limit int64) ([]*models.CollectionEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetUserCollections(ctx, userID, cursorCreatedAt, limit)
}

func (s *featureService) GetRecentReadingHistory(ctx context.Context, userID int64, cursor *time.Time, limit int64) ([]*models.ReadingHistoryEntity, error) {
	return s.repo.GetRecentReadingHistory(ctx, userID, cursor, limit)
}

func (s *featureService) RecordReadingActivity(ctx context.Context, input models.ReadingActivityInput) (*models.ReadingActivityEntity, error) {
	if strings.TrimSpace(input.BookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	if strings.TrimSpace(input.ChapterID) == "" {
		return nil, fmt.Errorf("chapterId is required")
	}

	s.activityMu.Lock()
	defer s.activityMu.Unlock()

	now := time.Now().UTC()
	existing, err := s.repo.GetReadingProgress(ctx, input.UserID, input.BookID)
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
		OpenedCount:        openedCount,
		QualifiedReadCount: qualifiedReadCount,
		LastCountedAt:      lastCountedAt,
	}

	saved, err := s.repo.UpsertReadingProgress(ctx, progress)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpsertBookReadStats(ctx, input.BookID, 1, qualifiedDelta, lastCountedAt); err != nil {
		return nil, err
	}
	stats, err := s.repo.GetBookReadStats(ctx, input.BookID)
	if err != nil {
		return nil, err
	}
	return &models.ReadingActivityEntity{
		Progress:        saved,
		Stats:           stats,
		Counted:         counted,
		CooldownSeconds: int64(readCountCooldown.Seconds()),
	}, nil
}

func (s *featureService) GetBookReadStats(ctx context.Context, bookID string) (*models.BookReadStatsEntity, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	return s.repo.GetBookReadStats(ctx, bookID)
}

func (s *featureService) RecordDownload(ctx context.Context, bookID string) error {
	if strings.TrimSpace(bookID) == "" {
		return fmt.Errorf("bookId is required")
	}
	return s.repo.UpsertBookDownloadStats(ctx, bookID, 1)
}

func (s *featureService) GetBookDownloadStats(ctx context.Context, bookID string) (*models.BookDownloadStatsEntity, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	return s.repo.GetBookDownloadStats(ctx, bookID)
}

func (s *featureService) GetBookSocialStats(ctx context.Context, bookID string) (*models.BookSocialStatsEntity, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	return s.repo.GetBookSocialStats(ctx, bookID)
}

func (s *featureService) GetBookEngagementStats(ctx context.Context, bookID string) (*models.BookEngagementStatsEntity, error) {
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
	return &models.BookEngagementStatsEntity{
		BookID:        bookID,
		SocialStats:   socialStats,
		DownloadStats: downloadStats,
		ReadStats:     readStats,
	}, nil
}

func (s *featureService) RecordShare(ctx context.Context, input models.ShareInput) (*models.BookSocialStatsEntity, error) {
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
	created, err := s.repo.CreateBookShareEvent(ctx, bookID, actorKey, windowBucket)
	if err != nil {
		return nil, err
	}
	if created {
		if err := s.repo.UpsertBookShareStats(ctx, bookID, 1); err != nil {
			return nil, err
		}
	}
	return s.repo.GetBookSocialStats(ctx, bookID)
}

func (s *featureService) SetBookmark(ctx context.Context, userID int64, bookID string, bookmarked bool) (*models.BookmarkEntity, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	return s.repo.SetBookmark(ctx, userID, bookID, bookmarked)
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

func (s *featureService) GetBookUserState(ctx context.Context, userID int64, bookID string) (*models.BookUserStateEntity, error) {
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
	return &models.BookUserStateEntity{
		BookID:        bookID,
		Bookmarked:    bookmark != nil,
		MyReview:      myReview,
		RatingSummary: ratingSummary,
		SocialStats:   socialStats,
		DownloadStats: downloadStats,
		ReadStats:     readStats,
		Collections:   collections,
	}, nil
}

func (s *featureService) UpsertBookReview(ctx context.Context, userID int64, bookID string, rating int64, review string) (*models.BookReviewEntity, error) {
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
	return s.repo.UpsertBookReview(ctx, userID, bookID, rating, reviewPtr)
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

func (s *featureService) ListAllReviews(ctx context.Context, limit, offset int64) ([]*models.BookReviewEntity, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListAllReviews(ctx, limit, offset)
}

func (s *featureService) ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, limit int64) ([]*models.BookReviewEntity, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListBookReviews(ctx, bookID, cursor, limit)
}

func (s *featureService) GetBookRatingSummary(ctx context.Context, bookID string) (*models.BookRatingSummaryEntity, error) {
	if strings.TrimSpace(bookID) == "" {
		return nil, fmt.Errorf("bookId is required")
	}
	return s.repo.GetBookRatingSummary(ctx, bookID)
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
	admin := claims != nil && s.permissions.IsAdmin(claims.RoleIDs, claims.Roles)
	return s.settings.PolicyAllows(policy, book.LibraryID, admin)
}

func (s *featureService) PolicyAllowsNoBook(ctx context.Context, policy string, claims *response.JWTClaims) bool {
	admin := claims != nil && s.permissions.IsAdmin(claims.RoleIDs, claims.Roles)
	if admin {
		return true
	}
	settings, err := s.settings.Public(ctx)
	if err != nil {
		return false
	}
	switch policy {
	case "collection":
		return settings.Collection.Mode != "disabled"
	default:
		return false
	}
}

func (s *featureService) ShareActorKey(clientID string, ip string, userAgent string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = "anonymous"
	}
	sum := sha256.Sum256([]byte(clientID + "|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}

