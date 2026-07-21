package services

import (
	"context"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
)

type JobService interface {
	GetJob(ctx context.Context, id string) (*response.JobResponse, error)
}

type jobService struct {
	repo repositories.JobRepository
}

func NewJobService(repo repositories.JobRepository) JobService {
	return &jobService{repo: repo}
}

func (s *jobService) GetJob(ctx context.Context, id string) (*response.JobResponse, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	return job.ToResponse(), nil
}
