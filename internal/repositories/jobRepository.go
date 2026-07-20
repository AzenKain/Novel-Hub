package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
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
	key := fmt.Sprintf("job:id:%s", id)
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
	rows, err := r.queries.ListUnfinishedJobs(ctx)
	if err != nil {
		return nil, err
	}
	jobs := make([]*models.JobEntity, len(rows))
	for i, row := range rows {
		jobs[i] = (&models.JobEntity{}).FromSqlc(row)
	}
	return jobs, nil
}
