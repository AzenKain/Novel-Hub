package repositories

import (
	"context"
	"database/sql"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type HighlightRepository interface {
	WithTx(tx *sql.Tx) HighlightRepository
	Create(ctx context.Context, arg sqlc.CreateHighlightParams) (*models.HighlightEntity, error)
	GetByChapter(ctx context.Context, userID int64, chapterID string) ([]*models.HighlightEntity, error)
	GetHighlightsByIDs(ctx context.Context, ids []string) ([]*models.HighlightEntity, error)
	Delete(ctx context.Context, arg sqlc.DeleteHighlightParams) error
	UpdateNote(ctx context.Context, arg sqlc.UpdateHighlightNoteParams) (*models.HighlightEntity, error)
}

type highlightRepository struct {
	queries *sqlc.Queries
	db      *sql.DB
	c       cache.Cache
	sfg     *singleflight.Group
}

func NewHighlightRepository(db *sql.DB, c cache.Cache) HighlightRepository {
	return &highlightRepository{
		queries: sqlc.New(db),
		db:      db,
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

func (r *highlightRepository) WithTx(tx *sql.Tx) HighlightRepository {
	if tx == nil {
		return r
	}
	return &highlightRepository{
		queries: sqlc.New(tx),
		db:      r.db,
		c:       r.c,
		sfg:     r.sfg,
	}
}

func (r *highlightRepository) Create(ctx context.Context, arg sqlc.CreateHighlightParams) (*models.HighlightEntity, error) {
	res, err := r.queries.CreateHighlight(ctx, arg)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		r.c.Del(ctx, cache.BuildKey("highlight", "ids", "chapter", arg.UserID, arg.ChapterID))
		r.c.Del(ctx, cache.BuildKey("highlight", "id", res.ID))
	}
	entity := &models.HighlightEntity{}
	return entity.FromSqlc(res), nil
}

func (r *highlightRepository) GetByChapter(ctx context.Context, userID int64, chapterID string) ([]*models.HighlightEntity, error) {
	key := cache.BuildKey("highlight", "ids", "chapter", userID, chapterID)

	if r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			result, err := r.GetHighlightsByIDs(ctx, ids)
			if err == nil {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.GetHighlightsByChapter(ctx, sqlc.GetHighlightsByChapterParams{
			UserID:    userID,
			ChapterID: chapterID,
		})
		if err != nil {
			return nil, err
		}

		ids := make([]string, len(rows))
		result := make([]*models.HighlightEntity, len(rows))
		for i, row := range rows {
			h := (&models.HighlightEntity{}).FromSqlc(row)
			result[i] = h
			ids[i] = h.ID
		}

		if r.c != nil {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			missingToCache := make(map[string]any)
			for _, h := range result {
				missingToCache[cache.BuildKey("highlight", "id", h.ID)] = h
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}

		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.HighlightEntity), nil
}

func (r *highlightRepository) GetHighlightsByIDs(ctx context.Context, ids []string) ([]*models.HighlightEntity, error) {
	if len(ids) == 0 {
		return []*models.HighlightEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("highlight", "id", id)
	}

	highlights := make([]*models.HighlightEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var h models.HighlightEntity
				if err := jsonx.Unmarshal(bytes, &h); err == nil {
					highlights = append(highlights, &h)
					continue
				}
			}
			missingIds = append(missingIds, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIds = ids
		missingKeys = keys
	}

	if len(missingIds) > 0 {
		rows, err := r.queries.GetHighlightsByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}

		missingMap := make(map[string]*models.HighlightEntity)
		for _, row := range rows {
			h := (&models.HighlightEntity{}).FromSqlc(row)
			missingMap[h.ID] = h
			highlights = append(highlights, h)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if h, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = h
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	hMap := make(map[string]*models.HighlightEntity)
	for _, h := range highlights {
		hMap[h.ID] = h
	}
	ordered := make([]*models.HighlightEntity, 0, len(ids))
	for _, id := range ids {
		if h, ok := hMap[id]; ok {
			ordered = append(ordered, h)
		}
	}

	return ordered, nil
}

func (r *highlightRepository) Delete(ctx context.Context, arg sqlc.DeleteHighlightParams) error {
	err := r.queries.DeleteHighlight(ctx, arg)
	if err == nil && r.c != nil {
		r.c.Del(ctx, cache.BuildKey("highlight", "id", arg.ID))
		r.c.DelByPattern(ctx, cache.BuildKey("highlight", "ids", "chapter", arg.UserID, "*"))
	}
	return err
}

func (r *highlightRepository) UpdateNote(ctx context.Context, arg sqlc.UpdateHighlightNoteParams) (*models.HighlightEntity, error) {
	res, err := r.queries.UpdateHighlightNote(ctx, arg)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		r.c.Del(ctx, cache.BuildKey("highlight", "id", res.ID))
	}
	entity := &models.HighlightEntity{}
	return entity.FromSqlc(res), nil
}
