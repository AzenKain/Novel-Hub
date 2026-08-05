package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/rs/zerolog/log"
)

var ErrQueueStopped = errors.New("job queue stopped")

type JobFunc func(ctx context.Context, jobID string, payload string) error

type Lifecycle interface {
	Queued(ctx context.Context, job Job) error
	Running(ctx context.Context, job Job) error
	Completed(ctx context.Context, job Job) error
	Failed(ctx context.Context, job Job, err error) error
}

type Job struct {
	ID      string
	Type    string
	Payload string
}

type Queue struct {
	jobs      chan Job
	pool      pond.Pool
	wg        sync.WaitGroup
	workers   int
	ctx       context.Context
	cancel    context.CancelFunc
	handler   map[string]JobFunc
	lifecycle Lifecycle
	mu        sync.RWMutex
	stopped   bool
}

const BufferSize = 5000

func NewQueue(workers int, bufferSize ...int) *Queue {
	size := BufferSize
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		size = bufferSize[0]
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{
		jobs:    make(chan Job, size),
		pool:    pond.NewPool(workers),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
		handler: make(map[string]JobFunc),
	}
}

func (q *Queue) RegisterHandler(jobType string, handler JobFunc) {
	q.handler[jobType] = handler
}

func (q *Queue) SetLifecycle(lifecycle Lifecycle) {
	q.lifecycle = lifecycle
}

func (q *Queue) Enqueue(ctx context.Context, job Job) error {
	return q.enqueue(ctx, job, true)
}

func (q *Queue) EnqueueExisting(ctx context.Context, job Job) error {
	return q.enqueue(ctx, job, false)
}

func (q *Queue) enqueue(ctx context.Context, job Job, persist bool) error {
	q.mu.RLock()
	stopped := q.stopped
	q.mu.RUnlock()
	if stopped {
		return ErrQueueStopped
	}
	if persist && q.lifecycle != nil {
		if err := q.lifecycle.Queued(ctx, job); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		if q.lifecycle != nil {
			_ = q.lifecycle.Failed(context.Background(), job, ctx.Err())
		}
		return ctx.Err()
	case <-q.ctx.Done():
		if q.lifecycle != nil {
			_ = q.lifecycle.Failed(context.Background(), job, ErrQueueStopped)
		}
		return ErrQueueStopped
	case q.jobs <- job:
		return nil
	default:
		q.wg.Add(1)
		q.pool.Submit(func() {
			defer q.wg.Done()
			q.process(job)
		})
		return nil
	}
}

func (q *Queue) Start() {
	q.wg.Go(func() {
		for {
			select {
			case <-q.ctx.Done():
				return
			case job, ok := <-q.jobs:
				if !ok {
					return
				}
				q.wg.Add(1)
				q.pool.Submit(func() {
					defer q.wg.Done()
					q.process(job)
				})
			}
		}
	})
}

func (q *Queue) process(job Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Str("job_id", job.ID).Msg("Worker job panicked")
			if q.lifecycle != nil {
				_ = q.lifecycle.Failed(q.ctx, job, fmt.Errorf("panic: %v", r))
			}
		}
	}()
	handler, ok := q.handler[job.Type]
	if !ok {
		err := errors.New("no handler found for job type")
		if q.lifecycle != nil {
			_ = q.lifecycle.Failed(q.ctx, job, err)
		}
		log.Error().Str("type", job.Type).Msg("No handler found for job type")
		return
	}
	if q.lifecycle != nil {
		_ = q.lifecycle.Running(q.ctx, job)
	}
	start := time.Now()
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		ctx, cancel := context.WithTimeout(q.ctx, time.Hour)
		err := handler(ctx, job.ID, job.Payload)
		cancel()
		if err == nil {
			if q.lifecycle != nil {
				_ = q.lifecycle.Completed(q.ctx, job)
			}
			log.Info().Str("job_id", job.ID).Dur("duration", time.Since(start)).Msg("Job completed successfully")
			return
		}
		lastErr = err
		log.Warn().Err(err).Int("attempt", attempt).Str("job_id", job.ID).Msg("Job failed, retrying...")
		if attempt < 3 {
			select {
			case <-q.ctx.Done():
				if q.lifecycle != nil {
					_ = q.lifecycle.Failed(context.Background(), job, q.ctx.Err())
				}
				return
			case <-time.After(time.Duration(attempt*500) * time.Millisecond):
			}
		}
	}
	if q.lifecycle != nil {
		_ = q.lifecycle.Failed(q.ctx, job, lastErr)
	}
	log.Error().Err(lastErr).Str("job_id", job.ID).Msg("Job permanently failed after retries")
}

func (q *Queue) Stop() {
	_ = q.StopContext(context.Background())
}

func (q *Queue) StopContext(ctx context.Context) error {
	log.Info().Msg("Stopping job queue and draining workers...")
	q.cancel()
	q.mu.Lock()
	q.stopped = true
	q.mu.Unlock()
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		if q.pool != nil {
			q.pool.Stop()
		}
		close(done)
	}()
	select {
	case <-done:
		log.Info().Msg("Job queue stopped.")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
