package repositories

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/pkg/cache"
)

// A repository handed a *sql.Tx must not answer reads from the RAM cache: the cache holds pre-transaction state, so a read-modify-write inside the transaction would compute from a value the transaction itself already superseded.
func TestCachedReadsInsideTransactionSeeTransactionState(t *testing.T) {
	db := newFeatureCursorTestDB(t)
	ctx := context.Background()
	c := cache.NewRamCache()
	repo := NewFeatureRepository(db, c)

	percent := 10.0
	if _, err := repo.UpsertReadingProgress(ctx, &models.ReadingProgressEntity{
		UserID: "user-1", BookID: "book-1", ChapterID: "ch-1", ChapterTitle: "One",
		ProgressPercent: &percent, OpenedCount: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetReadingProgress(ctx, "user-1", "book-1"); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := repo.WithTx(tx)

	if _, err := tx.ExecContext(ctx,
		`UPDATE reading_progress SET opened_count = opened_count + 1 WHERE user_id='user-1' AND book_id='book-1'`); err != nil {
		t.Fatal(err)
	}

	got, err := txRepo.GetReadingProgress(ctx, "user-1", "book-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("no reading progress inside the transaction")
	}
	if got.OpenedCount != 2 {
		t.Fatalf("opened_count = %d inside the transaction, want 2 (served from pre-transaction cache)", got.OpenedCount)
	}
}
