package services

import (
	"context"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
)

type JobService interface {
	GetJob(ctx context.Context, id string) (*models.JobEntity, error)
}

type jobService struct {
	repo repositories.JobRepository
}

func NewJobService(repo repositories.JobRepository) JobService {
	return &jobService{repo: repo}
}

func (s *jobService) GetJob(ctx context.Context, id string) (*models.JobEntity, error) {
	return s.repo.GetJob(ctx, id)
}
