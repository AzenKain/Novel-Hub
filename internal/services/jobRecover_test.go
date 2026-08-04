package services

import (
	"context"
	"testing"
	"time"

	"novelhub/pkg/worker"
)

// Recover runs at boot, before jobQueue.Start(), so nothing drains the channel yet.
// The queue buffer is 1000, so enqueueing 1000+ pending jobs blocks forever on the
// unbuffered send and the process never finishes booting. Recover must therefore never
// hand the queue more jobs than the buffer can hold.
func TestRecoverDoesNotBlockWhenQueueBufferFills(t *testing.T) {
	queue := worker.NewQueue(1) // buffer is fixed at 1000 regardless of worker count
	// deliberately NOT calling queue.Start(): this is the boot-time state

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < recoverBatchSize; i++ {
			if err := queue.EnqueueExisting(context.Background(), worker.Job{
				ID: string(rune(i)), Type: "scan",
			}); err != nil {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("enqueueing %d jobs blocked with no workers running; "+
			"recoverBatchSize must stay within the queue buffer", recoverBatchSize)
	}
}
