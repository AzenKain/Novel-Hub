package services

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/apperrors"
)

// A book that has never been opened must produce ErrNotFound, not a nil entity with no error: the controller maps ErrNotFound to 404, and Kobo's readingState branches on it to decide whether the device gets a ReadingState at all.
func TestGetReadingProgressIsNotFoundBeforeTheBookIsOpened(t *testing.T) {
	svc, _ := newActivityService(t)

	_, err := svc.GetReadingProgress(context.Background(), "user", "book")
	if err == nil {
		t.Fatal("progress for an unopened book returned no error; every caller treats that as real data")
	}
	if !apperrors.IsNotFound(err) {
		t.Fatalf("error = %v, want not-found", err)
	}
}
