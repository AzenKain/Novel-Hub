package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type JobRepository interface {
	GetJob(ctx context.Context, id string) (*models.JobEntity, error)
	GetJobsByIDs(ctx context.Context, ids []string) ([]*models.JobEntity, error)
	ListUnfinishedJobs(ctx context.Context, limit, offset int64) ([]*models.JobEntity, error)
	CreateJob(ctx context.Context, id, jobType, status, payload string) (*models.JobEntity, error)
	UpdateJobStatus(ctx context.Context, id, status, errorMsg string) (*models.JobEntity, error)
	ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*models.JobEntity, int64, error)
	MarkRunningJobsInterrupted(ctx context.Context) error
	CountUnfinishedJobs(ctx context.Context) (int64, error)
	PruneFinishedJobs(ctx context.Context, keep int64) error
	WithTx(tx *sql.Tx) JobRepository
}

type jobRepository struct {
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewJobRepository(db *sql.DB, c cache.Cache) JobRepository {
	return &jobRepository{
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

func (r *jobRepository) WithTx(tx *sql.Tx) JobRepository {
	if tx == nil {
		return r
	}
	return &jobRepository{
		queries: r.queries.WithTx(tx),
		c:       r.c,
		inTx:    true,
		sfg:     r.sfg,
	}
}

func (r *jobRepository) GetJob(ctx context.Context, id string) (*models.JobEntity, error) {
	key := cache.BuildKey("job", "id", id)
	if r.c != nil && !r.inTx {
		var job models.JobEntity
		if err := r.c.Get(ctx, key, &job); err == nil {
			return &job, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetJob(ctx, id)
		if err != nil {
			return nil, err
		}
		job := (&models.JobEntity{}).FromSqlc(row)
		if r.c != nil {
			_ = r.c.Set(ctx, key, job, constants.NormalCacheDuration)
		}
		return job, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.JobEntity), nil
}

func (r *jobRepository) GetJobsByIDs(ctx context.Context, ids []string) ([]*models.JobEntity, error) {
	if len(ids) == 0 {
		return []*models.JobEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("job", "id", id)
	}

	jobs := make([]*models.JobEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var job models.JobEntity
				if err := jsonx.Unmarshal(bytes, &job); err == nil {
					jobs = append(jobs, &job)
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
		sort.Strings(missingIds)
		sfgKey := "jobs:ids:" + strings.Join(missingIds, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := queryInChunks(missingIds, func(chunk []string) ([]sqlc.Job, error) {
				return r.queries.GetJobsByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			missingMap := make(map[string]*models.JobEntity)
			for _, row := range rows {
				j := (&models.JobEntity{}).FromSqlc(row)
				missingMap[j.ID] = j
			}
			return missingMap, nil
		})
		if err != nil {
			return nil, err
		}
		missingMap := v.(map[string]*models.JobEntity)

		for _, j := range missingMap {
			jobs = append(jobs, j)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for _, missingId := range missingIds {
				if j, ok := missingMap[missingId]; ok {
					missingToCache[cache.BuildKey("job", "id", missingId)] = j
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	jobMap := make(map[string]*models.JobEntity)
	for _, j := range jobs {
		jobMap[j.ID] = j
	}
	ordered := make([]*models.JobEntity, 0, len(ids))
	for _, id := range ids {
		if j, ok := jobMap[id]; ok {
			ordered = append(ordered, j)
		}
	}

	return ordered, nil
}

func (r *jobRepository) ListUnfinishedJobs(ctx context.Context, limit, offset int64) ([]*models.JobEntity, error) {
	key := cache.BuildKey("job", "unfinished", limit, offset)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetJobsByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		dbIds, err := r.queries.ListUnfinishedJobIDs(ctx, sqlc.ListUnfinishedJobIDsParams{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		if dbIds == nil {
			dbIds = []string{}
		}
		if r.c != nil {
			_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
		}
		return dbIds, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetJobsByIDs(ctx, v.([]string))
}

func (r *jobRepository) CreateJob(ctx context.Context, id, jobType, status, payload string) (*models.JobEntity, error) {
	params := sqlc.CreateJobParams{
		ID:          id,
		Type:        jobType,
		Status:      sql.NullString{String: status, Valid: status != ""},
		PayloadJson: sql.NullString{String: payload, Valid: payload != ""},
	}
	row, err := r.queries.CreateJob(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobListPattern)
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobCountPattern)
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobUnfinishedPattern)
	}
	return (&models.JobEntity{}).FromSqlc(row), nil
}

func (r *jobRepository) UpdateJobStatus(ctx context.Context, id, status, errorMsg string) (*models.JobEntity, error) {
	row, err := r.queries.UpdateJobStatus(ctx, sqlc.UpdateJobStatusParams{
		Status:   sql.NullString{String: status, Valid: status != ""},
		ErrorMsg: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		ID:       id,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("job", "id", id))
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobListPattern)
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobCountPattern)
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobUnfinishedPattern)
	}
	return (&models.JobEntity{}).FromSqlc(row), nil
}

func (r *jobRepository) ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*models.JobEntity, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	countKey := cache.BuildKey("job", "count", status, jobType)
	var total int64
	var countFetched bool
	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, countKey, &total); err == nil {
			countFetched = true
		}
	}
	if !countFetched {
		v, err, _ := r.sfg.Do(countKey, func() (any, error) {
			t, err := r.queries.CountJobs(ctx, sqlc.CountJobsParams{Status: status, Type: jobType})
			if err != nil {
				return int64(0), err
			}
			if r.c != nil {
				_ = r.c.Set(ctx, countKey, t, constants.ListCacheDuration)
			}
			return t, nil
		})
		if err != nil {
			return nil, 0, err
		}
		total = v.(int64)
	}

	if total == 0 {
		return []*models.JobEntity{}, 0, nil
	}

	key := cache.BuildKey("job", "list", status, jobType, limit, offset)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			jobs, err := r.GetJobsByIDs(ctx, ids)
			if err != nil {
				return nil, 0, err
			}
			return jobs, total, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		dbIds, err := r.queries.ListFilteredJobIDs(ctx, sqlc.ListFilteredJobIDsParams{
			Status: status,
			Type:   jobType,
			Offset: offset,
			Limit:  limit,
		})
		if err != nil {
			return nil, err
		}
		if dbIds == nil {
			dbIds = []string{}
		}
		if r.c != nil {
			_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
		}
		return dbIds, nil
	})
	if err != nil {
		return nil, 0, err
	}
	dbIds := v.([]string)

	jobs, err := r.GetJobsByIDs(ctx, dbIds)
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r *jobRepository) MarkRunningJobsInterrupted(ctx context.Context) error {
	if err := r.queries.MarkRunningJobsInterrupted(ctx); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobAllPattern)
	}
	return nil
}

func (r *jobRepository) CountUnfinishedJobs(ctx context.Context) (int64, error) {
	return r.queries.CountUnfinishedJobs(ctx)
}

func (r *jobRepository) PruneFinishedJobs(ctx context.Context, keep int64) error {
	if _, err := r.queries.PruneFinishedJobs(ctx, keep); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, constants.CacheKeyJobAllPattern)
	}
	return nil
}
