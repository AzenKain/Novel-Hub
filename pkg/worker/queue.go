package worker

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type JobFunc func(ctx context.Context, jobID string, payload string) error

type Job struct {
	ID      string
	Type    string
	Payload string
}

type Queue struct {
	jobs    chan Job
	wg      sync.WaitGroup
	workers int
	ctx     context.Context
	cancel  context.CancelFunc
	handler map[string]JobFunc
}

func NewQueue(workers int) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{
		jobs:    make(chan Job, 1000),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
		handler: make(map[string]JobFunc),
	}
}

func (q *Queue) RegisterHandler(jobType string, handler JobFunc) {
	q.handler[jobType] = handler
}

func (q *Queue) Enqueue(job Job) {
	select {
	case q.jobs <- job:
	default:
		log.Warn().Str("job_id", job.ID).Msg("Job queue is full, dropping job")
	}
}

func (q *Queue) Start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go func(workerID int) {
			defer q.wg.Done()
			for job := range q.jobs {
				q.process(job)
			}
		}(i)
	}
}

func (q *Queue) process(job Job) {
	handler, ok := q.handler[job.Type]
	if !ok {
		log.Error().Str("type", job.Type).Msg("No handler found for job type")
		return
	}

	start := time.Now()
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(q.ctx, 1*time.Hour)
		err := handler(ctx, job.ID, job.Payload)
		cancel()

		if err == nil {
			log.Info().Str("job_id", job.ID).Dur("duration", time.Since(start)).Msg("Job completed successfully")
			return
		}

		lastErr = err
		log.Warn().Err(err).Int("attempt", attempt).Str("job_id", job.ID).Msg("Job failed, retrying...")
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt*500) * time.Millisecond)
		}
	}

	log.Error().Err(lastErr).Str("job_id", job.ID).Msg("Job permanently failed after retries")
}

func (q *Queue) Stop() {
	log.Info().Msg("Stopping job queue and draining workers...")
	close(q.jobs)
	q.wg.Wait()
	q.cancel()
	log.Info().Msg("Job queue stopped.")
}
