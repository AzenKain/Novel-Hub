package apperrors

import (
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"novelhub/pkg/worker"
)

// A full queue is transient. Without a case for it the switch falls through to 500, which tells the
// client the server is broken rather than busy, and 500 is the one status no client retries.
func TestQueueFullIsRetryable(t *testing.T) {
	app := fiber.New()
	app.Get("/x", func(c fiber.Ctx) error {
		return HandleError(c, fmt.Errorf("queue post-processing: %w", worker.ErrQueueFull))
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 503: %s", resp.StatusCode, body)
	}
}
