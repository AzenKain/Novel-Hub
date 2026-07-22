package repositories

import (
	"context"
	"database/sql"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/crypto"
	"novelhub/pkg/jsonx"
)

type TrackerRepository interface {
	GetByID(ctx context.Context, id int64) (*models.UserTrackerEntity, error)
	GetUserTracker(ctx context.Context, userID int64, provider string) (*models.UserTrackerEntity, error)
	GetUserTrackersByIDs(ctx context.Context, ids []int64) ([]*models.UserTrackerEntity, error)
	UpsertUserTracker(ctx context.Context, userID int64, provider string, accessToken string) (*models.UserTrackerEntity, error)
	DeleteUserTracker(ctx context.Context, userID int64, provider string) error

	GetMappingByID(ctx context.Context, id int64) (*models.BookTrackerMappingEntity, error)
	GetBookTrackerMapping(ctx context.Context, bookID int64, provider string) (*models.BookTrackerMappingEntity, error)
	GetBookTrackerMappingsByIDs(ctx context.Context, ids []int64) ([]*models.BookTrackerMappingEntity, error)
	UpsertBookTrackerMapping(ctx context.Context, bookID int64, provider string, externalSeriesID string) (*models.BookTrackerMappingEntity, error)
	WithTx(tx *sql.Tx) TrackerRepository
}

type trackerRepository struct {
	q   *sqlc.Queries
	c   cache.Cache
	sfg *singleflight.Group
}

func NewTrackerRepository(db sqlc.DBTX, c cache.Cache) TrackerRepository {
	return &trackerRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *trackerRepository) WithTx(tx *sql.Tx) TrackerRepository {
	return &trackerRepository{
		q:   r.q.WithTx(tx),
		c:   r.c,
		sfg: r.sfg,
	}
}

func (r *trackerRepository) GetByID(ctx context.Context, id int64) (*models.UserTrackerEntity, error) {
	key := cache.BuildKey("user_tracker", "id", id)
	if r.c != nil {
		var entity models.UserTrackerEntity
		if err := r.c.Get(ctx, key, &entity); err == nil {
			return &entity, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.GetUserTrackersByIDs(ctx, []int64{id})
		if err != nil || len(rows) == 0 {
			return nil, sql.ErrNoRows
		}

		entity := (&models.UserTrackerEntity{}).FromSqlc(rows[0])
		if r.c != nil {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserTrackerEntity), nil
}

func (r *trackerRepository) GetUserTrackersByIDs(ctx context.Context, ids []int64) ([]*models.UserTrackerEntity, error) {
	if len(ids) == 0 {
		return []*models.UserTrackerEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("user_tracker", "id", id)
	}

	trackers := make([]*models.UserTrackerEntity, 0, len(ids))
	missingIds := []int64{}
	missingKeys := []string{}

	if r.c != nil {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var tracker models.UserTrackerEntity
				if err := jsonx.Unmarshal(bytes, &tracker); err == nil {
					trackers = append(trackers, &tracker)
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
		rows, err := r.q.GetUserTrackersByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[int64]*models.UserTrackerEntity)
		for _, row := range rows {
			u := (&models.UserTrackerEntity{}).FromSqlc(row)
			missingMap[u.ID] = u
			trackers = append(trackers, u)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if u, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = u
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	trackerMap := make(map[int64]*models.UserTrackerEntity)
	for _, u := range trackers {
		trackerMap[u.ID] = u
	}
	ordered := make([]*models.UserTrackerEntity, 0, len(ids))
	for _, id := range ids {
		if u, ok := trackerMap[id]; ok {
			ordered = append(ordered, u)
		}
	}

	return ordered, nil
}

func (r *trackerRepository) GetUserTracker(ctx context.Context, userID int64, provider string) (*models.UserTrackerEntity, error) {
	key := cache.BuildKey("user_tracker", "user_provider", userID, provider)
	if r.c != nil {
		var entity models.UserTrackerEntity
		if err := r.c.Get(ctx, key, &entity); err == nil {
			return &entity, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.q.GetUserTracker(ctx, sqlc.GetUserTrackerParams{
			UserID:   userID,
			Provider: provider,
		})
		if err != nil {
			return nil, err
		}

		entity := (&models.UserTrackerEntity{}).FromSqlc(res)
		if r.c != nil {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("user_tracker", "id", entity.ID), entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserTrackerEntity), nil
}

func (r *trackerRepository) UpsertUserTracker(ctx context.Context, userID int64, provider string, accessToken string) (*models.UserTrackerEntity, error) {
	encToken, err := crypto.EncryptAES(accessToken)
	if err != nil {
		encToken = accessToken
	}

	res, err := r.q.UpsertUserTracker(ctx, sqlc.UpsertUserTrackerParams{
		UserID:      userID,
		Provider:    provider,
		AccessToken: encToken,
	})
	if err != nil {
		return nil, err
	}
	entity := (&models.UserTrackerEntity{}).FromSqlc(res)
	if r.c != nil {
		key := cache.BuildKey("user_tracker", "user_provider", userID, provider)
		idKey := cache.BuildKey("user_tracker", "id", entity.ID)
		_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		_ = r.c.Set(ctx, idKey, entity, constants.NormalCacheDuration)
	}
	return entity, nil
}

func (r *trackerRepository) DeleteUserTracker(ctx context.Context, userID int64, provider string) error {
	existing, _ := r.GetUserTracker(ctx, userID, provider)

	err := r.q.DeleteUserTracker(ctx, sqlc.DeleteUserTrackerParams{
		UserID:   userID,
		Provider: provider,
	})
	if err == nil && r.c != nil {
		key := cache.BuildKey("user_tracker", "user_provider", userID, provider)
		_ = r.c.Del(ctx, key)
		if existing != nil {
			idKey := cache.BuildKey("user_tracker", "id", existing.ID)
			_ = r.c.Del(ctx, idKey)
		}
	}
	return err
}

func (r *trackerRepository) GetMappingByID(ctx context.Context, id int64) (*models.BookTrackerMappingEntity, error) {
	key := cache.BuildKey("book_tracker_mapping", "id", id)
	if r.c != nil {
		var entity models.BookTrackerMappingEntity
		if err := r.c.Get(ctx, key, &entity); err == nil {
			return &entity, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.GetBookTrackerMappingsByIDs(ctx, []int64{id})
		if err != nil || len(rows) == 0 {
			return nil, sql.ErrNoRows
		}

		entity := (&models.BookTrackerMappingEntity{}).FromSqlc(rows[0])
		if r.c != nil {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookTrackerMappingEntity), nil
}

func (r *trackerRepository) GetBookTrackerMappingsByIDs(ctx context.Context, ids []int64) ([]*models.BookTrackerMappingEntity, error) {
	if len(ids) == 0 {
		return []*models.BookTrackerMappingEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("book_tracker_mapping", "id", id)
	}

	mappings := make([]*models.BookTrackerMappingEntity, 0, len(ids))
	missingIds := []int64{}
	missingKeys := []string{}

	if r.c != nil {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var mapping models.BookTrackerMappingEntity
				if err := jsonx.Unmarshal(bytes, &mapping); err == nil {
					mappings = append(mappings, &mapping)
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
		rows, err := r.q.GetBookTrackerMappingsByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[int64]*models.BookTrackerMappingEntity)
		for _, row := range rows {
			m := (&models.BookTrackerMappingEntity{}).FromSqlc(row)
			missingMap[m.ID] = m
			mappings = append(mappings, m)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if m, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = m
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	mappingMap := make(map[int64]*models.BookTrackerMappingEntity)
	for _, m := range mappings {
		mappingMap[m.ID] = m
	}
	ordered := make([]*models.BookTrackerMappingEntity, 0, len(ids))
	for _, id := range ids {
		if m, ok := mappingMap[id]; ok {
			ordered = append(ordered, m)
		}
	}

	return ordered, nil
}

func (r *trackerRepository) GetBookTrackerMapping(ctx context.Context, bookID int64, provider string) (*models.BookTrackerMappingEntity, error) {
	key := cache.BuildKey("book_tracker_mapping", "book_provider", bookID, provider)
	if r.c != nil {
		var entity models.BookTrackerMappingEntity
		if err := r.c.Get(ctx, key, &entity); err == nil {
			return &entity, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.q.GetBookTrackerMapping(ctx, sqlc.GetBookTrackerMappingParams{
			BookID:   bookID,
			Provider: provider,
		})
		if err != nil {
			return nil, err
		}

		entity := (&models.BookTrackerMappingEntity{}).FromSqlc(res)
		if r.c != nil {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("book_tracker_mapping", "id", entity.ID), entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookTrackerMappingEntity), nil
}

func (r *trackerRepository) UpsertBookTrackerMapping(ctx context.Context, bookID int64, provider string, externalSeriesID string) (*models.BookTrackerMappingEntity, error) {
	res, err := r.q.UpsertBookTrackerMapping(ctx, sqlc.UpsertBookTrackerMappingParams{
		BookID:           bookID,
		Provider:         provider,
		ExternalSeriesID: externalSeriesID,
	})
	if err != nil {
		return nil, err
	}
	entity := (&models.BookTrackerMappingEntity{}).FromSqlc(res)
	if r.c != nil {
		key := cache.BuildKey("book_tracker_mapping", "book_provider", bookID, provider)
		idKey := cache.BuildKey("book_tracker_mapping", "id", entity.ID)
		_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		_ = r.c.Set(ctx, idKey, entity, constants.NormalCacheDuration)
	}
	return entity, nil
}
