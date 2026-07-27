package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/worker"
)

type JobService interface {
	GetJob(ctx context.Context, id string) (*response.JobResponse, error)
	ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*response.JobResponse, int64, error)
	ListTasks() []*response.JobTaskResponse
	Trigger(ctx context.Context, jobType, payload string) (*response.JobResponse, error)
	Recover(ctx context.Context) error
}

type jobService struct {
	repo  repositories.JobRepository
	queue *worker.Queue
	tasks map[string]string
}

func NewJobService(repo repositories.JobRepository, queue *worker.Queue) *jobService {
	return &jobService{
		repo:  repo,
		queue: queue,
		tasks: map[string]string{
			"maintenance":           "Run full library maintenance",
			"scan_library_inbox":    "Scan library inbox folders for new files",
			"clean_empty_book_dirs": "Remove empty managed book directories",
			"clean_orphan_uploads":  "Remove stale upload chunks",
			"database_health_check": "Check database connectivity and schema access",
			"database_backup":       "Create a database backup",
			"database_books_backup": "Create a database and books backup",
		},
	}
}

func (s *jobService) GetJob(ctx context.Context, id string) (*response.JobResponse, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.New(apperrors.ErrNotFound, "job not found")
		}
		return nil, err
	}
	return job.ToResponse(), nil
}

func (s *jobService) ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*response.JobResponse, int64, error) {
	jobs, total, err := s.repo.ListJobs(ctx, status, jobType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return models.JobEntitiesToResponse(jobs), total, nil
}

func (s *jobService) ListTasks() []*response.JobTaskResponse {
	order := []string{
		"maintenance",
		"clean_empty_book_dirs",
		"clean_orphan_uploads",
		"database_health_check",
		"database_backup",
		"database_books_backup",
	}
	out := make([]*response.JobTaskResponse, 0, len(order))
	for _, taskType := range order {
		out = append(out, &response.JobTaskResponse{Type: taskType, Description: s.tasks[taskType]})
	}
	return out
}

func (s *jobService) Trigger(ctx context.Context, jobType, payload string) (*response.JobResponse, error) {
	jobType = strings.TrimSpace(jobType)
	if _, ok := s.tasks[jobType]; !ok {
		return nil, apperrors.New(apperrors.ErrBadRequest, "unsupported job type")
	}
	return s.triggerWithID(ctx, uuid.Must(uuid.NewV7()).String(), jobType, payload)
}

func (s *jobService) triggerWithID(ctx context.Context, id, jobType, payload string) (*response.JobResponse, error) {
	if _, ok := s.tasks[jobType]; !ok {
		return nil, fmt.Errorf("unsupported job type")
	}
	job := worker.Job{ID: id, Type: jobType, Payload: payload}
	if err := s.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}
	entity, err := s.repo.GetJob(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *jobService) Recover(ctx context.Context) error {
	if err := s.repo.MarkRunningJobsInterrupted(ctx); err != nil {
		return err
	}
	pending, err := s.repo.ListUnfinishedJobs(ctx, 1000, 0)
	if err != nil {
		return err
	}
	for _, job := range pending {
		if job == nil || job.Status == nil || *job.Status != "pending" {
			continue
		}
		if err := s.queue.EnqueueExisting(ctx, worker.Job{ID: job.ID, Type: job.Type, Payload: stringValue(job.PayloadJSON)}); err != nil {
			_, _ = s.repo.UpdateJobStatus(ctx, job.ID, "failed", "failed to recover pending job: "+err.Error())
		}
	}
	return s.repo.PruneFinishedJobs(ctx, 1000)
}

func (s *jobService) Queued(ctx context.Context, job worker.Job) error {
	_, err := s.repo.CreateJob(ctx, job.ID, job.Type, "pending", job.Payload)
	return err
}

func (s *jobService) Running(ctx context.Context, job worker.Job) error {
	_, err := s.repo.UpdateJobStatus(ctx, job.ID, "running", "")
	return err
}

func (s *jobService) Completed(ctx context.Context, job worker.Job) error {
	_, err := s.repo.UpdateJobStatus(ctx, job.ID, "completed", "")
	return err
}

func (s *jobService) Failed(ctx context.Context, job worker.Job, jobErr error) error {
	message := "job failed"
	if jobErr != nil {
		message = jobErr.Error()
	}
	_, err := s.repo.UpdateJobStatus(ctx, job.ID, "failed", message)
	return err
}

var _ worker.Lifecycle = (*jobService)(nil)
