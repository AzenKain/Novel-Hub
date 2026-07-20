package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type FeatureRepository interface {
	GetLibraryStats(ctx context.Context) (*models.LibraryStatsEntity, error)
	CreateCollection(ctx context.Context, id, name string, userID int64) (*models.CollectionEntity, error)
	GetUserCollections(ctx context.Context, userID int64) ([]*models.CollectionEntity, error)
	GetRecentReadingHistory(ctx context.Context, userID int64, limit int64) ([]*models.ReadingHistoryEntity, error)
	GetReadingProgress(ctx context.Context, userID int64, bookID string) (*models.ReadingProgressEntity, error)
	UpsertReadingProgress(ctx context.Context, progress *models.ReadingProgressEntity) (*models.ReadingProgressEntity, error)
	UpsertBookReadStats(ctx context.Context, bookID string, openDelta int64, qualifiedDelta int64, lastCountedAt *time.Time) error
	GetBookReadStats(ctx context.Context, bookID string) (*models.BookReadStatsEntity, error)
	UpsertBookDownloadStats(ctx context.Context, bookID string, downloadDelta int64) error
	GetBookDownloadStats(ctx context.Context, bookID string) (*models.BookDownloadStatsEntity, error)
	GetBookmark(ctx context.Context, userID int64, bookID string) (*models.BookmarkEntity, error)
	SetBookmark(ctx context.Context, userID int64, bookID string, bookmarked bool) (*models.BookmarkEntity, error)
	GetBookmarkedBookIDs(ctx context.Context, userID int64, limit, offset int64) ([]string, error)
	UpsertBookReview(ctx context.Context, userID int64, bookID string, rating int64, review *string) (*models.BookReviewEntity, error)
	DeleteBookReview(ctx context.Context, userID int64, bookID string) error
	GetBookReview(ctx context.Context, userID int64, bookID string) (*models.BookReviewEntity, error)
	ListBookReviews(ctx context.Context, bookID string, limit, offset int64) ([]*models.BookReviewEntity, error)
	ListAllReviews(ctx context.Context, limit, offset int64) ([]*models.BookReviewEntity, error)
	GetBookRatingSummary(ctx context.Context, bookID string) (*models.BookRatingSummaryEntity, error)
	GetBookSocialStats(ctx context.Context, bookID string) (*models.BookSocialStatsEntity, error)
	CreateBookShareEvent(ctx context.Context, bookID string, actorKey string, windowBucket int64) (bool, error)
	UpsertBookShareStats(ctx context.Context, bookID string, shareDelta int64) error
	AddBookToCollection(ctx context.Context, collectionID string, bookID string) error
	RemoveBookFromCollection(ctx context.Context, collectionID string, bookID string) error
	GetBookCollectionIDs(ctx context.Context, userID int64, bookID string) ([]string, error)
}

type featureRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
}

func NewFeatureRepository(db *sql.DB, c cache.Cache) FeatureRepository {
	return &featureRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
	}
}

func (r *featureRepository) GetLibraryStats(ctx context.Context) (*models.LibraryStatsEntity, error) {
	key := "feature:library_stats"
	if r.c != nil {
		var stats models.LibraryStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}
	stats, err := r.queries.GetLibraryStats(ctx)
	if err != nil {
		return nil, err
	}
	result := (&models.LibraryStatsEntity{}).FromSqlc(stats)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) CreateCollection(ctx context.Context, id, name string, userID int64) (*models.CollectionEntity, error) {
	collection, err := r.queries.CreateCollection(ctx, sqlc.CreateCollectionParams{
		ID:     id,
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	result := (&models.CollectionEntity{}).FromSqlc(collection)
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("collection:user:%d", userID))
		_ = r.c.Set(ctx, fmt.Sprintf("collection:id:%s", result.ID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) GetUserCollections(ctx context.Context, userID int64) ([]*models.CollectionEntity, error) {
	key := fmt.Sprintf("collection:user:%d", userID)
	if r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.getCollectionsByIDs(ctx, ids)
		}
	}

	ids, err := r.queries.GetUserCollectionIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
	}
	return r.getCollectionsByIDs(ctx, ids)
}

func (r *featureRepository) GetRecentReadingHistory(ctx context.Context, userID int64, limit int64) ([]*models.ReadingHistoryEntity, error) {
	params := sqlc.GetRecentReadingHistoryParams{
		UserID: userID,
		Limit:  limit,
	}
	rows, err := r.queries.GetRecentReadingHistory(ctx, params)
	if err != nil {
		return nil, err
	}
	return (&models.ReadingHistoryEntities{}).FromSqlc(rows), nil
}

func (r *featureRepository) GetReadingProgress(ctx context.Context, userID int64, bookID string) (*models.ReadingProgressEntity, error) {
	row, err := r.queries.GetReadingProgress(ctx, sqlc.GetReadingProgressParams{
		UserID: userID,
		BookID: bookID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return (&models.ReadingProgressEntity{}).FromSqlc(row), nil
}

func (r *featureRepository) UpsertReadingProgress(ctx context.Context, progress *models.ReadingProgressEntity) (*models.ReadingProgressEntity, error) {
	if progress == nil {
		return nil, fmt.Errorf("reading progress is nil")
	}
	params := sqlc.UpsertReadingProgressParams{
		UserID:             progress.UserID,
		BookID:             progress.BookID,
		FileID:             stringPtrToNullString(progress.FileID),
		ChapterRef:         progress.ChapterID,
		ChapterTitle:       progress.ChapterTitle,
		ChapterIndex:       progress.ChapterIndex,
		ProgressPercent:    floatPtrToNullFloat64(progress.ProgressPercent),
		OpenedCount:        progress.OpenedCount,
		QualifiedReadCount: progress.QualifiedReadCount,
		LastCountedAt:      timePtrToNullTime(progress.LastCountedAt),
	}
	row, err := r.queries.UpsertReadingProgress(ctx, params)
	if err != nil {
		return nil, err
	}
	return (&models.ReadingProgressEntity{}).FromSqlc(row), nil
}

func (r *featureRepository) UpsertBookReadStats(ctx context.Context, bookID string, openDelta int64, qualifiedDelta int64, lastCountedAt *time.Time) error {
	if err := r.queries.UpsertBookReadStats(ctx, sqlc.UpsertBookReadStatsParams{
		BookID:             bookID,
		TotalOpenCount:     openDelta,
		QualifiedReadCount: qualifiedDelta,
		LastCountedAt:      timePtrToNullTime(lastCountedAt),
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("feature:read_stats:%s", bookID))
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *featureRepository) GetBookReadStats(ctx context.Context, bookID string) (*models.BookReadStatsEntity, error) {
	key := fmt.Sprintf("feature:read_stats:%s", bookID)
	if r.c != nil {
		var stats models.BookReadStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}
	row, err := r.queries.GetBookReadStats(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.BookReadStatsEntity{BookID: bookID}, nil
		}
		return nil, err
	}
	result := (&models.BookReadStatsEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) UpsertBookDownloadStats(ctx context.Context, bookID string, downloadDelta int64) error {
	if err := r.queries.UpsertBookDownloadStats(ctx, sqlc.UpsertBookDownloadStatsParams{
		BookID:             bookID,
		TotalDownloadCount: downloadDelta,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("feature:download_stats:%s", bookID))
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *featureRepository) GetBookDownloadStats(ctx context.Context, bookID string) (*models.BookDownloadStatsEntity, error) {
	key := fmt.Sprintf("feature:download_stats:%s", bookID)
	if r.c != nil {
		var stats models.BookDownloadStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}
	row, err := r.queries.GetBookDownloadStats(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.BookDownloadStatsEntity{BookID: bookID}, nil
		}
		return nil, err
	}
	result := (&models.BookDownloadStatsEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) GetBookmark(ctx context.Context, userID int64, bookID string) (*models.BookmarkEntity, error) {
	key := fmt.Sprintf("bookmark:user:%d:book:%s", userID, bookID)
	if r.c != nil {
		var bookmark models.BookmarkEntity
		if err := r.c.Get(ctx, key, &bookmark); err == nil {
			return &bookmark, nil
		}
	}
	row, err := r.queries.GetBookmark(ctx, sqlc.GetBookmarkParams{UserID: userID, BookID: bookID})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	result := (&models.BookmarkEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) SetBookmark(ctx context.Context, userID int64, bookID string, bookmarked bool) (*models.BookmarkEntity, error) {
	key := fmt.Sprintf("bookmark:user:%d:book:%s", userID, bookID)
	if !bookmarked {
		if err := r.queries.DeleteBookmark(ctx, sqlc.DeleteBookmarkParams{UserID: userID, BookID: bookID}); err != nil {
			return nil, err
		}
		if r.c != nil {
			_ = r.c.Del(ctx, key)
			_ = r.c.DelByPattern(context.Background(), fmt.Sprintf("bookmark:user:%d:ids*", userID))
		}
		if err := r.refreshBookBookmarkStats(ctx, bookID); err != nil {
			return nil, err
		}
		return nil, nil
	}
	row, err := r.queries.UpsertBookmark(ctx, sqlc.UpsertBookmarkParams{UserID: userID, BookID: bookID})
	if err != nil {
		return nil, err
	}
	result := (&models.BookmarkEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		_ = r.c.DelByPattern(context.Background(), fmt.Sprintf("bookmark:user:%d:ids*", userID))
	}
	if err := r.refreshBookBookmarkStats(ctx, bookID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *featureRepository) GetBookmarkedBookIDs(ctx context.Context, userID int64, limit, offset int64) ([]string, error) {
	params := sqlc.GetBookmarkedBookIDsParams{UserID: userID, Limit: limit, Offset: offset}
	key := cache.QueryKey(fmt.Sprintf("bookmark:user:%d:ids", userID), params)
	if r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return ids, nil
		}
	}
	ids, err := r.queries.GetBookmarkedBookIDs(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
	}
	return ids, nil
}

func (r *featureRepository) UpsertBookReview(ctx context.Context, userID int64, bookID string, rating int64, review *string) (*models.BookReviewEntity, error) {
	row, err := r.queries.UpsertBookReview(ctx, sqlc.UpsertBookReviewParams{
		UserID: userID,
		BookID: bookID,
		Rating: rating,
		Review: stringPtrToNullString(review),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.BookReviewEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			fmt.Sprintf("review:user:%d:book:%s", userID, bookID),
			fmt.Sprintf("rating:summary:%s", bookID),
			fmt.Sprintf("social:stats:%s", bookID),
		)
		_ = r.c.DelByPattern(context.Background(), fmt.Sprintf("review:book:%s*", bookID))
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	if err := r.refreshBookRatingStats(ctx, bookID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *featureRepository) DeleteBookReview(ctx context.Context, userID int64, bookID string) error {
	if err := r.queries.DeleteBookReview(ctx, sqlc.DeleteBookReviewParams{UserID: userID, BookID: bookID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			fmt.Sprintf("review:user:%d:book:%s", userID, bookID),
			fmt.Sprintf("rating:summary:%s", bookID),
			fmt.Sprintf("social:stats:%s", bookID),
		)
		_ = r.c.DelByPattern(context.Background(), fmt.Sprintf("review:book:%s*", bookID))
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	if err := r.refreshBookRatingStats(ctx, bookID); err != nil {
		return err
	}
	return nil
}

func (r *featureRepository) GetBookReview(ctx context.Context, userID int64, bookID string) (*models.BookReviewEntity, error) {
	key := fmt.Sprintf("review:user:%d:book:%s", userID, bookID)
	if r.c != nil {
		var review models.BookReviewEntity
		if err := r.c.Get(ctx, key, &review); err == nil {
			return &review, nil
		}
	}
	row, err := r.queries.GetBookReview(ctx, sqlc.GetBookReviewParams{UserID: userID, BookID: bookID})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	result := (&models.BookReviewEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) GetBookSocialStats(ctx context.Context, bookID string) (*models.BookSocialStatsEntity, error) {
	key := fmt.Sprintf("social:stats:%s", bookID)
	if r.c != nil {
		var stats models.BookSocialStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}
	row, err := r.queries.GetBookSocialStats(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.BookSocialStatsEntity{BookID: bookID}, nil
		}
		return nil, err
	}
	result := (&models.BookSocialStatsEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) CreateBookShareEvent(ctx context.Context, bookID string, actorKey string, windowBucket int64) (bool, error) {
	_, err := r.queries.CreateBookShareEvent(ctx, sqlc.CreateBookShareEventParams{
		BookID:       bookID,
		ActorKey:     actorKey,
		WindowBucket: windowBucket,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *featureRepository) UpsertBookShareStats(ctx context.Context, bookID string, shareDelta int64) error {
	if err := r.queries.UpsertBookShareStats(ctx, sqlc.UpsertBookShareStatsParams{
		BookID:     bookID,
		BookID_2:   bookID,
		BookID_3:   bookID,
		BookID_4:   bookID,
		ShareCount: shareDelta,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("social:stats:%s", bookID))
	}
	return nil
}

func (r *featureRepository) refreshBookBookmarkStats(ctx context.Context, bookID string) error {
	if err := r.queries.RefreshBookBookmarkStats(ctx, sqlc.RefreshBookBookmarkStatsParams{
		BookID:   bookID,
		BookID_2: bookID,
		BookID_3: bookID,
		BookID_4: bookID,
		BookID_5: bookID,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("social:stats:%s", bookID))
	}
	return nil
}

func (r *featureRepository) refreshBookRatingStats(ctx context.Context, bookID string) error {
	if err := r.queries.RefreshBookRatingStats(ctx, sqlc.RefreshBookRatingStatsParams{
		BookID:   bookID,
		BookID_2: bookID,
		BookID_3: bookID,
		BookID_4: bookID,
		BookID_5: bookID,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("rating:summary:%s", bookID), fmt.Sprintf("social:stats:%s", bookID))
	}
	return nil
}

func (r *featureRepository) ListBookReviews(ctx context.Context, bookID string, limit, offset int64) ([]*models.BookReviewEntity, error) {
	params := sqlc.ListBookReviewsParams{BookID: bookID, Limit: limit, Offset: offset}
	key := cache.QueryKey(fmt.Sprintf("review:book:%s", bookID), params)
	if r.c != nil {
		var reviews []*models.BookReviewEntity
		if err := r.c.Get(ctx, key, &reviews); err == nil {
			return reviews, nil
		}
	}
	rows, err := r.queries.ListBookReviews(ctx, params)
	if err != nil {
		return nil, err
	}
	result := (&models.BookReviewEntities{}).FromSqlc(rows)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

// ListAllReviews returns all reviews across all books with user and book info, ordered by most recent first.
func (r *featureRepository) ListAllReviews(ctx context.Context, limit, offset int64) ([]*models.BookReviewEntity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT br.user_id, br.book_id, br.rating, br.review, br.created_at, br.updated_at,
		       u.full_name as user_name, u.email as user_email,
		       b.title as book_title
		FROM book_reviews br
		JOIN users u ON u.id = br.user_id
		JOIN books b ON b.id = br.book_id
		ORDER BY br.updated_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*models.BookReviewEntity
	for rows.Next() {
		var entity models.BookReviewEntity
		var userName, userEmail, bookTitle string
		if err := rows.Scan(
			&entity.UserID, &entity.BookID, &entity.Rating, &entity.Review,
			&entity.CreatedAt, &entity.UpdatedAt,
			&userName, &userEmail, &bookTitle,
		); err != nil {
			return nil, err
		}
		entity.UserName = userName
		entity.UserEmail = userEmail
		entity.BookTitle = bookTitle
		reviews = append(reviews, &entity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *featureRepository) GetBookRatingSummary(ctx context.Context, bookID string) (*models.BookRatingSummaryEntity, error) {
	key := fmt.Sprintf("rating:summary:%s", bookID)
	if r.c != nil {
		var summary models.BookRatingSummaryEntity
		if err := r.c.Get(ctx, key, &summary); err == nil {
			return &summary, nil
		}
	}
	row, err := r.queries.GetBookRatingSummary(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.BookRatingSummaryEntity{BookID: bookID}, nil
		}
		return nil, err
	}
	result := (&models.BookRatingSummaryEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) getCollectionsByIDs(ctx context.Context, ids []string) ([]*models.CollectionEntity, error) {
	if len(ids) == 0 {
		return []*models.CollectionEntity{}, nil
	}
	if r.c == nil {
		return r.fetchCollectionsByIDs(ctx, ids)
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("collection:id:%s", id)
	}

	collectionsByID := make(map[string]*models.CollectionEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))
	raws := r.c.MGet(ctx, keys...)
	for i, raw := range raws {
		if len(raw) > 0 {
			var collection models.CollectionEntity
			if err := jsonx.Unmarshal(raw, &collection); err == nil {
				collectionsByID[collection.ID] = &collection
				continue
			}
		}
		missingIDs = append(missingIDs, ids[i])
		missingKeys = append(missingKeys, keys[i])
	}

	if len(missingIDs) > 0 {
		rows, err := r.queries.GetCollectionsByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[string]*models.CollectionEntity, len(rows))
		for _, row := range rows {
			collection := (&models.CollectionEntity{}).FromSqlc(row)
			collectionsByID[collection.ID] = collection
			missingMap[collection.ID] = collection
		}
		toCache := make(map[string]any, len(missingMap))
		for i, missingID := range missingIDs {
			if collection, ok := missingMap[missingID]; ok {
				toCache[missingKeys[i]] = collection
			}
		}
		if len(toCache) > 0 {
			_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
		}
	}

	return orderCollections(ids, collectionsByID), nil
}

func (r *featureRepository) fetchCollectionsByIDs(ctx context.Context, ids []string) ([]*models.CollectionEntity, error) {
	rows, err := r.queries.GetCollectionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	collectionsByID := make(map[string]*models.CollectionEntity, len(rows))
	for _, row := range rows {
		collection := (&models.CollectionEntity{}).FromSqlc(row)
		collectionsByID[collection.ID] = collection
	}
	return orderCollections(ids, collectionsByID), nil
}

func orderCollections(ids []string, collectionsByID map[string]*models.CollectionEntity) []*models.CollectionEntity {
	ordered := make([]*models.CollectionEntity, 0, len(ids))
	for _, id := range ids {
		if collection, ok := collectionsByID[id]; ok {
			ordered = append(ordered, collection)
		}
	}
	return ordered
}

func stringPtrToNullString(value *string) sql.NullString {
	if value == nil || *value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func floatPtrToNullFloat64(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func timePtrToNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func (r *featureRepository) AddBookToCollection(ctx context.Context, collectionID string, bookID string) error {
	return r.queries.AddBookToCollection(ctx, sqlc.AddBookToCollectionParams{
		CollectionID: collectionID,
		BookID:       bookID,
	})
}

func (r *featureRepository) RemoveBookFromCollection(ctx context.Context, collectionID string, bookID string) error {
	return r.queries.RemoveBookFromCollection(ctx, sqlc.RemoveBookFromCollectionParams{
		CollectionID: collectionID,
		BookID:       bookID,
	})
}

func (r *featureRepository) GetBookCollectionIDs(ctx context.Context, userID int64, bookID string) ([]string, error) {
	return r.queries.GetBookCollectionIDs(ctx, sqlc.GetBookCollectionIDsParams{
		UserID: userID,
		BookID: bookID,
	})
}

