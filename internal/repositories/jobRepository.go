package repositories

import (
	"context"
	"database/sql"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type JobRepository interface {
	GetJob(ctx context.Context, id string) (*models.JobEntity, error)
	ListUnfinishedJobs(ctx context.Context) ([]*models.JobEntity, error)
}

type jobRepository struct {
	queries *sqlc.Queries
	c       cache.Cache
}

func NewJobRepository(db *sql.DB, c cache.Cache) JobRepository {
	return &jobRepository{queries: sqlc.New(db), c: c}
}

func (r *jobRepository) GetJob(ctx context.Context, id string) (*models.JobEntity, error) {
	key := cache.BuildKey("job", "id", id)
	if r.c != nil {
		var job models.JobEntity
		if err := r.c.Get(ctx, key, &job); err == nil {
			return &job, nil
		}
	}
	row, err := r.queries.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	job := (&models.JobEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, job, constants.NormalCacheDuration)
	}
	return job, nil
}

func (r *jobRepository) ListUnfinishedJobs(ctx context.Context) ([]*models.JobEntity, error) {
	key := "job:unfinished"
	if r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getJobsByIDs(ctx, ids); ok {
				return result, nil
			}
		}
	}
	
	idRows, err := r.queries.ListUnfinishedJobIDs(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(idRows) == 0 {
		if r.c != nil {
			_ = r.c.Set(ctx, key, []string{}, constants.ListCacheDuration)
		}
		return []*models.JobEntity{}, nil
	}
	
	rows, err := r.queries.GetJobsByIDs(ctx, idRows)
	if err != nil {
		return nil, err
	}
	
	jobs := make([]*models.JobEntity, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		jobs[i] = (&models.JobEntity{}).FromSqlc(row)
		ids[i] = row.ID
	}
	
	if r.c != nil {
		_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		r.cacheJobEntities(ctx, jobs)
	}
	return jobs, nil
}

func (r *jobRepository) getJobsByIDs(ctx context.Context, ids []string) ([]*models.JobEntity, bool) {
	if len(ids) == 0 {
		return []*models.JobEntity{}, true
	}
	if r.c == nil {
		return nil, false
	}

	cacheKeys := make([]string, len(ids))
	for i, id := range ids {
		cacheKeys[i] = cache.BuildKey("job", "id", id)
	}

	cachedBytes := r.c.MGet(ctx, cacheKeys...)
	ordered := make([]*models.JobEntity, 0, len(ids))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.JobEntity
		if err := jsonx.Unmarshal(bytes, &entity); err != nil {
			return nil, false
		}
		ordered = append(ordered, &entity)
	}

	return ordered, true
}

func (r *jobRepository) cacheJobEntities(ctx context.Context, entities []*models.JobEntity) {
	if r.c == nil || len(entities) == 0 {
		return
	}
	toCache := make(map[string]any, len(entities))
	for _, entity := range entities {
		toCache[cache.BuildKey("job", "id", entity.ID)] = entity
	}
	_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
}
