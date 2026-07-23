package worker

import (
	"context"
	"errors"
	"testing"
)

func TestEnqueueAfterStopReturnsError(t *testing.T) {
	queue := NewQueue(1)
	queue.Start()
	queue.Stop()
	if err := queue.Enqueue(context.Background(), Job{ID: "1", Type: "test"}); !errors.Is(err, ErrQueueStopped) {
		t.Fatalf("expected ErrQueueStopped, got %v", err)
	}
}
