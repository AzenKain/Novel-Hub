package repositories

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type JobScheduleRepository interface {
	Get(ctx context.Context, id string) (*models.JobScheduleEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.JobScheduleEntity, error)
	List(ctx context.Context) ([]*models.JobScheduleEntity, error)
	Create(ctx context.Context, params sqlc.CreateJobScheduleParams) (*models.JobScheduleEntity, error)
	Update(ctx context.Context, params sqlc.UpdateJobScheduleParams) (*models.JobScheduleEntity, error)
	Delete(ctx context.Context, id string) error
	ListDue(ctx context.Context, now time.Time) ([]*models.JobScheduleEntity, error)
	Claim(ctx context.Context, id, jobID string, now, nextRunAt time.Time) (bool, error)
	ReleaseClaim(ctx context.Context, id, jobID string, retryAt time.Time) error
	WithTx(tx *sql.Tx) JobScheduleRepository
}

type jobScheduleRepository struct {
	q   *sqlc.Queries
	c   cache.Cache
	sfg *singleflight.Group
}

func NewJobScheduleRepository(db sqlc.DBTX, c cache.Cache) JobScheduleRepository {
	return &jobScheduleRepository{q: sqlc.New(db), c: c, sfg: &singleflight.Group{}}
}

func (r *jobScheduleRepository) WithTx(tx *sql.Tx) JobScheduleRepository {
	if tx == nil {
		return r
	}
	return &jobScheduleRepository{q: r.q.WithTx(tx), c: r.c, sfg: r.sfg}
}

func (r *jobScheduleRepository) Get(ctx context.Context, id string) (*models.JobScheduleEntity, error) {
	key := cache.BuildKey("job_schedule", "id", id)
	if r.c != nil {
		var entity models.JobScheduleEntity
		if err := r.c.Get(ctx, key, &entity); err == nil {
			return &entity, nil
		}
	}
	value, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetJobSchedule(ctx, id)
		if err != nil {
			return nil, err
		}
		entity := (&models.JobScheduleEntity{}).FromSqlc(row)
		_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.JobScheduleEntity), nil
}

func (r *jobScheduleRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.JobScheduleEntity, error) {
	if len(ids) == 0 {
		return []*models.JobScheduleEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("job_schedule", "id", id)
	}

	schedules := make([]*models.JobScheduleEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var schedule models.JobScheduleEntity
				if err := jsonx.Unmarshal(bytes, &schedule); err == nil {
					schedules = append(schedules, &schedule)
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
		rows, err := r.q.GetJobSchedulesByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[string]*models.JobScheduleEntity)
		for _, row := range rows {
			s := (&models.JobScheduleEntity{}).FromSqlc(row)
			missingMap[s.ID] = s
			schedules = append(schedules, s)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if s, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = s
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	scheduleMap := make(map[string]*models.JobScheduleEntity)
	for _, s := range schedules {
		scheduleMap[s.ID] = s
	}
	ordered := make([]*models.JobScheduleEntity, 0, len(ids))
	for _, id := range ids {
		if s, ok := scheduleMap[id]; ok {
			ordered = append(ordered, s)
		}
	}

	return ordered, nil
}

func (r *jobScheduleRepository) List(ctx context.Context) ([]*models.JobScheduleEntity, error) {
	key := constants.CacheKeyJobScheduleList
	if r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		dbIds, err := r.q.ListJobScheduleIDs(ctx)
		if err != nil {
			return nil, err
		}
		if dbIds == nil {
			dbIds = []string{}
		}
		_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
		return dbIds, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByIDs(ctx, v.([]string))
}

func (r *jobScheduleRepository) Create(ctx context.Context, params sqlc.CreateJobScheduleParams) (*models.JobScheduleEntity, error) {
	row, err := r.q.CreateJobSchedule(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyJobScheduleList)
	}
	return (&models.JobScheduleEntity{}).FromSqlc(row), nil
}

func (r *jobScheduleRepository) Update(ctx context.Context, params sqlc.UpdateJobScheduleParams) (*models.JobScheduleEntity, error) {
	row, err := r.q.UpdateJobSchedule(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("job_schedule", "id", params.ID), constants.CacheKeyJobScheduleList)
	}
	return (&models.JobScheduleEntity{}).FromSqlc(row), nil
}

func (r *jobScheduleRepository) Delete(ctx context.Context, id string) error {
	if err := r.q.DeleteJobSchedule(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("job_schedule", "id", id), constants.CacheKeyJobScheduleList)
	}
	return nil
}

func (r *jobScheduleRepository) ListDue(ctx context.Context, now time.Time) ([]*models.JobScheduleEntity, error) {
	ids, err := r.q.ListDueJobScheduleIDs(ctx, now)
	if err != nil {
		return nil, err
	}
	return r.GetByIDs(ctx, ids)
}

func (r *jobScheduleRepository) Claim(ctx context.Context, id, jobID string, now, nextRunAt time.Time) (bool, error) {
	rows, err := r.q.ClaimJobSchedule(ctx, sqlc.ClaimJobScheduleParams{
		Now:       sql.NullTime{Time: now, Valid: true},
		JobID:     sql.NullString{String: jobID, Valid: true},
		NextRunAt: nextRunAt,
		ID:        id,
	})
	if err == nil && rows > 0 {
		if r.c != nil {
			_ = r.c.Del(ctx, cache.BuildKey("job_schedule", "id", id), constants.CacheKeyJobScheduleList)
		}
	}
	return rows > 0, err
}

func (r *jobScheduleRepository) ReleaseClaim(ctx context.Context, id, jobID string, retryAt time.Time) error {
	_, err := r.q.ReleaseJobScheduleClaim(ctx, sqlc.ReleaseJobScheduleClaimParams{
		RetryAt: retryAt,
		ID:      id,
		JobID:   sql.NullString{String: jobID, Valid: true},
	})
	if err == nil {
		if r.c != nil {
			_ = r.c.Del(ctx, cache.BuildKey("job_schedule", "id", id), constants.CacheKeyJobScheduleList)
		}
	}
	return err
}
