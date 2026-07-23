package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var ErrQueueStopped = errors.New("job queue stopped")

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
	mu      sync.RWMutex
	stopped bool
}

func NewQueue(workers int) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{jobs: make(chan Job, 1000), workers: workers, ctx: ctx, cancel: cancel, handler: make(map[string]JobFunc)}
}

func (q *Queue) RegisterHandler(jobType string, handler JobFunc) { q.handler[jobType] = handler }

func (q *Queue) Enqueue(ctx context.Context, job Job) error {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.stopped {
		return ErrQueueStopped
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.jobs <- job:
		return nil
	}
}

func (q *Queue) Start() {
	for range q.workers {
		q.wg.Go(func() {
			for job := range q.jobs {
				q.process(job)
			}
		})
	}
}

func (q *Queue) process(job Job) {
	handler, ok := q.handler[job.Type]
	if !ok {
		log.Error().Str("type", job.Type).Msg("No handler found for job type")
		return
	}
	start := time.Now()
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		ctx, cancel := context.WithTimeout(q.ctx, time.Hour)
		err := handler(ctx, job.ID, job.Payload)
		cancel()
		if err == nil {
			log.Info().Str("job_id", job.ID).Dur("duration", time.Since(start)).Msg("Job completed successfully")
			return
		}
		lastErr = err
		log.Warn().Err(err).Int("attempt", attempt).Str("job_id", job.ID).Msg("Job failed, retrying...")
		if attempt < 3 {
			select {
			case <-q.ctx.Done():
				return
			case <-time.After(time.Duration(attempt*500) * time.Millisecond):
			}
		}
	}
	log.Error().Err(lastErr).Str("job_id", job.ID).Msg("Job permanently failed after retries")
}

func (q *Queue) Stop() {
	log.Info().Msg("Stopping job queue and draining workers...")
	q.mu.Lock()
	if !q.stopped {
		q.stopped = true
		close(q.jobs)
	}
	q.mu.Unlock()
	q.wg.Wait()
	q.cancel()
	log.Info().Msg("Job queue stopped.")
}
