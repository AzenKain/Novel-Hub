package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type DeviceRepository interface {
	Create(ctx context.Context, params sqlc.CreateUserDeviceParams) (*models.UserDeviceEntity, error)
	GetByID(ctx context.Context, id string) (*models.UserDeviceEntity, error)
	ListByUserID(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*models.UserDeviceEntity, error)
	Delete(ctx context.Context, id string, userID string) error
	WithTx(tx *sql.Tx) DeviceRepository
}

type deviceRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewDeviceRepository(db sqlc.DBTX, c cache.Cache) DeviceRepository {
	return &deviceRepository{q: sqlc.New(db), c: c, sf: &singleflight.Group{}}
}

func (r *deviceRepository) WithTx(tx *sql.Tx) DeviceRepository {
	if tx == nil {
		return r
	}
	return &deviceRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sf: r.sf}
}

func deviceCacheKey(id string) string {
	return cache.BuildKey("user_device", "id", id)
}

func deviceUserListCacheKey(userID string, cursor *time.Time, cursorID string, limit int64) string {
	cursorStr := ""
	if cursor != nil {
		cursorStr = cursor.Format(time.RFC3339Nano)
	}
	return cache.BuildKey("user_device", "user", userID, fmt.Sprintf("%s_%s_%d", cursorStr, cursorID, limit))
}

func (r *deviceRepository) invalidate(ctx context.Context, id string, userID string) {
	if r.c != nil {
		_ = r.c.Del(ctx, deviceCacheKey(id))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("user_device", "user", userID, "*"))
	}
}

func (r *deviceRepository) Create(ctx context.Context, params sqlc.CreateUserDeviceParams) (*models.UserDeviceEntity, error) {
	row, err := r.q.CreateUserDevice(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := (&models.UserDeviceEntity{}).FromSqlc(row)
	r.invalidate(ctx, entity.ID, entity.UserID)
	return entity, nil
}

func (r *deviceRepository) GetByID(ctx context.Context, id string) (*models.UserDeviceEntity, error) {
	key := deviceCacheKey(id)
	if r.c != nil && !r.inTx {
		var cached models.UserDeviceEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetUserDeviceByID(ctx, id)
		if err != nil {
			return nil, err
		}
		entity := (&models.UserDeviceEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserDeviceEntity), nil
}

func (r *deviceRepository) ListByUserID(ctx context.Context, userID string, cursor *time.Time, cursorID string, limit int64) ([]*models.UserDeviceEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	listKey := deviceUserListCacheKey(userID, cursor, cursorID, limit)
	var ids []string

	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err != nil {
			ids = nil
		}
	}

	if ids == nil {
		v, err, _ := r.sf.Do(listKey, func() (any, error) {
			fetchedIDs, err := r.q.ListUserDeviceIDs(ctx, sqlc.ListUserDeviceIDsParams{
				UserID:          userID,
				CursorUpdatedAt: cursorTimeArg(cursor),
				CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
				Limit:           limit,
			})
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, listKey, fetchedIDs, constants.NormalCacheDuration)
			}
			return fetchedIDs, nil
		})
		if err != nil {
			return nil, err
		}
		ids = v.([]string)
	}

	if len(ids) == 0 {
		return []*models.UserDeviceEntity{}, nil
	}

	entities := make([]*models.UserDeviceEntity, len(ids))
	missingIDs := make([]string, 0)
	missingIndices := make([]int, 0)

	for i, id := range ids {
		key := deviceCacheKey(id)
		if r.c != nil && !r.inTx {
			var cached models.UserDeviceEntity
			if err := r.c.Get(ctx, key, &cached); err == nil {
				entities[i] = &cached
				continue
			}
		}
		missingIDs = append(missingIDs, id)
		missingIndices = append(missingIndices, i)
	}

	if len(missingIDs) > 0 {
		sfKey := "mget_user_devices_" + userID
		v, err, _ := r.sf.Do(sfKey, func() (any, error) {
			rows, err := r.q.GetUserDevicesByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetchedEntities := (&models.UserDeviceEntities{}).FromSqlc(rows)
			if r.c != nil && !r.inTx {
				pairs := make(map[string]any, len(fetchedEntities))
				for _, entity := range fetchedEntities {
					pairs[deviceCacheKey(entity.ID)] = entity
				}
				_ = r.c.MSet(ctx, pairs, constants.NormalCacheDuration)
			}
			return fetchedEntities, nil
		})
		if err != nil {
			return nil, err
		}

		fetchedList := v.([]*models.UserDeviceEntity)
		fetchedMap := make(map[string]*models.UserDeviceEntity, len(fetchedList))
		for _, item := range fetchedList {
			fetchedMap[item.ID] = item
		}
		for idx, missingID := range missingIDs {
			originalIndex := missingIndices[idx]
			if item, found := fetchedMap[missingID]; found {
				entities[originalIndex] = item
			}
		}
	}

	result := make([]*models.UserDeviceEntity, 0, len(entities))
	for _, e := range entities {
		if e != nil {
			result = append(result, e)
		}
	}

	return result, nil
}

func (r *deviceRepository) Delete(ctx context.Context, id string, userID string) error {
	if err := r.q.DeleteUserDevice(ctx, sqlc.DeleteUserDeviceParams{ID: id, UserID: userID}); err != nil {
		return err
	}
	r.invalidate(ctx, id, userID)
	return nil
}
