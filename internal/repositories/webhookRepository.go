package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type WebhookRepository interface {
	Create(ctx context.Context, entity *models.WebhookEntity) (*models.WebhookEntity, error)
	GetByID(ctx context.Context, id string) (*models.WebhookEntity, error)
	GetWebhooksByIDs(ctx context.Context, ids []string) ([]*models.WebhookEntity, error)
	ListAll(ctx context.Context, limit, offset int64) ([]*models.WebhookEntity, error)
	ListActive(ctx context.Context, limit, offset int64) ([]*models.WebhookEntity, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, entity *models.WebhookEntity) (*models.WebhookEntity, error)
	Delete(ctx context.Context, id string) error
	WithTx(tx *sql.Tx) WebhookRepository
}

type webhookRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewWebhookRepository(db sqlc.DBTX, c cache.Cache) WebhookRepository {
	return &webhookRepository{q: sqlc.New(db), c: c, sf: &singleflight.Group{}}
}

func (r *webhookRepository) WithTx(tx *sql.Tx) WebhookRepository {
	if tx == nil {
		return r
	}
	return &webhookRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sf: r.sf}
}

func (r *webhookRepository) Create(ctx context.Context, entity *models.WebhookEntity) (*models.WebhookEntity, error) {
	eventsStr := strings.Join(entity.Events, ",")
	isActiveInt := int64(0)
	if entity.IsActive {
		isActiveInt = 1
	}

	row, err := r.q.CreateWebhook(ctx, sqlc.CreateWebhookParams{
		ID:            entity.ID,
		Name:          entity.Name,
		Url:           entity.URL,
		TemplateType:  entity.TemplateType,
		Secret:        convert.StrPtrToNullString(entity.Secret),
		CustomHeaders: convert.StrPtrToNullString(entity.CustomHeaders),
		Events:        eventsStr,
		IsActive:      isActiveInt,
	})
	if err != nil {
		return nil, err
	}

	res := (&models.WebhookEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("webhook", res.ID), res, constants.NormalCacheDuration)
		_ = r.c.DelByPattern(ctx, "webhook:list:*")
		_ = r.c.Del(ctx, "webhook:count")
	}
	return res, nil
}

func (r *webhookRepository) GetByID(ctx context.Context, id string) (*models.WebhookEntity, error) {
	cacheKey := cache.BuildKey("webhook", id)
	if r.c != nil && !r.inTx {
		var entity models.WebhookEntity
		if err := r.c.Get(ctx, cacheKey, &entity); err == nil {
			return &entity, nil
		}
	}

	v, err, _ := r.sf.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetWebhookByID(ctx, id)
		if err != nil {
			return nil, err
		}
		res := (&models.WebhookEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.WebhookEntity), nil
}

func (r *webhookRepository) GetWebhooksByIDs(ctx context.Context, ids []string) ([]*models.WebhookEntity, error) {
	if len(ids) == 0 {
		return []*models.WebhookEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("webhook", id)
	}

	webhooks := make([]*models.WebhookEntity, len(ids))
	missingIDs := make([]string, 0)
	missingIndices := make([]int, 0)

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var wh models.WebhookEntity
				if err := jsonx.Unmarshal(bytes, &wh); err == nil {
					webhooks[i] = &wh
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
			missingIndices = append(missingIndices, i)
		}
	} else {
		missingIDs = ids
		for i := range ids {
			missingIndices = append(missingIndices, i)
		}
	}

	if len(missingIDs) > 0 {
		rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.Webhook, error) {
			return r.q.GetWebhooksByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}

		missingMap := make(map[string]*models.WebhookEntity, len(rows))
		for _, row := range rows {
			wh := (&models.WebhookEntity{}).FromSqlc(row)
			missingMap[wh.ID] = wh
		}

		if r.c != nil && !r.inTx {
			toCache := make(map[string]any)
			for _, id := range missingIDs {
				if wh, ok := missingMap[id]; ok {
					toCache[cache.BuildKey("webhook", id)] = wh
				}
			}
			if len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
		}

		for idx, i := range missingIndices {
			id := missingIDs[idx]
			if wh, ok := missingMap[id]; ok {
				webhooks[i] = wh
			}
		}
	}

	result := make([]*models.WebhookEntity, 0, len(webhooks))
	for _, wh := range webhooks {
		if wh != nil {
			result = append(result, wh)
		}
	}
	return result, nil
}

func (r *webhookRepository) ListAll(ctx context.Context, limit, offset int64) ([]*models.WebhookEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	listKey := cache.BuildKey("webhook", "list", "all", fmt.Sprintf("%d_%d", limit, offset))
	var ids []string

	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err == nil {
			return r.GetWebhooksByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sf.Do(listKey, func() (any, error) {
		fetchedIDs, err := r.q.ListAllWebhookIDs(ctx, sqlc.ListAllWebhookIDsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, listKey, fetchedIDs, constants.ListCacheDuration)
		}
		return fetchedIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetWebhooksByIDs(ctx, v.([]string))
}

func (r *webhookRepository) ListActive(ctx context.Context, limit, offset int64) ([]*models.WebhookEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	listKey := cache.BuildKey("webhook", "list", "active", fmt.Sprintf("%d_%d", limit, offset))
	var ids []string

	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err == nil {
			return r.GetWebhooksByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sf.Do(listKey, func() (any, error) {
		fetchedIDs, err := r.q.ListActiveWebhookIDs(ctx, sqlc.ListActiveWebhookIDsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, listKey, fetchedIDs, constants.ListCacheDuration)
		}
		return fetchedIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetWebhooksByIDs(ctx, v.([]string))
}

func (r *webhookRepository) Count(ctx context.Context) (int64, error) {
	key := "webhook:count"
	if r.c != nil && !r.inTx {
		var cached int64
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		count, err := r.q.CountWebhooks(ctx)
		if err != nil {
			return nil, err
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

func (r *webhookRepository) Update(ctx context.Context, entity *models.WebhookEntity) (*models.WebhookEntity, error) {
	eventsStr := strings.Join(entity.Events, ",")
	isActiveInt := int64(0)
	if entity.IsActive {
		isActiveInt = 1
	}

	row, err := r.q.UpdateWebhook(ctx, sqlc.UpdateWebhookParams{
		ID:            entity.ID,
		Name:          entity.Name,
		Url:           entity.URL,
		TemplateType:  entity.TemplateType,
		Secret:        convert.StrPtrToNullString(entity.Secret),
		CustomHeaders: convert.StrPtrToNullString(entity.CustomHeaders),
		Events:        eventsStr,
		IsActive:      isActiveInt,
	})
	if err != nil {
		return nil, err
	}

	res := (&models.WebhookEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("webhook", res.ID), res, constants.NormalCacheDuration)
		_ = r.c.DelByPattern(ctx, "webhook:list:*")
	}
	return res, nil
}

func (r *webhookRepository) Delete(ctx context.Context, id string) error {
	err := r.q.DeleteWebhook(ctx, id)
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("webhook", id), "webhook:count")
		_ = r.c.DelByPattern(ctx, "webhook:list:*")
	}
	return nil
}
