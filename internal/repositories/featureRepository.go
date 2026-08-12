package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type FeatureRepository interface {
	GetLibraryStats(ctx context.Context) (*models.LibraryStatsEntity, error)
	CreateCollection(ctx context.Context, id, name string, userID string) (*models.CollectionEntity, error)
	UpdateCollection(ctx context.Context, id, name string, userID string) (*models.CollectionEntity, error)
	DeleteCollection(ctx context.Context, id string, userID string) error
	CollectionOwnedByUser(ctx context.Context, collectionID, userID string) (bool, error)
	GetUserCollections(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*models.CollectionEntity, error)
	GetCollectionsByIDs(ctx context.Context, ids []string) ([]*models.CollectionEntity, error)
	GetRecentReadingHistory(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*models.ReadingHistoryEntity, error)
	GetReadingProgress(ctx context.Context, userID string, bookID string) (*models.ReadingProgressEntity, error)
	UpsertReadingProgress(ctx context.Context, progress *models.ReadingProgressEntity) (*models.ReadingProgressEntity, error)
	UpsertBookReadStats(ctx context.Context, bookID string, openDelta int64, qualifiedDelta int64, lastCountedAt *time.Time) error
	GetBookReadStats(ctx context.Context, bookID string) (*models.BookReadStatsEntity, error)
	UpsertBookDownloadStats(ctx context.Context, bookID string, downloadDelta int64) error
	GetBookDownloadStats(ctx context.Context, bookID string) (*models.BookDownloadStatsEntity, error)
	GetBookmark(ctx context.Context, userID string, bookID string) (*models.BookmarkEntity, error)
	SetBookmark(ctx context.Context, userID string, bookID string, bookmarked bool) (*models.BookmarkEntity, error)
	GetBookmarkedBooks(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]BookmarkedBookPage, error)
	UpsertBookReview(ctx context.Context, userID string, bookID string, rating int64, review *string) (*models.BookReviewEntity, error)
	DeleteBookReview(ctx context.Context, userID string, bookID string) error
	GetBookReview(ctx context.Context, userID string, bookID string) (*models.BookReviewEntity, error)
	ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, cursorID string, limit int64) ([]*models.BookReviewEntity, error)
	ListAllReviews(ctx context.Context, limit, offset int64) ([]*models.BookReviewEntity, error)
	GetBookRatingSummary(ctx context.Context, bookID string) (*models.BookRatingSummaryEntity, error)
	GetBookSocialStats(ctx context.Context, bookID string) (*models.BookSocialStatsEntity, error)
	CreateBookShareEvent(ctx context.Context, bookID string, actorKey string, windowBucket int64) (bool, error)
	UpsertBookShareStats(ctx context.Context, bookID string, shareDelta int64) error
	AddBookToCollection(ctx context.Context, collectionID string, bookID string) error
	RemoveBookFromCollection(ctx context.Context, collectionID string, bookID string) error
	GetBookCollectionIDs(ctx context.Context, userID string, bookID string) ([]string, error)
	UpsertReadingSession(ctx context.Context, arg sqlc.UpsertReadingSessionParams) (*models.ReadingSessionEntity, error)
	GetReadingHeatmap(ctx context.Context, userID string) ([]*models.ReadingHeatmapEntity, error)
	GetReadingGoal(ctx context.Context, userID string) (*models.ReadingGoalEntity, error)
	UpsertReadingGoal(ctx context.Context, userID string, wordsPerDay int64, booksPerYear int64) (*models.ReadingGoalEntity, error)
	GetReadingStatsByBook(ctx context.Context, userID string, bookID string) (*models.ReadingStatsByBookEntity, error)
	GetReadingStatsSince(ctx context.Context, userID string, since time.Time) (*models.ReadingStatsSinceEntity, error)
	GetListeningHistory(ctx context.Context, userID string) ([]*models.ListeningHistoryEntity, error)
	GetListeningStats(ctx context.Context, userID string) (*models.ListeningStatsEntity, error)
	GetLibraryBreakdown(ctx context.Context) (*models.LibraryBreakdownEntity, error)
	ListSmartCollections(ctx context.Context, userID string) ([]*models.SmartCollectionEntity, error)
	GetSmartCollection(ctx context.Context, id string, userID string) (*models.SmartCollectionEntity, error)
	CreateSmartCollection(ctx context.Context, id string, userID string, name string, ruleJson string) (*models.SmartCollectionEntity, error)
	UpdateSmartCollection(ctx context.Context, id string, userID string, name string, ruleJson string) (*models.SmartCollectionEntity, error)
	DeleteSmartCollection(ctx context.Context, id string, userID string) error
	ListSmartFilters(ctx context.Context, userID string) ([]*models.SmartFilterEntity, error)
	GetSmartFilter(ctx context.Context, id string, userID string) (*models.SmartFilterEntity, error)
	GetSmartFiltersByIDs(ctx context.Context, ids []string) ([]*models.SmartFilterEntity, error)
	CreateSmartFilter(ctx context.Context, id string, userID string, name string, rulesJson string, isPinnedSidebar bool, isPinnedHome bool, homePosition int64) (*models.SmartFilterEntity, error)
	UpdateSmartFilter(ctx context.Context, id string, userID string, name string, rulesJson string, isPinnedSidebar bool, isPinnedHome bool) (*models.SmartFilterEntity, error)
	DeleteSmartFilter(ctx context.Context, id string, userID string) error
	UpdateSmartFilterPinSidebar(ctx context.Context, id string, userID string, isPinned bool) (*models.SmartFilterEntity, error)
	UpdateSmartFilterPinHome(ctx context.Context, id string, userID string, isPinned bool) (*models.SmartFilterEntity, error)
	UpdateSmartFilterHomePosition(ctx context.Context, id string, userID string, position int64) (*models.SmartFilterEntity, error)
	WithTx(tx *sql.Tx) FeatureRepository
}

type featureRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewFeatureRepository(db *sql.DB, c cache.Cache) FeatureRepository {
	return &featureRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

// The transactional copy gets its own singleflight group and writes nothing to the cache
// (inTx guards every Set). Sharing the parent's group let a plain reader join a call already
// in flight inside the transaction and receive a row that was not committed yet — which then
// vanished on rollback while the cache kept serving it. Both halves are needed: gating the
// writes alone still leaks through the shared group, and vice versa.
func (r *featureRepository) WithTx(tx *sql.Tx) FeatureRepository {
	if tx == nil {
		return r
	}
	return &featureRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
		c:       r.c,
		inTx:    true,
		sfg:     &singleflight.Group{},
	}
}

func (r *featureRepository) GetLibraryStats(ctx context.Context) (*models.LibraryStatsEntity, error) {
	key := constants.CacheKeyLibraryStats
	if r.c != nil && !r.inTx {
		var stats models.LibraryStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		stats, err := r.queries.GetLibraryStats(ctx)
		if err != nil {
			return nil, err
		}
		result := (&models.LibraryStatsEntity{}).FromSqlc(stats)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.LibraryStatsEntity), nil
}

func (r *featureRepository) CreateCollection(ctx context.Context, id, name string, userID string) (*models.CollectionEntity, error) {
	collection, err := r.queries.CreateCollection(ctx, sqlc.CreateCollectionParams{
		ID:     id,
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	result := (&models.CollectionEntity{}).FromSqlc(collection)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("collection", "user", userID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyCollectionOwnedPattern)
		_ = r.c.Set(ctx, cache.BuildKey("collection", "id", result.ID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) UpdateCollection(ctx context.Context, id, name string, userID string) (*models.CollectionEntity, error) {
	collection, err := r.queries.UpdateCollection(ctx, sqlc.UpdateCollectionParams{
		ID:     id,
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	result := (&models.CollectionEntity{}).FromSqlc(collection)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("collection", "user", userID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyCollectionOwnedPattern)
		_ = r.c.Set(ctx, cache.BuildKey("collection", "id", result.ID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) DeleteCollection(ctx context.Context, id string, userID string) error {
	err := r.queries.DeleteCollection(ctx, sqlc.DeleteCollectionParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("collection", "user", userID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyCollectionOwnedPattern)
		_ = r.c.Del(ctx, cache.BuildKey("collection", "id", id))
	}
	return nil
}

func (r *featureRepository) CollectionOwnedByUser(ctx context.Context, collectionID, userID string) (bool, error) {
	key := cache.BuildKey("collection", "owned", userID, collectionID)
	if r.c != nil && !r.inTx {
		var owned bool
		if err := r.c.Get(ctx, key, &owned); err == nil {
			return owned, nil
		}
	}

	value, err, _ := r.sfg.Do(key, func() (any, error) {
		owned, err := r.queries.CollectionOwnedByUser(ctx, sqlc.CollectionOwnedByUserParams{
			ID:     collectionID,
			UserID: userID,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, owned, constants.NormalCacheDuration)
		}
		return owned, nil
	})
	if err != nil {
		return false, err
	}
	return value.(bool), nil
}

func (r *featureRepository) GetUserCollections(ctx context.Context, userID string, cursorCreatedAt *time.Time, cursorID string, limit int64) ([]*models.CollectionEntity, error) {
	var key string
	if cursorCreatedAt == nil {
		key = cache.BuildKey("collection", "user", userID, "limit", limit)
	}

	if key != "" && r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetCollectionsByIDs(ctx, ids)
		}
	}

	sfKey := key
	if sfKey == "" {
		sfKey = fmt.Sprintf("collection:user:%s:cursor:%v:%s:limit:%d", userID, cursorCreatedAt, cursorID, limit)
	}

	v, err, _ := r.sfg.Do(sfKey, func() (any, error) {
		params := sqlc.GetUserCollectionIDsParams{
			UserID:          userID,
			CursorCreatedAt: cursorTimeArg(cursorCreatedAt),
			CursorID:        convert.StrPtrToNullString(&cursorID),
			Limit:           limit,
		}

		ids, err := r.queries.GetUserCollectionIDs(ctx, params)
		if err != nil {
			return nil, err
		}

		if key != "" && r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetCollectionsByIDs(ctx, v.([]string))
}

func (r *featureRepository) GetRecentReadingHistory(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*models.ReadingHistoryEntity, error) {
	params := sqlc.GetRecentReadingHistoryBookIDsParams{
		UserID:          userID,
		Limit:           limit,
		CursorUpdatedAt: cursorTimeArg(cursor),
		CursorID:        convert.StrPtrToNullString(&cursorID),
	}
	key := cache.QueryKeyParts(params, "feature", "reading_history", "user", userID)

	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getReadingHistoryByBookIDs(ctx, userID, ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		idRows, err := r.queries.GetRecentReadingHistoryBookIDs(ctx, params)
		if err != nil {
			return nil, err
		}

		if len(idRows) == 0 {
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, key, []string{}, constants.ListCacheDuration)
			}
			return []*models.ReadingHistoryEntity{}, nil
		}

		fullParams := sqlc.GetRecentReadingHistoryParams{
			UserID:          userID,
			Limit:           limit,
			CursorUpdatedAt: cursorTimeArg(cursor),
			CursorID:        convert.StrPtrToNullString(&cursorID),
		}
		rows, err := r.queries.GetRecentReadingHistory(ctx, fullParams)
		if err != nil {
			return nil, err
		}
		result := (&models.ReadingHistoryEntities{}).FromSqlc(rows)

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, idRows, constants.ListCacheDuration)
			r.cacheReadingHistoryEntities(ctx, userID, result)
		}

		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.ReadingHistoryEntity), nil
}

func (r *featureRepository) getReadingHistoryByBookIDs(ctx context.Context, userID string, bookIDs []string) ([]*models.ReadingHistoryEntity, bool) {
	if len(bookIDs) == 0 {
		return []*models.ReadingHistoryEntity{}, true
	}
	if r.c == nil {
		return nil, false
	}

	cacheKeys := make([]string, len(bookIDs))
	for i, id := range bookIDs {
		cacheKeys[i] = cache.BuildKey("reading_history_entity", "user", userID, "book", id)
	}

	cachedBytes := r.c.MGet(ctx, cacheKeys...)
	ordered := make([]*models.ReadingHistoryEntity, 0, len(bookIDs))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.ReadingHistoryEntity
		if err := jsonx.Unmarshal(bytes, &entity); err != nil {
			return nil, false
		}
		ordered = append(ordered, &entity)
	}

	return ordered, true
}

func (r *featureRepository) cacheReadingHistoryEntities(ctx context.Context, userID string, entities []*models.ReadingHistoryEntity) {
	if r.c == nil || len(entities) == 0 {
		return
	}
	toCache := make(map[string]any, len(entities))
	for _, entity := range entities {
		toCache[cache.BuildKey("reading_history_entity", "user", userID, "book", entity.BookID)] = entity
	}
	_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
}

func (r *featureRepository) GetReadingProgress(ctx context.Context, userID string, bookID string) (*models.ReadingProgressEntity, error) {
	key := cache.BuildKey("feature", "reading_progress", "user", userID, "book", bookID)
	if r.c != nil && !r.inTx {
		var progress models.ReadingProgressEntity
		if err := r.c.Get(ctx, key, &progress); err == nil {
			return &progress, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetReadingProgress(ctx, sqlc.GetReadingProgressParams{
			UserID: userID,
			BookID: bookID,
		})
		if err != nil {
			return nil, err
		}
		result := (&models.ReadingProgressEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.ReadingProgressEntity), nil
}

func (r *featureRepository) GetReadingStatsByBook(ctx context.Context, userID string, bookID string) (*models.ReadingStatsByBookEntity, error) {
	key := cache.BuildKey("feature", "reading_stats_by_book", "user", userID, "book", bookID)
	if r.c != nil && !r.inTx {
		var stats models.ReadingStatsByBookEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetReadingStatsByBook(ctx, sqlc.GetReadingStatsByBookParams{
			UserID: userID,
			BookID: bookID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		result := (&models.ReadingStatsByBookEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.ReadingStatsByBookEntity), nil
}

func (r *featureRepository) GetReadingStatsSince(ctx context.Context, userID string, since time.Time) (*models.ReadingStatsSinceEntity, error) {
	key := cache.BuildKey("feature", "reading_stats_since", "user", userID, "since", since.Format("2006-01-02"))
	if r.c != nil && !r.inTx {
		var stats models.ReadingStatsSinceEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetReadingStatsSince(ctx, sqlc.GetReadingStatsSinceParams{
			UserID:      userID,
			SessionDate: since,
		})
		if err != nil {
			return nil, err
		}
		result := (&models.ReadingStatsSinceEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.ReadingStatsSinceEntity), nil
}

func (r *featureRepository) GetListeningHistory(ctx context.Context, userID string) ([]*models.ListeningHistoryEntity, error) {
	key := cache.BuildKey("feature", "listening_history", "user", userID)
	if r.c != nil && !r.inTx {
		var history models.ListeningHistoryEntities
		if err := r.c.Get(ctx, key, &history); err == nil {
			return history, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.GetListeningHistory(ctx, userID)
		if err != nil {
			return nil, err
		}
		result := (&models.ListeningHistoryEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.ListeningHistoryEntity), nil
}

func (r *featureRepository) GetListeningStats(ctx context.Context, userID string) (*models.ListeningStatsEntity, error) {
	key := cache.BuildKey("feature", "listening_stats", "user", userID)
	if r.c != nil && !r.inTx {
		var stats models.ListeningStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetListeningStats(ctx, userID)
		if err != nil {
			return nil, err
		}
		result := (&models.ListeningStatsEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.ListeningStatsEntity), nil
}

func (r *featureRepository) GetLibraryBreakdown(ctx context.Context) (*models.LibraryBreakdownEntity, error) {
	key := cache.BuildKey("feature", "library_breakdown")
	if r.c != nil && !r.inTx {
		var breakdown models.LibraryBreakdownEntity
		if err := r.c.Get(ctx, key, &breakdown); err == nil {
			return &breakdown, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		result := &models.LibraryBreakdownEntity{}
		if formats, err := r.queries.StatsByFormat(ctx); err == nil {
			result.AddFormat(formats)
		}
		if tags, err := r.queries.StatsByTag(ctx); err == nil {
			result.AddTags(tags)
		}
		if authors, err := r.queries.StatsByAuthor(ctx); err == nil {
			result.AddAuthors(authors)
		}
		if publishers, err := r.queries.StatsByPublisher(ctx); err == nil {
			result.AddPublishers(publishers)
		}
		if languages, err := r.queries.StatsByLanguage(ctx); err == nil {
			result.AddLanguages(languages)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.LibraryBreakdownEntity), nil
}

func (r *featureRepository) UpsertReadingProgress(ctx context.Context, progress *models.ReadingProgressEntity) (*models.ReadingProgressEntity, error) {
	if progress == nil {
		return nil, fmt.Errorf("reading progress is nil")
	}
	progressPercent := float64(0)
	if progress.ProgressPercent != nil {
		progressPercent = *progress.ProgressPercent
	}
	params := sqlc.UpsertReadingProgressParams{
		UserID:             progress.UserID,
		BookID:             progress.BookID,
		FileID:             convert.StrPtrToNullStringNonEmpty(progress.FileID),
		ChapterRef:         progress.ChapterID,
		ChapterTitle:       progress.ChapterTitle,
		ChapterIndex:       progress.ChapterIndex,
		ProgressPercent:    progressPercent,
		LocationCfi:        convert.StrPtrToNullStringNonEmpty(progress.LocationCfi),
		LocationType:       convert.StrPtrToNullStringNonEmpty(progress.LocationType),
		OpenedCount:        progress.OpenedCount,
		QualifiedReadCount: progress.QualifiedReadCount,
		LastCountedAt:      convert.TimePtrToNullTime(progress.LastCountedAt),
	}
	row, err := r.queries.UpsertReadingProgress(ctx, params)
	if err != nil {
		return nil, err
	}
	result := (&models.ReadingProgressEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("feature", "reading_progress", "user", progress.UserID, "book", progress.BookID))
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("feature", "reading_history", "user", progress.UserID)+"*")
	}
	return result, nil
}

func (r *featureRepository) UpsertBookReadStats(ctx context.Context, bookID string, openDelta int64, qualifiedDelta int64, lastCountedAt *time.Time) error {
	var wasFirstOpen bool
	if r.c != nil {
		before, err := r.queries.GetBookReadStats(ctx, bookID)
		wasFirstOpen = err != nil || (before.TotalOpenCount == 0 && before.QualifiedReadCount == 0)
	}
	if err := r.queries.UpsertBookReadStats(ctx, sqlc.UpsertBookReadStatsParams{
		BookID:             bookID,
		TotalOpenCount:     openDelta,
		QualifiedReadCount: qualifiedDelta,
		LastCountedAt:      convert.TimePtrToNullTime(lastCountedAt),
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("feature", "read_stats", bookID))
		if wasFirstOpen {
			_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		}
	}
	return nil
}

func (r *featureRepository) GetBookReadStats(ctx context.Context, bookID string) (*models.BookReadStatsEntity, error) {
	key := cache.BuildKey("feature", "read_stats", bookID)
	if r.c != nil && !r.inTx {
		var stats models.BookReadStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetBookReadStats(ctx, bookID)
		if err != nil {
			if err == sql.ErrNoRows {
				return &models.BookReadStatsEntity{BookID: bookID}, nil
			}
			return nil, err
		}
		result := (&models.BookReadStatsEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookReadStatsEntity), nil
}

func (r *featureRepository) UpsertBookDownloadStats(ctx context.Context, bookID string, downloadDelta int64) error {
	if err := r.queries.UpsertBookDownloadStats(ctx, sqlc.UpsertBookDownloadStatsParams{
		BookID:             bookID,
		TotalDownloadCount: downloadDelta,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("feature", "download_stats", bookID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *featureRepository) GetBookDownloadStats(ctx context.Context, bookID string) (*models.BookDownloadStatsEntity, error) {
	key := cache.BuildKey("feature", "download_stats", bookID)
	if r.c != nil && !r.inTx {
		var stats models.BookDownloadStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetBookDownloadStats(ctx, bookID)
		if err != nil {
			if err == sql.ErrNoRows {
				return &models.BookDownloadStatsEntity{BookID: bookID}, nil
			}
			return nil, err
		}
		result := (&models.BookDownloadStatsEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookDownloadStatsEntity), nil
}

func (r *featureRepository) GetBookmark(ctx context.Context, userID string, bookID string) (*models.BookmarkEntity, error) {
	key := cache.BuildKey("bookmark", "user", userID, "book", bookID)
	if r.c != nil && !r.inTx {
		var bookmark models.BookmarkEntity
		if err := r.c.Get(ctx, key, &bookmark); err == nil {
			return &bookmark, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetBookmark(ctx, sqlc.GetBookmarkParams{UserID: userID, BookID: bookID})
		if err != nil {
			if err == sql.ErrNoRows {
				return (*models.BookmarkEntity)(nil), nil
			}
			return nil, err
		}
		result := (&models.BookmarkEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.BookmarkEntity), nil
}

func (r *featureRepository) SetBookmark(ctx context.Context, userID string, bookID string, bookmarked bool) (*models.BookmarkEntity, error) {
	key := cache.BuildKey("bookmark", "user", userID, "book", bookID)
	if !bookmarked {
		if err := r.queries.DeleteBookmark(ctx, sqlc.DeleteBookmarkParams{UserID: userID, BookID: bookID}); err != nil {
			return nil, err
		}
		if r.c != nil {
			_ = r.c.Del(ctx, key)
			_ = r.c.DelByPattern(context.Background(), cache.BuildKey("bookmark", "user", userID, "ids")+"*")
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
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("bookmark", "user", userID, "ids")+"*")
	}
	if err := r.refreshBookBookmarkStats(ctx, bookID); err != nil {
		return nil, err
	}
	return result, nil
}

type BookmarkedBookPage struct {
	BookID    string
	CreatedAt time.Time
}

func (r *featureRepository) GetBookmarkedBooks(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]BookmarkedBookPage, error) {
	params := sqlc.GetBookmarkedBooksParams{
		UserID:          userID,
		Limit:           limit,
		CursorCreatedAt: cursorTimeArg(cursor),
		CursorID:        convert.StrPtrToNullString(&cursorID),
	}
	key := cache.QueryKeyParts(params, "bookmark", "user", userID, "ids")
	if r.c != nil && !r.inTx {
		var rows []BookmarkedBookPage
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.GetBookmarkedBooks(ctx, params)
		if err != nil {
			return nil, err
		}
		result := make([]BookmarkedBookPage, len(rows))
		for i, row := range rows {
			result[i] = BookmarkedBookPage{BookID: row.BookID, CreatedAt: row.CreatedAt}
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]BookmarkedBookPage), nil
}

func (r *featureRepository) UpsertBookReview(ctx context.Context, userID string, bookID string, rating int64, review *string) (*models.BookReviewEntity, error) {
	row, err := r.queries.UpsertBookReview(ctx, sqlc.UpsertBookReviewParams{
		UserID: userID,
		BookID: bookID,
		Rating: rating,
		Review: convert.StrPtrToNullStringNonEmpty(review),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.BookReviewEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			cache.BuildKey("review", "user", userID, "book", bookID),
			cache.BuildKey("rating", "summary", bookID),
			cache.BuildKey("social", "stats", bookID),
		)
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("review", "book", bookID)+"*")
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyAllReviewsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	if err := r.refreshBookRatingStats(ctx, bookID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *featureRepository) DeleteBookReview(ctx context.Context, userID string, bookID string) error {
	if err := r.queries.DeleteBookReview(ctx, sqlc.DeleteBookReviewParams{UserID: userID, BookID: bookID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			cache.BuildKey("review", "user", userID, "book", bookID),
			cache.BuildKey("rating", "summary", bookID),
			cache.BuildKey("social", "stats", bookID),
		)
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("review", "book", bookID)+"*")
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyAllReviewsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	if err := r.refreshBookRatingStats(ctx, bookID); err != nil {
		return err
	}
	return nil
}

func (r *featureRepository) GetBookReview(ctx context.Context, userID string, bookID string) (*models.BookReviewEntity, error) {
	key := cache.BuildKey("review", "user", userID, "book", bookID)
	if r.c != nil && !r.inTx {
		var review models.BookReviewEntity
		if err := r.c.Get(ctx, key, &review); err == nil {
			return &review, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetBookReview(ctx, sqlc.GetBookReviewParams{UserID: userID, BookID: bookID})
		if err != nil {
			if err == sql.ErrNoRows {
				return (*models.BookReviewEntity)(nil), nil
			}
			return nil, err
		}
		result := (&models.BookReviewEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.BookReviewEntity), nil
}

func (r *featureRepository) GetBookSocialStats(ctx context.Context, bookID string) (*models.BookSocialStatsEntity, error) {
	key := cache.BuildKey("social", "stats", bookID)
	if r.c != nil && !r.inTx {
		var stats models.BookSocialStatsEntity
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetBookSocialStats(ctx, bookID)
		if err != nil {
			if err == sql.ErrNoRows {
				return &models.BookSocialStatsEntity{BookID: bookID}, nil
			}
			return nil, err
		}
		result := (&models.BookSocialStatsEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookSocialStatsEntity), nil
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
		_ = r.c.Del(ctx, cache.BuildKey("social", "stats", bookID))
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
		_ = r.c.Del(ctx, cache.BuildKey("social", "stats", bookID))
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
		_ = r.c.Del(ctx, cache.BuildKey("rating", "summary", bookID), cache.BuildKey("social", "stats", bookID))
	}
	return nil
}

func (r *featureRepository) ListBookReviews(ctx context.Context, bookID string, cursor *time.Time, cursorID string, limit int64) ([]*models.BookReviewEntity, error) {
	params := sqlc.ListBookReviewCompositeKeysParams{
		BookID:          bookID,
		Limit:           limit,
		CursorUpdatedAt: cursorTimeArg(cursor),
		CursorID:        convert.StrPtrToNullString(&cursorID),
	}
	key := cache.QueryKeyParts(params, "review", "book", bookID)

	if r.c != nil && !r.inTx {
		var compositeKeys []string
		if err := r.c.Get(ctx, key, &compositeKeys); err == nil {
			if result, ok := r.getBookReviewsByCompositeKeys(ctx, compositeKeys); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		keysRows, err := r.queries.ListBookReviewCompositeKeys(ctx, params)
		if err != nil {
			return nil, err
		}

		if len(keysRows) == 0 {
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, key, []string{}, constants.ListCacheDuration)
			}
			return []*models.BookReviewEntity{}, nil
		}

		fullParams := sqlc.ListBookReviewsParams{
			BookID:          bookID,
			Limit:           limit,
			CursorUpdatedAt: cursorTimeArg(cursor),
			CursorID:        convert.StrPtrToNullString(&cursorID),
		}
		rows, err := r.queries.ListBookReviews(ctx, fullParams)
		if err != nil {
			return nil, err
		}

		result := (&models.BookReviewEntities{}).FromSqlc(rows)

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, keysRows, constants.ListCacheDuration)
			for _, entity := range result {
				_ = r.c.Set(ctx, cache.BuildKey("review", "user", entity.UserID, "book", entity.BookID), entity, constants.NormalCacheDuration)
			}
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.BookReviewEntity), nil
}

func (r *featureRepository) getBookReviewsByCompositeKeys(ctx context.Context, keys []string) ([]*models.BookReviewEntity, bool) {
	if len(keys) == 0 {
		return []*models.BookReviewEntity{}, true
	}
	if r.c == nil {
		return nil, false
	}

	cacheKeys := make([]string, len(keys))
	for i, k := range keys {
		// Keys are "<userID>:<bookID>"; both are UUIDs, so they never contain a colon.
		userID, bookID, ok := strings.Cut(k, ":")
		if !ok || userID == "" || bookID == "" {
			return nil, false
		}
		cacheKeys[i] = cache.BuildKey("review", "user", userID, "book", bookID)
	}

	cachedBytes := r.c.MGet(ctx, cacheKeys...)
	ordered := make([]*models.BookReviewEntity, 0, len(keys))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.BookReviewEntity
		if err := jsonx.Unmarshal(bytes, &entity); err != nil {
			return nil, false
		}
		ordered = append(ordered, &entity)
	}

	return ordered, true
}

func (r *featureRepository) ListAllReviews(ctx context.Context, limit, offset int64) ([]*models.BookReviewEntity, error) {
	key := cache.BuildKey("feature", "all_reviews", "limit", limit, "offset", offset)
	if r.c != nil && !r.inTx {
		var reviews []*models.BookReviewEntity
		if err := r.c.Get(ctx, key, &reviews); err == nil {
			return reviews, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.ListAllReviews(ctx, sqlc.ListAllReviewsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}

		reviews := (&models.BookReviewEntities{}).FromListAllReviewsSqlc(rows)

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, reviews, constants.ListCacheDuration)
		}
		return reviews, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.BookReviewEntity), nil
}

func (r *featureRepository) GetBookRatingSummary(ctx context.Context, bookID string) (*models.BookRatingSummaryEntity, error) {
	key := cache.BuildKey("rating", "summary", bookID)
	if r.c != nil && !r.inTx {
		var summary models.BookRatingSummaryEntity
		if err := r.c.Get(ctx, key, &summary); err == nil {
			return &summary, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetBookRatingSummary(ctx, bookID)
		if err != nil {
			if err == sql.ErrNoRows {
				return &models.BookRatingSummaryEntity{BookID: bookID}, nil
			}
			return nil, err
		}
		result := (&models.BookRatingSummaryEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookRatingSummaryEntity), nil
}

func (r *featureRepository) GetCollectionsByIDs(ctx context.Context, ids []string) ([]*models.CollectionEntity, error) {
	if len(ids) == 0 {
		return []*models.CollectionEntity{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("collection", "id", id)
	}

	collectionsByID := make(map[string]*models.CollectionEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))

	if r.c != nil && !r.inTx {
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
	} else {
		missingIDs = ids
		missingKeys = keys
	}

	if len(missingIDs) > 0 {
		rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.Collection, error) {
			return r.queries.GetCollectionsByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}
		missingMap := make(map[string]*models.CollectionEntity, len(rows))
		for _, row := range rows {
			collection := (&models.CollectionEntity{}).FromSqlc(row)
			collectionsByID[collection.ID] = collection
			missingMap[collection.ID] = collection
		}

		if r.c != nil {
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
	}
	ordered := make([]*models.CollectionEntity, 0, len(ids))
	for _, id := range ids {
		if collection, ok := collectionsByID[id]; ok {
			ordered = append(ordered, collection)
		}
	}
	return ordered, nil
}

func (r *featureRepository) AddBookToCollection(ctx context.Context, collectionID string, bookID string) error {
	if err := r.queries.AddBookToCollection(ctx, sqlc.AddBookToCollectionParams{
		CollectionID: collectionID,
		BookID:       bookID,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("feature", "book_collections", "user", "")+"*")
	}
	return nil
}

func (r *featureRepository) RemoveBookFromCollection(ctx context.Context, collectionID string, bookID string) error {
	if err := r.queries.RemoveBookFromCollection(ctx, sqlc.RemoveBookFromCollectionParams{
		CollectionID: collectionID,
		BookID:       bookID,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), cache.BuildKey("feature", "book_collections", "user", "")+"*")
	}
	return nil
}

func (r *featureRepository) GetBookCollectionIDs(ctx context.Context, userID string, bookID string) ([]string, error) {
	key := cache.BuildKey("feature", "book_collections", "user", userID, "book", bookID)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return ids, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.queries.GetBookCollectionIDs(ctx, sqlc.GetBookCollectionIDsParams{
			UserID: userID,
			BookID: bookID,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *featureRepository) UpsertReadingSession(ctx context.Context, arg sqlc.UpsertReadingSessionParams) (*models.ReadingSessionEntity, error) {
	res, err := r.queries.UpsertReadingSession(ctx, arg)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		r.c.Del(ctx, cache.BuildKey("feature", "reading_heatmap", arg.UserID))
	}
	entity := &models.ReadingSessionEntity{}
	return entity.FromSqlc(res), nil
}

func (r *featureRepository) GetReadingHeatmap(ctx context.Context, userID string) ([]*models.ReadingHeatmapEntity, error) {
	key := cache.BuildKey("feature", "reading_heatmap", userID)
	if r.c != nil {
		var cached []*models.ReadingHeatmapEntity
		err := r.c.GetOrFetch(ctx, key, &cached, 60*time.Minute, func() (any, error) {
			res, err, _ := r.sfg.Do(key, func() (any, error) {
				dbRes, dbErr := r.queries.GetReadingHeatmap(ctx, userID)
				if dbErr != nil {
					return nil, dbErr
				}
				entities := models.ReadingHeatmapEntities{}
				return entities.FromSqlc(dbRes), nil
			})
			if err != nil {
				return nil, err
			}
			return res, nil
		})
		if err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetReadingHeatmap(ctx, userID)
		if err != nil {
			return nil, err
		}
		entities := models.ReadingHeatmapEntities{}
		return entities.FromSqlc(res), nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.ReadingHeatmapEntity), nil
}

func (r *featureRepository) GetReadingGoal(ctx context.Context, userID string) (*models.ReadingGoalEntity, error) {
	key := cache.BuildKey("reading_goal", "user", userID)
	if r.c != nil && !r.inTx {
		var goal models.ReadingGoalEntity
		if err := r.c.Get(ctx, key, &goal); err == nil {
			return &goal, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetUserReadingGoal(ctx, userID)
		if err != nil {
			return nil, err
		}
		result := (&models.ReadingGoalEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.ReadingGoalEntity), nil
}

func (r *featureRepository) UpsertReadingGoal(ctx context.Context, userID string, wordsPerDay int64, booksPerYear int64) (*models.ReadingGoalEntity, error) {
	row, err := r.queries.UpsertUserReadingGoal(ctx, sqlc.UpsertUserReadingGoalParams{
		UserID:             userID,
		TargetWordsPerDay:  wordsPerDay,
		TargetBooksPerYear: booksPerYear,
	})
	if err != nil {
		return nil, err
	}
	result := (&models.ReadingGoalEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("reading_goal", "user", userID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) ListSmartCollections(ctx context.Context, userID string) ([]*models.SmartCollectionEntity, error) {
	key := cache.BuildKey("smart_collection", "user", userID)
	if r.c != nil && !r.inTx {
		var cached []*models.SmartCollectionEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.ListSmartCollectionsByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		entities := models.SmartCollectionEntities{}
		result := entities.FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.SmartCollectionEntity), nil
}

func (r *featureRepository) GetSmartCollection(ctx context.Context, id string, userID string) (*models.SmartCollectionEntity, error) {
	key := cache.BuildKey("smart_collection", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.SmartCollectionEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetSmartCollection(ctx, sqlc.GetSmartCollectionParams{ID: id, UserID: userID})
		if err != nil {
			return nil, err
		}
		result := (&models.SmartCollectionEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.SmartCollectionEntity), nil
}

func (r *featureRepository) CreateSmartCollection(ctx context.Context, id string, userID string, name string, ruleJson string) (*models.SmartCollectionEntity, error) {
	row, err := r.queries.CreateSmartCollection(ctx, sqlc.CreateSmartCollectionParams{
		ID:       id,
		UserID:   userID,
		Name:     name,
		RuleJson: ruleJson,
	})
	if err != nil {
		return nil, err
	}
	result := (&models.SmartCollectionEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("smart_collection", "user", userID))
		_ = r.c.Set(ctx, cache.BuildKey("smart_collection", "id", result.ID), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *featureRepository) UpdateSmartCollection(ctx context.Context, id string, userID string, name string, ruleJson string) (*models.SmartCollectionEntity, error) {
	row, err := r.queries.UpdateSmartCollection(ctx, sqlc.UpdateSmartCollectionParams{
		ID:       id,
		UserID:   userID,
		Name:     name,
		RuleJson: ruleJson,
	})
	if err != nil {
		return nil, err
	}
	result := (&models.SmartCollectionEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("smart_collection", "user", userID), cache.BuildKey("smart_collection", "id", id))
	}
	return result, nil
}

func (r *featureRepository) DeleteSmartCollection(ctx context.Context, id string, userID string) error {
	if err := r.queries.DeleteSmartCollection(ctx, sqlc.DeleteSmartCollectionParams{ID: id, UserID: userID}); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("smart_collection", "user", userID), cache.BuildKey("smart_collection", "id", id))
	}
	return nil
}

func (r *featureRepository) ListSmartFilters(ctx context.Context, userID string) ([]*models.SmartFilterEntity, error) {
	listKey := cache.BuildKey("smart_filter_list", userID)
	
	var ids []string
	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err == nil {
			return r.GetSmartFiltersByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(listKey, func() (any, error) {
		dbIDs, err := r.queries.ListSmartFilterIDsByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, listKey, dbIDs, constants.ListCacheDuration)
		}
		return dbIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetSmartFiltersByIDs(ctx, v.([]string))
}

func (r *featureRepository) GetSmartFilter(ctx context.Context, id string, userID string) (*models.SmartFilterEntity, error) {
	entities, err := r.GetSmartFiltersByIDs(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, sql.ErrNoRows
	}
	entity := entities[0]
	if entity.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return entity, nil
}

func (r *featureRepository) GetSmartFiltersByIDs(ctx context.Context, ids []string) ([]*models.SmartFilterEntity, error) {
	if len(ids) == 0 {
		return []*models.SmartFilterEntity{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("smart_filter", id)
	}

	resultMap := make(map[string]*models.SmartFilterEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var entity models.SmartFilterEntity
				if err := jsonx.Unmarshal(bytes, &entity); err == nil {
					resultMap[ids[i]] = &entity
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
		}
	} else {
		missingIDs = ids
	}

	if len(missingIDs) > 0 {
		sfgKey := "smart_filters:ids:" + strings.Join(missingIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.queries.GetSmartFiltersByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetchedMap := make(map[string]*models.SmartFilterEntity, len(rows))
			cachePairs := make(map[string]any, len(rows))
			for _, row := range rows {
				entity := (&models.SmartFilterEntity{}).FromSqlc(row)
				fetchedMap[row.ID] = entity
				cachePairs[cache.BuildKey("smart_filter", row.ID)] = entity
			}

			if r.c != nil && !r.inTx && len(cachePairs) > 0 {
				_ = r.c.MSet(ctx, cachePairs, 1*time.Hour)
			}
			return fetchedMap, nil
		})
		if err != nil {
			return nil, err
		}
		if fetched, ok := v.(map[string]*models.SmartFilterEntity); ok {
			for k, val := range fetched {
				resultMap[k] = val
			}
		}
	}

	out := make([]*models.SmartFilterEntity, 0, len(ids))
	for _, id := range ids {
		if entity, ok := resultMap[id]; ok && entity != nil {
			out = append(out, entity)
		}
	}
	return out, nil
}

func (r *featureRepository) CreateSmartFilter(ctx context.Context, id string, userID string, name string, rulesJson string, isPinnedSidebar bool, isPinnedHome bool, homePosition int64) (*models.SmartFilterEntity, error) {
	row, err := r.queries.CreateSmartFilter(ctx, sqlc.CreateSmartFilterParams{
		ID:              id,
		UserID:          userID,
		Name:            name,
		RulesJson:       rulesJson,
		IsPinnedSidebar: isPinnedSidebar,
		IsPinnedHome:    isPinnedHome,
		HomePosition:    homePosition,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter_list", userID))
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter", id))
	}
	return (&models.SmartFilterEntity{}).FromSqlc(row), nil
}

func (r *featureRepository) UpdateSmartFilter(ctx context.Context, id string, userID string, name string, rulesJson string, isPinnedSidebar bool, isPinnedHome bool) (*models.SmartFilterEntity, error) {
	row, err := r.queries.UpdateSmartFilter(ctx, sqlc.UpdateSmartFilterParams{
		ID:              id,
		UserID:          userID,
		Name:            name,
		RulesJson:       rulesJson,
		IsPinnedSidebar: isPinnedSidebar,
		IsPinnedHome:    isPinnedHome,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter_list", userID))
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter", id))
	}
	return (&models.SmartFilterEntity{}).FromSqlc(row), nil
}

func (r *featureRepository) DeleteSmartFilter(ctx context.Context, id string, userID string) error {
	if err := r.queries.DeleteSmartFilter(ctx, sqlc.DeleteSmartFilterParams{ID: id, UserID: userID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter_list", userID))
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter", id))
	}
	return nil
}

func (r *featureRepository) UpdateSmartFilterPinSidebar(ctx context.Context, id string, userID string, isPinned bool) (*models.SmartFilterEntity, error) {
	row, err := r.queries.UpdateSmartFilterPinSidebar(ctx, sqlc.UpdateSmartFilterPinSidebarParams{
		IsPinnedSidebar: isPinned,
		ID:              id,
		UserID:          userID,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter_list", userID))
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter", id))
	}
	return (&models.SmartFilterEntity{}).FromSqlc(row), nil
}

func (r *featureRepository) UpdateSmartFilterPinHome(ctx context.Context, id string, userID string, isPinned bool) (*models.SmartFilterEntity, error) {
	row, err := r.queries.UpdateSmartFilterPinHome(ctx, sqlc.UpdateSmartFilterPinHomeParams{
		IsPinnedHome: isPinned,
		ID:           id,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter_list", userID))
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter", id))
	}
	return (&models.SmartFilterEntity{}).FromSqlc(row), nil
}

func (r *featureRepository) UpdateSmartFilterHomePosition(ctx context.Context, id string, userID string, position int64) (*models.SmartFilterEntity, error) {
	row, err := r.queries.UpdateSmartFilterHomePosition(ctx, sqlc.UpdateSmartFilterHomePositionParams{
		HomePosition: position,
		ID:           id,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter_list", userID))
		_ = r.c.Del(ctx, cache.BuildKey("smart_filter", id))
	}
	return (&models.SmartFilterEntity{}).FromSqlc(row), nil
}
