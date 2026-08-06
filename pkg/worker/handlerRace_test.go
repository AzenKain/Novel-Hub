package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// Only -race catches this: the worker reads q.handler and q.lifecycle from its own goroutine while
// RegisterHandler/SetLifecycle write them, and nothing in the API forces registration before Start.
func TestRegisterHandlerConcurrentWithWorker(t *testing.T) {
	q := NewQueue(2)
	q.RegisterHandler("a", func(ctx context.Context, jobID, payload string) error { return nil })
	q.Start()
	t.Cleanup(q.Stop)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if err := q.Enqueue(context.Background(), Job{ID: fmt.Sprintf("j-%d", i), Type: "a"}); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			q.RegisterHandler(fmt.Sprintf("late-%d", i), func(ctx context.Context, jobID, payload string) error { return nil })
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			q.SetLifecycle(nil)
		}
	}()
	wg.Wait()
}
