package repositories

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type AudiobookRepository interface {
	ListChapters(ctx context.Context, bookID string) ([]*models.AudiobookChapterEntity, error)
	ListBooksWithAudio(ctx context.Context, cursorTime *time.Time, cursorID string, limit int64) ([]string, error)
	UpsertChapter(ctx context.Context, id string, bookID string, fileID *string, chapterIndex int64, title string, startSec float64, endSec *float64) (*models.AudiobookChapterEntity, error)
	DeleteChapter(ctx context.Context, id string) error
	DeleteChaptersForBook(ctx context.Context, bookID string) error
	WithTx(tx *sql.Tx) AudiobookRepository
}

type audiobookRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewAudiobookRepository(db *sql.DB, c cache.Cache) AudiobookRepository {
	return &audiobookRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

func (r *audiobookRepository) WithTx(tx *sql.Tx) AudiobookRepository {
	if tx == nil {
		return r
	}
	return &audiobookRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
		c:       r.c,
		inTx:    true,
		sfg:     &singleflight.Group{},
	}
}

func (r *audiobookRepository) ListChapters(ctx context.Context, bookID string) ([]*models.AudiobookChapterEntity, error) {
	key := cache.BuildKey("audiobook_chapters", "book", bookID)
	if r.c != nil && !r.inTx {
		var cached models.AudiobookChapterEntities
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.ListAudiobookChapters(ctx, bookID)
		if err != nil {
			return nil, err
		}
		result := (&models.AudiobookChapterEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AudiobookChapterEntity), nil
}

func (r *audiobookRepository) ListBooksWithAudio(ctx context.Context, cursorTime *time.Time, cursorID string, limit int64) ([]string, error) {
	cacheKey := cache.BuildKey("audiobook_book_ids", cursorTime, cursorID, limit)
	if r.c != nil && !r.inTx {
		var cachedIDs []string
		if err := r.c.Get(ctx, cacheKey, &cachedIDs); err == nil {
			return cachedIDs, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		ids, err := r.queries.ListBooksWithAudioChapters(ctx, sqlc.ListBooksWithAudioChaptersParams{
			CursorTime: cursorTimeArg(cursorTime),
			CursorID:   convert.StrPtrToNullString(&cursorID),
			Limit:      limit,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *audiobookRepository) UpsertChapter(ctx context.Context, id string, bookID string, fileID *string, chapterIndex int64, title string, startSec float64, endSec *float64) (*models.AudiobookChapterEntity, error) {
	row, err := r.queries.UpsertAudiobookChapter(ctx, sqlc.UpsertAudiobookChapterParams{
		ID:           id,
		BookID:       bookID,
		FileID:       convert.StrPtrToNullString(fileID),
		ChapterIndex: chapterIndex,
		Title:        title,
		StartSec:     startSec,
		EndSec:       convert.FloatPtrToNullFloat64(endSec),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.AudiobookChapterEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(ctx, "audiobook_book_ids*")
		_ = r.c.Del(ctx, cache.BuildKey("audiobook_chapters", "book", bookID))
	}
	return result, nil
}

// ponytail: DeleteChapter nukes the whole chapters cache because the row delete
// doesn't return book_id; per-book invalidation if cache pressure ever matters.
func (r *audiobookRepository) DeleteChapter(ctx context.Context, id string) error {
	if err := r.queries.DeleteAudiobookChapter(ctx, id); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(ctx, "audiobook_chapters:book:*")
		_ = r.c.DelByPattern(ctx, "audiobook_book_ids*")
	}
	return nil
}

func (r *audiobookRepository) DeleteChaptersForBook(ctx context.Context, bookID string) error {
	if err := r.queries.DeleteAudiobookChaptersForBook(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(ctx, "audiobook_book_ids*")
		_ = r.c.Del(ctx, cache.BuildKey("audiobook_chapters", "book", bookID))
	}
	return nil
}