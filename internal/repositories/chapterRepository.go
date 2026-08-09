package repositories

import (
	"context"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

func (r *bookDBRepository) CreateChapter(ctx context.Context, chapter *models.ChapterEntity) error {
	params := sqlc.CreateChapterParams{
		ID:           chapter.ID,
		BookID:       chapter.BookID,
		Title:        chapter.Title,
		ContentPath:  convert.StrPtrToNullString(chapter.ContentPath),
		ChapterIndex: chapter.ChapterIndex,
	}
	res, err := r.queries.CreateChapter(ctx, params)
	if err != nil {
		return err
	}
	chapter.CreatedAt = res.CreatedAt.Time
	chapter.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("chapter", "id", chapter.ID), cache.BuildKey("chapter", "book", chapter.BookID))
		// SearchFTSInBook joins chapters for chapter_title/chapter_index.
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyFTSBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) GetChapter(ctx context.Context, id string) (*models.ChapterEntity, error) {
	key := cache.BuildKey("chapter", "id", id)
	if r.c != nil && !r.inTx {
		var chapter models.ChapterEntity
		if err := r.c.Get(ctx, key, &chapter); err == nil {
			return &chapter, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetChapter(ctx, id)
		if err != nil {
			return nil, err
		}
		chapter := (&models.ChapterEntity{}).FromSqlc(res)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, chapter, constants.NormalCacheDuration)
		}
		return chapter, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.ChapterEntity), nil
}

func (r *bookDBRepository) ListChaptersByBook(ctx context.Context, bookID string) ([]*models.ChapterEntity, error) {
	key := cache.BuildKey("chapter", "book", bookID)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetChaptersByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.queries.ListChapterIDsByBook(ctx, bookID)
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
	return r.GetChaptersByIDs(ctx, v.([]string))
}

func (r *bookDBRepository) GetChaptersByIDs(ctx context.Context, ids []string) ([]*models.ChapterEntity, error) {
	if len(ids) == 0 {
		return []*models.ChapterEntity{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("chapter", "id", id)
	}

	chaptersByID := make(map[string]*models.ChapterEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var chapter models.ChapterEntity
				if err := jsonx.Unmarshal(bytes, &chapter); err == nil {
					chaptersByID[chapter.ID] = &chapter
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
		rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.Chapter, error) {
			return r.queries.GetChaptersByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}
		missingMap := make(map[string]*models.ChapterEntity, len(rows))
		for _, row := range rows {
			chapter := (&models.ChapterEntity{}).FromSqlc(row)
			chaptersByID[chapter.ID] = chapter
			missingMap[chapter.ID] = chapter
		}

		if r.c != nil && !r.inTx {
			missingToCache := make(map[string]any, len(missingMap))
			for i, missingID := range missingIDs {
				if chapter, ok := missingMap[missingID]; ok {
					missingToCache[missingKeys[i]] = chapter
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	ordered := make([]*models.ChapterEntity, 0, len(ids))
	for _, id := range ids {
		if chapter, ok := chaptersByID[id]; ok {
			ordered = append(ordered, chapter)
		}
	}
	return ordered, nil
}

func (r *bookDBRepository) DeleteChapter(ctx context.Context, id string) error {
	// Read through the query, not GetChapter: the latter caches chapter:id:<id> on a miss,
	// populating a key for the row we are about to delete.
	chapter, preErr := r.queries.GetChapter(ctx, id)
	if err := r.queries.DeleteChapter(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("chapter", "id", id))
		if preErr == nil {
			_ = r.c.Del(ctx, cache.BuildKey("chapter", "book", chapter.BookID))
		} else {
			_ = r.c.DelByPattern(context.Background(), constants.CacheKeyChapterByBookPattern)
		}
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyFTSBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) DeleteChaptersByBook(ctx context.Context, bookID string) error {
	if err := r.queries.DeleteChaptersByBook(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("chapter", "book", bookID))
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyChapterByBookPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyFTSBookSearchPattern)
	}
	return nil
}
