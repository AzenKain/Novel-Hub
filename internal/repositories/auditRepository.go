package repositories

import (
	"context"
	"database/sql"
	"strconv"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type AuditFilter struct {
	Action          string
	ActorID         string
	CursorCreatedAt string
	CursorID        string
	Limit           int64
}

type AuditRepository interface {
	Create(ctx context.Context, entity *models.AuditLogEntity) (*models.AuditLogEntity, error)
	List(ctx context.Context, filter AuditFilter) ([]*models.AuditLogEntity, error)
	Count(ctx context.Context, filter AuditFilter) (int64, error)
	ListActions(ctx context.Context) ([]string, error)
	Prune(ctx context.Context, keepDays int64) (int64, error)
	WithTx(tx *sql.Tx) AuditRepository
}

type auditRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewAuditRepository(db sqlc.DBTX, c cache.Cache) AuditRepository {
	return &auditRepository{q: sqlc.New(db), c: c, sf: &singleflight.Group{}}
}

func (r *auditRepository) WithTx(tx *sql.Tx) AuditRepository {
	if tx == nil {
		return r
	}
	return &auditRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sf: r.sf}
}

func nullableFilter(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *auditRepository) Create(ctx context.Context, entity *models.AuditLogEntity) (*models.AuditLogEntity, error) {
	row, err := r.q.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		ID:          entity.ID,
		ActorID:     convert.StrPtrToNullString(entity.ActorID),
		ActorEmail:  entity.ActorEmail,
		Action:      entity.Action,
		TargetType:  entity.TargetType,
		TargetID:    convert.StrPtrToNullString(entity.TargetID),
		TargetLabel: entity.TargetLabel,
		Ip:          entity.IP,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, constants.CacheKeyAuditListPattern)
		_ = r.c.DelByPattern(ctx, constants.CacheKeyAuditCountPattern)
		_ = r.c.Del(ctx, constants.CacheKeyAuditActions)
	}
	return models.AuditLogFromSqlc(row), nil
}

func (r *auditRepository) List(ctx context.Context, filter AuditFilter) ([]*models.AuditLogEntity, error) {
	key := cache.BuildKey("audit:list", filter.Action, filter.ActorID, filter.CursorCreatedAt, filter.CursorID, filter.Limit)
	if r.c != nil && !r.inTx {
		var cached []*models.AuditLogEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		rows, err := r.q.ListAuditLogs(ctx, sqlc.ListAuditLogsParams{
			Action:          nullableFilter(filter.Action),
			ActorID:         nullableFilter(filter.ActorID),
			CursorCreatedAt: convert.StrPtrToNullStringNonEmpty(&filter.CursorCreatedAt),
			CursorID:        convert.StrPtrToNullStringNonEmpty(&filter.CursorID),
			Limit:           filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.AuditLogEntity, 0, len(rows))
		for _, row := range rows {
			res = append(res, models.AuditLogFromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.ListCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AuditLogEntity), nil
}

func (r *auditRepository) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	key := cache.BuildKey("audit:count", filter.Action, filter.ActorID)
	if r.c != nil && !r.inTx {
		var cached int64
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		total, err := r.q.CountAuditLogs(ctx, sqlc.CountAuditLogsParams{
			Action:  nullableFilter(filter.Action),
			ActorID: nullableFilter(filter.ActorID),
		})
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, total, constants.ListCacheDuration)
		}
		return total, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (r *auditRepository) ListActions(ctx context.Context) ([]string, error) {
	key := constants.CacheKeyAuditActions
	if r.c != nil && !r.inTx {
		var cached []string
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		actions, err := r.q.ListAuditActions(ctx)
		if err != nil {
			return nil, err
		}
		if actions == nil {
			actions = []string{}
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, actions, constants.ListCacheDuration)
		}
		return actions, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *auditRepository) Prune(ctx context.Context, keepDays int64) (int64, error) {
	removed, err := r.q.PruneAuditLogs(ctx, strconv.FormatInt(keepDays, 10))
	if err != nil {
		return 0, err
	}
	if r.c != nil && removed > 0 {
		_ = r.c.DelByPattern(ctx, constants.CacheKeyAuditListPattern)
		_ = r.c.DelByPattern(ctx, constants.CacheKeyAuditCountPattern)
		_ = r.c.Del(ctx, constants.CacheKeyAuditActions)
	}
	return removed, nil
}
