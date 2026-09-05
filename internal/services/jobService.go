package services

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/worker"
)

const keepFinishedJobs = 1000

const recoverBatchSize = worker.BufferSize

type JobService interface {
	GetJob(ctx context.Context, id string) (*response.JobResponse, error)
	ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*response.JobResponse, int64, error)
	ListTasks() []*response.JobTaskResponse
	Trigger(ctx context.Context, jobType, payload string) (*response.JobResponse, error)
	PruneFinishedJobs(ctx context.Context) error
	Recover(ctx context.Context) error
	SetWebhookService(webhook WebhookService)
}

type jobService struct {
	repo           repositories.JobRepository
	queue          *worker.Queue
	tasks          map[string]string
	webhookService WebhookService
}

func (s *jobService) SetWebhookService(webhook WebhookService) {
	s.webhookService = webhook
}

func NewJobService(repo repositories.JobRepository, queue *worker.Queue) *jobService {
	return &jobService{
		repo:  repo,
		queue: queue,
		tasks: map[string]string{
			"maintenance":           "Run full library maintenance",
			"scan_library_inbox":    "Scan library inbox folders for new files",
			"scan_metadata_enrich":  "Scan and enrich metadata from online APIs (AniList, OpenLibrary, Google Books)",
			"repair_books":          "Auto-diagnose and repair corrupted EPUB book files in libraries",
			"clean_empty_book_dirs": "Remove empty managed book directories",
			"clean_orphan_uploads":  "Remove stale upload chunks",
			"database_health_check": "Check database connectivity and schema access",
			"database_backup":       "Create a database backup",
			"database_books_backup": "Create a database and books backup",
			"prune_finished_jobs":   "Delete old completed and failed job rows",
			"prune_audit_logs":      "Delete audit log entries older than 90 days",
			"merge_audio":           "Merge multiple audio files into a chaptered M4B audiobook",
			"convert_book":          "Convert a book file to another format (epub, fb2, txt, docx, cbz)",
			"podcast_refresh":       "Refresh podcast feeds and download new episodes (auto-download only)",
			"podcast_download":      "Download a podcast episode as a library book",
		},
	}
}

func (s *jobService) GetJob(ctx context.Context, id string) (*response.JobResponse, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		if apperrors.IsNotFound(err) {
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

// order is display order only; anything in tasks but missing here is appended, so a new task can never be silently unreachable from the admin UI the way scan_library_inbox was.
func (s *jobService) ListTasks() []*response.JobTaskResponse {
	order := []string{
		"maintenance",
		"scan_library_inbox",
		"scan_metadata_enrich",
		"repair_books",
		"clean_empty_book_dirs",
		"clean_orphan_uploads",
		"prune_finished_jobs",
		"prune_audit_logs",
		"database_health_check",
		"database_backup",
		"database_books_backup",
	}
	out := make([]*response.JobTaskResponse, 0, len(s.tasks))
	listed := make(map[string]bool, len(s.tasks))
	for _, taskType := range order {
		description, ok := s.tasks[taskType]
		if !ok {
			continue
		}
		listed[taskType] = true
		out = append(out, &response.JobTaskResponse{Type: taskType, Description: description})
	}
	for _, taskType := range slices.Sorted(maps.Keys(s.tasks)) {
		if !listed[taskType] {
			out = append(out, &response.JobTaskResponse{Type: taskType, Description: s.tasks[taskType]})
		}
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
	pending, err := s.repo.ListUnfinishedJobs(ctx, recoverBatchSize, 0)
	if err != nil {
		return err
	}
	requeued := 0
	for _, job := range pending {
		if job == nil || job.Status == nil || *job.Status != "pending" {
			continue
		}
		if err := s.queue.EnqueueExisting(ctx, worker.Job{ID: job.ID, Type: job.Type, Payload: stringValue(job.PayloadJSON)}); err != nil {
			_, _ = s.repo.UpdateJobStatus(ctx, job.ID, "failed", "failed to recover pending job: "+err.Error())
			continue
		}
		requeued++
	}
	if len(pending) == recoverBatchSize {
		remaining, err := s.repo.CountUnfinishedJobs(ctx)
		if err == nil && remaining > int64(len(pending)) {
			log.Warn().
				Int("requeued", requeued).
				Int64("still_pending", remaining-int64(len(pending))).
				Msg("Job recovery hit its batch limit; remaining jobs resume on next restart")
		}
	}
	return s.PruneFinishedJobs(ctx)
}

func (s *jobService) PruneFinishedJobs(ctx context.Context) error {
	return s.repo.PruneFinishedJobs(ctx, keepFinishedJobs)
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
	if s.webhookService != nil && job.Type != "webhook.dispatch" {
		s.webhookService.DispatchEvent(ctx, "job.failed", map[string]any{
			"job_id":   job.ID,
			"job_type": job.Type,
			"error":    message,
		})
	}
	return err
}

var _ worker.Lifecycle = (*jobService)(nil)
