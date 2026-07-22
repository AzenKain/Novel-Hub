package repositories

import (
	"context"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

func (r *bookDBRepository) SearchFTS(ctx context.Context, query string, limit, offset int64) ([]*models.FTSResultEntity, error) {
	params := sqlc.SearchFTSParams{
		Content: query,
		Limit:   limit,
		Offset:  offset,
	}
	key := cache.QueryKey("fts:search", params)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getFTSResultsByIDs(ctx, ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.SearchFTS(ctx, params)
		if err != nil {
			return nil, err
		}
		result := (&models.FTSResultEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			ids := make([]string, len(result))
			toCache := make(map[string]any, len(result))
			for i, res := range result {
				ids[i] = res.ChapterID
				toCache[cache.BuildKey("fts", "result", res.ChapterID)] = res
			}
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			if len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.FTSResultEntity), nil
}

func (r *bookDBRepository) getFTSResultsByIDs(ctx context.Context, ids []string) ([]*models.FTSResultEntity, bool) {
	if len(ids) == 0 {
		return []*models.FTSResultEntity{}, true
	}
	if r.c == nil {
		return nil, false
	}

	cacheKeys := make([]string, len(ids))
	for i, id := range ids {
		cacheKeys[i] = cache.BuildKey("fts", "result", id)
	}

	cachedBytes := r.c.MGet(ctx, cacheKeys...)
	ordered := make([]*models.FTSResultEntity, 0, len(ids))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.FTSResultEntity
		if err := jsonx.Unmarshal(bytes, &entity); err != nil {
			return nil, false
		}
		ordered = append(ordered, &entity)
	}

	return ordered, true
}

func (r *bookDBRepository) DeleteFTSBook(ctx context.Context, bookID string) error {
	if err := r.queries.DeleteFTSBook(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), "fts:*")
	}
	return nil
}

func (r *bookDBRepository) InsertFTSChapter(ctx context.Context, bookID, chapterID, title, content string) error {
	if err := r.queries.InsertFTSChapter(ctx, sqlc.InsertFTSChapterParams{
		BookID:    bookID,
		ChapterID: chapterID,
		Title:     title,
		Content:   content,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(context.Background(), cache.BuildKey("fts", "result", chapterID))
		_ = r.c.DelByPattern(context.Background(), "fts:search*")
	}
	return nil
}
