package repositories

import (
	"context"
	"database/sql"
	"strings"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type WebhookRepository interface {
	Create(ctx context.Context, entity *models.WebhookEntity) (*models.WebhookEntity, error)
	GetByID(ctx context.Context, id string) (*models.WebhookEntity, error)
	ListAll(ctx context.Context) ([]*models.WebhookEntity, error)
	ListActive(ctx context.Context) ([]*models.WebhookEntity, error)
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

	res := models.WebhookFromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("webhook", res.ID), res, constants.NormalCacheDuration)
		_ = r.c.Del(ctx, constants.CacheKeyWebhookAll, constants.CacheKeyWebhookActive)
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
		return models.WebhookFromSqlc(row), nil
	})
	if err != nil {
		return nil, err
	}

	res := v.(*models.WebhookEntity)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
	}
	return res, nil
}

func (r *webhookRepository) ListAll(ctx context.Context) ([]*models.WebhookEntity, error) {
	key := constants.CacheKeyWebhookAll
	if r.c != nil && !r.inTx {
		var cached []*models.WebhookEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		rows, err := r.q.ListAllWebhooks(ctx)
		if err != nil {
			return nil, err
		}
		var res []*models.WebhookEntity
		for _, row := range rows {
			res = append(res, models.WebhookFromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.ListCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.WebhookEntity), nil
}

func (r *webhookRepository) ListActive(ctx context.Context) ([]*models.WebhookEntity, error) {
	key := constants.CacheKeyWebhookActive
	if r.c != nil && !r.inTx {
		var cached []*models.WebhookEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		rows, err := r.q.ListActiveWebhooks(ctx)
		if err != nil {
			return nil, err
		}
		var res []*models.WebhookEntity
		for _, row := range rows {
			res = append(res, models.WebhookFromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.ListCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.WebhookEntity), nil
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

	res := models.WebhookFromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("webhook", res.ID), res, constants.NormalCacheDuration)
		_ = r.c.Del(ctx, constants.CacheKeyWebhookAll, constants.CacheKeyWebhookActive)
	}
	return res, nil
}

func (r *webhookRepository) Delete(ctx context.Context, id string) error {
	err := r.q.DeleteWebhook(ctx, id)
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("webhook", id), constants.CacheKeyWebhookAll, constants.CacheKeyWebhookActive)
	}
	return nil
}
