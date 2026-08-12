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
	GetByChapter(ctx context.Context, userID string, chapterID string) ([]*models.HighlightEntity, error)
	GetHighlightsByIDs(ctx context.Context, ids []string) ([]*models.HighlightEntity, error)
	GetHighlightsByBook(ctx context.Context, userID string, bookID string) ([]*models.HighlightBookEntity, error)
	Delete(ctx context.Context, arg sqlc.DeleteHighlightParams) error
	UpdateNote(ctx context.Context, arg sqlc.UpdateHighlightNoteParams) (*models.HighlightEntity, error)
}

type highlightRepository struct {
	queries *sqlc.Queries
	db      *sql.DB
	c       cache.Cache
	inTx    bool
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
		inTx:    true,
		sfg:     &singleflight.Group{},
	}
}

func (r *highlightRepository) Create(ctx context.Context, arg sqlc.CreateHighlightParams) (*models.HighlightEntity, error) {
	res, err := r.queries.CreateHighlight(ctx, arg)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		r.c.Del(ctx, cache.BuildKey("highlight", "ids", "chapter", arg.UserID, arg.ChapterID))
		r.c.Del(ctx, cache.BuildKey("highlight", "book_export", "user_book", arg.UserID, arg.BookID))
		r.c.Del(ctx, cache.BuildKey("highlight", "id", res.ID))
		r.c.Del(ctx, cache.BuildKey("highlight", "book_export", "id", res.ID))
	}
	entity := &models.HighlightEntity{}
	return entity.FromSqlc(res), nil
}

func (r *highlightRepository) GetByChapter(ctx context.Context, userID string, chapterID string) ([]*models.HighlightEntity, error) {
	key := cache.BuildKey("highlight", "ids", "chapter", userID, chapterID)

	if r.c != nil && !r.inTx {
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

		if r.c != nil && !r.inTx {
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

	if r.c != nil && !r.inTx {
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
		rows, err := queryInChunks(missingIds, func(chunk []string) ([]sqlc.Highlight, error) {
			return r.queries.GetHighlightsByIDs(ctx, chunk)
		})
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

func (r *highlightRepository) GetHighlightsByBook(ctx context.Context, userID string, bookID string) ([]*models.HighlightBookEntity, error) {
	listKey := cache.BuildKey("highlight", "book_export", "user_book", userID, bookID)

	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, listKey, &ids); err == nil {
			result, err := r.getHighlightBooksByIDs(ctx, ids)
			if err == nil {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(listKey, func() (any, error) {
		rows, err := r.queries.GetHighlightsByBook(ctx, sqlc.GetHighlightsByBookParams{
			UserID: userID,
			BookID: bookID,
		})
		if err != nil {
			return nil, err
		}

		ids := make([]string, len(rows))
		result := make([]*models.HighlightBookEntity, len(rows))
		for i, row := range rows {
			h := (&models.HighlightBookEntity{}).FromSqlc(row)
			result[i] = h
			ids[i] = h.ID
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, listKey, ids, constants.ListCacheDuration)
			toCache := make(map[string]any, len(result))
			for _, h := range result {
				toCache[cache.BuildKey("highlight", "book_export", "id", h.ID)] = h
			}
			_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
		}

		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.HighlightBookEntity), nil
}

func (r *highlightRepository) getHighlightBooksByIDs(ctx context.Context, ids []string) ([]*models.HighlightBookEntity, error) {
	if len(ids) == 0 {
		return []*models.HighlightBookEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("highlight", "book_export", "id", id)
	}

	byID := make(map[string]*models.HighlightBookEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))

	if r.c != nil && !r.inTx {
		raws := r.c.MGet(ctx, keys...)
		for i, raw := range raws {
			if len(raw) > 0 {
				var h models.HighlightBookEntity
				if err := jsonx.Unmarshal(raw, &h); err == nil {
					byID[h.ID] = &h
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
		rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.GetHighlightBooksByIDsRow, error) {
			return r.queries.GetHighlightBooksByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}
		fetched := make(map[string]*models.HighlightBookEntity, len(rows))
		for _, row := range rows {
			h := (&models.HighlightBookEntity{}).FromSqlcByIDs(row)
			byID[h.ID] = h
			fetched[h.ID] = h
		}

		if r.c != nil && !r.inTx {
			toCache := make(map[string]any, len(fetched))
			for i, missingID := range missingIDs {
				if h, ok := fetched[missingID]; ok {
					toCache[missingKeys[i]] = h
				}
			}
			if len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
		}
	}

	ordered := make([]*models.HighlightBookEntity, 0, len(ids))
	for _, id := range ids {
		if h := byID[id]; h != nil {
			ordered = append(ordered, h)
		}
	}
	return ordered, nil
}

func (r *highlightRepository) Delete(ctx context.Context, arg sqlc.DeleteHighlightParams) error {
	err := r.queries.DeleteHighlight(ctx, arg)
	if err == nil && r.c != nil {
		r.c.Del(ctx, cache.BuildKey("highlight", "id", arg.ID))
		r.c.Del(ctx, cache.BuildKey("highlight", "book_export", "id", arg.ID))
		r.c.DelByPattern(ctx, cache.BuildKey("highlight", "ids", "chapter", arg.UserID, "*"))
		r.c.DelByPattern(ctx, cache.BuildKey("highlight", "book_export", "user_book", arg.UserID, "*"))
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
		r.c.Del(ctx, cache.BuildKey("highlight", "book_export", "id", res.ID))
	}
	entity := &models.HighlightEntity{}
	return entity.FromSqlc(res), nil
}
