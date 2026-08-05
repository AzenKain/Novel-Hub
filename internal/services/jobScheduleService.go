package services

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
)

type JobScheduleService interface {
	List(ctx context.Context) ([]*response.JobScheduleResponse, error)
	Create(ctx context.Context, dto *request.UpsertJobScheduleDto) (*response.JobScheduleResponse, error)
	Update(ctx context.Context, id string, dto *request.UpsertJobScheduleDto) (*response.JobScheduleResponse, error)
	Delete(ctx context.Context, id string) error
	RunNow(ctx context.Context, id string) (*response.JobResponse, error)
}

type jobScheduleService struct {
	repo      repositories.JobScheduleRepository
	jobs      *jobService
	stop      chan struct{}
	stopOnce  sync.Once
	wake      chan struct{}
	started   bool
	startedMu sync.Mutex
}

func NewJobScheduleService(repo repositories.JobScheduleRepository, jobs *jobService) *jobScheduleService {
	return &jobScheduleService{repo: repo, jobs: jobs, stop: make(chan struct{}), wake: make(chan struct{}, 1)}
}

func (s *jobScheduleService) List(ctx context.Context) ([]*response.JobScheduleResponse, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return models.JobSchedulesToResponse(rows), nil
}

func (s *jobScheduleService) validate(dto *request.UpsertJobScheduleDto) error {
	if dto == nil || strings.TrimSpace(dto.Name) == "" {
		return apperrors.New(apperrors.ErrBadRequest, "schedule name is required")
	}
	if _, ok := s.jobs.tasks[dto.TaskType]; !ok {
		return apperrors.New(apperrors.ErrBadRequest, "unsupported task type")
	}
	if dto.IntervalMinutes < 5 || dto.IntervalMinutes > 525600 {
		return apperrors.New(apperrors.ErrBadRequest, "interval must be between 5 and 525600 minutes")
	}
	return nil
}

func schedulePayload(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func (s *jobScheduleService) Create(ctx context.Context, dto *request.UpsertJobScheduleDto) (*response.JobScheduleResponse, error) {
	if err := s.validate(dto); err != nil {
		return nil, err
	}
	now := time.Now()
	entity, err := s.repo.Create(ctx, sqlc.CreateJobScheduleParams{
		ID: uuid.Must(uuid.NewV7()).String(), Name: strings.TrimSpace(dto.Name), TaskType: dto.TaskType,
		PayloadJson: schedulePayload(dto.PayloadJSON), IntervalMinutes: dto.IntervalMinutes,
		Enabled: boolToInt(dto.Enabled), NextRunAt: now.Add(time.Duration(dto.IntervalMinutes) * time.Minute),
	})
	if err != nil {
		return nil, err
	}
	s.signal()
	return entity.ToResponse(), nil
}

func (s *jobScheduleService) Update(ctx context.Context, id string, dto *request.UpsertJobScheduleDto) (*response.JobScheduleResponse, error) {
	if err := s.validate(dto); err != nil {
		return nil, err
	}
	entity, err := s.repo.Update(ctx, sqlc.UpdateJobScheduleParams{
		Name: strings.TrimSpace(dto.Name), TaskType: dto.TaskType, PayloadJson: schedulePayload(dto.PayloadJSON),
		IntervalMinutes: dto.IntervalMinutes, Enabled: boolToInt(dto.Enabled),
		NextRunAt: time.Now().Add(time.Duration(dto.IntervalMinutes) * time.Minute), ID: id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "job schedule not found")
		}
		return nil, err
	}
	s.signal()
	return entity.ToResponse(), nil
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func (s *jobScheduleService) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil && apperrors.IsNotFound(err) {
		return apperrors.New(apperrors.ErrNotFound, "job schedule not found")
	}
	return err
}

func (s *jobScheduleService) RunNow(ctx context.Context, id string) (*response.JobResponse, error) {
	schedule, err := s.repo.Get(ctx, id)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "job schedule not found")
		}
		return nil, err
	}
	return s.jobs.Trigger(ctx, schedule.TaskType, stringValue(schedule.PayloadJSON))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *jobScheduleService) Start() {
	s.startedMu.Lock()
	if s.started {
		s.startedMu.Unlock()
		return
	}
	s.started = true
	s.startedMu.Unlock()
	go s.loop()
}

func (s *jobScheduleService) Stop() {
	s.stopOnce.Do(func() { close(s.stop) })
}

func (s *jobScheduleService) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *jobScheduleService) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.runDue()
		case <-s.wake:
			s.runDue()
		}
	}
}

func (s *jobScheduleService) runDue() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now()
	due, err := s.repo.ListDue(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("failed to list due job schedules")
		return
	}
	for _, schedule := range due {
		jobID := uuid.Must(uuid.NewV7()).String()
		nextRun := now.Add(time.Duration(schedule.IntervalMinutes) * time.Minute)
		claimed, err := s.repo.Claim(ctx, schedule.ID, jobID, now, nextRun)
		if err != nil || !claimed {
			continue
		}
		if _, err := s.jobs.triggerWithID(ctx, jobID, schedule.TaskType, stringValue(schedule.PayloadJSON)); err != nil {
			_ = s.repo.ReleaseClaim(ctx, schedule.ID, jobID, time.Now().Add(time.Minute))
			log.Error().Err(err).Str("schedule_id", schedule.ID).Msg("failed to enqueue scheduled job")
		}
	}
}
