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

type SettingsRepository interface {
	List(ctx context.Context) ([]*models.AppSettingEntity, error)
	Get(ctx context.Context, key string) (*models.AppSettingEntity, error)
	Upsert(ctx context.Context, key string, valueJSON string) error
	GetSetupState(ctx context.Context, key string) (string, error)
	ClaimInitialSetup(ctx context.Context) (bool, error)
	UpsertSetupState(ctx context.Context, key string, value string) error
	CountAdminUsers(ctx context.Context) (int64, error)
	WithTx(tx *sql.Tx) SettingsRepository
}

type settingsRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewSettingsRepository(db sqlc.DBTX, c cache.Cache) SettingsRepository {
	return &settingsRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *settingsRepository) WithTx(tx *sql.Tx) SettingsRepository {
	return &settingsRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  r.sfg,
	}
}

func (r *settingsRepository) List(ctx context.Context) ([]*models.AppSettingEntity, error) {
	key := "settings:all"
	if r.c != nil {
		var keys []string
		if err := r.c.Get(ctx, key, &keys); err == nil {
			if result, ok := r.getSettingsByKeys(ctx, keys); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		keyRows, err := r.q.ListAppSettingKeys(ctx)
		if err != nil {
			return nil, err
		}

		if len(keyRows) == 0 {
			if r.c != nil {
				_ = r.c.Set(ctx, key, []string{}, constants.ListCacheDuration)
			}
			return []*models.AppSettingEntity{}, nil
		}

		rows, err := r.q.GetAppSettingsByKeys(ctx, keyRows)
		if err != nil {
			return nil, err
		}

		out := (&models.AppSettingEntities{}).FromSqlc(rows)
		keys := make([]string, len(out))
		for i, entity := range out {
			keys[i] = entity.Key
		}

		if r.c != nil {
			_ = r.c.Set(ctx, key, keys, constants.ListCacheDuration)
			r.cacheSettingEntities(ctx, out)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AppSettingEntity), nil
}

func (r *settingsRepository) getSettingsByKeys(ctx context.Context, keys []string) ([]*models.AppSettingEntity, bool) {
	if len(keys) == 0 {
		return []*models.AppSettingEntity{}, true
	}
	if r.c == nil {
		return nil, false
	}

	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = cache.BuildKey("settings", "key", key)
	}

	cachedBytes := r.c.MGet(ctx, cacheKeys...)
	ordered := make([]*models.AppSettingEntity, 0, len(keys))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.AppSettingEntity
		if err := jsonx.Unmarshal(bytes, &entity); err == nil {
			ordered = append(ordered, &entity)
		} else {
			return nil, false
		}
	}

	return ordered, true
}

func (r *settingsRepository) cacheSettingEntities(ctx context.Context, entities []*models.AppSettingEntity) {
	if r.c == nil || len(entities) == 0 {
		return
	}
	toCache := make(map[string]any, len(entities))
	for _, entity := range entities {
		toCache[cache.BuildKey("settings", "key", entity.Key)] = entity
	}
	_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
}

func (r *settingsRepository) Get(ctx context.Context, key string) (*models.AppSettingEntity, error) {
	cacheKey := cache.BuildKey("settings", "key", key)
	if r.c != nil {
		var setting models.AppSettingEntity
		if err := r.c.Get(ctx, cacheKey, &setting); err == nil {
			return &setting, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetAppSetting(ctx, key)
		if err != nil {
			return nil, err
		}
		result := (&models.AppSettingEntity{}).FromSqlc(row)
		if r.c != nil {
			_ = r.c.Set(ctx, cacheKey, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.AppSettingEntity), nil
}

func (r *settingsRepository) Upsert(ctx context.Context, key string, valueJSON string) error {
	if err := r.q.UpsertAppSetting(ctx, sqlc.UpsertAppSettingParams{Key: key, ValueJson: valueJSON}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "settings:all", cache.BuildKey("settings", "key", key))
	}
	return nil
}

func (r *settingsRepository) GetSetupState(ctx context.Context, key string) (string, error) {
	cacheKey := cache.BuildKey("setup_state", "key", key)
	if r.c != nil {
		var state string
		if err := r.c.Get(ctx, cacheKey, &state); err == nil {
			return state, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetSetupState(ctx, key)
		if err != nil {
			return "", err
		}
		if r.c != nil {
			_ = r.c.Set(ctx, cacheKey, row.Value, constants.NormalCacheDuration)
		}
		return row.Value, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (r *settingsRepository) ClaimInitialSetup(ctx context.Context) (bool, error) {
	updated, err := r.q.ClaimInitialSetup(ctx)
	if err != nil {
		return false, err
	}
	if updated > 0 && r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("setup_state", "key", "completed"))
	}
	return updated > 0, nil
}

func (r *settingsRepository) UpsertSetupState(ctx context.Context, key string, value string) error {
	if err := r.q.UpsertSetupState(ctx, sqlc.UpsertSetupStateParams{Key: key, Value: value}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("setup_state", "key", key))
	}
	return nil
}

func (r *settingsRepository) CountAdminUsers(ctx context.Context) (int64, error) {
	key := constants.CacheKeySettingsAdminCount
	if r.c != nil && !r.inTx {
		var count int64
		if err := r.c.Get(ctx, key, &count); err == nil {
			return count, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		count, err := r.q.CountAdminUsers(ctx)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, count, constants.NormalCacheDuration)
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}
