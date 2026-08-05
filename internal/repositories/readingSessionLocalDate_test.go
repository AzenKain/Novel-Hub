package repositories

import (
	"context"
	"testing"

	"novelhub/internal/gen/sqlc"
)

func TestReadingSessionBucketsOnTheClientDate(t *testing.T) {
	db := newFeatureCursorTestDB(t)
	ctx := context.Background()
	repo := NewFeatureRepository(db, nil)

	record := func(id string, date string, words int64) {
		t.Helper()
		if _, err := repo.UpsertReadingSession(ctx, sqlc.UpsertReadingSessionParams{
			ID:              id,
			UserID:          "user-1",
			BookID:          "book-1",
			SessionDate:     date,
			DurationSeconds: 60,
			WordsRead:       words,
		}); err != nil {
			t.Fatalf("record %s: %v", id, err)
		}
	}

	record("session-1", "2026-08-04", 100)
	record("session-2", "2026-08-04", 250)
	record("session-3", "2026-08-05", 7)

	rows, err := repo.GetReadingHeatmap(ctx, "user-1")
	if err != nil {
		t.Fatalf("heatmap: %v", err)
	}
	got := make(map[string]int64, len(rows))
	for _, row := range rows {
		got[row.Date.Format("2006-01-02")] = row.WordsRead
	}
	if len(got) != 2 {
		t.Fatalf("want one cell per client date, got %v", got)
	}
	if got["2026-08-04"] != 350 {
		t.Errorf("same-day sessions did not merge: %v", got)
	}
	if got["2026-08-05"] != 7 {
		t.Errorf("next day was swallowed: %v", got)
	}
}
