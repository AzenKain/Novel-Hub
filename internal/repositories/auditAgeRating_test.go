package repositories

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
)

// TestAuditAgeRatingUpdateNotAtomic proves task T5.1: UpdateBookAgeRatingAndWarnings (ageRatingRepository.go:148-174) runs UpdateBookAgeRating + ClearBookContentWarnings + N x AddBookContentWarning with no transaction wrapping, even though the interface exposes WithTx.
func TestAuditAgeRatingUpdateNotAtomic(t *testing.T) {
	db := newAuditRepoTestDB(t)
	ctx := context.Background()
	seedAuditLibraryAndBooks(t, db, "b1")

	if _, err := db.Exec(`INSERT OR IGNORE INTO content_warnings (id, name) VALUES ('cw-violence', 'Violence')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO book_content_warnings (book_id, warning_id) VALUES ('b1', 'cw-violence')`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`CREATE TRIGGER audit_fail_clear BEFORE DELETE ON book_content_warnings BEGIN SELECT RAISE(ABORT, 'audit fail'); END`); err != nil {
		t.Fatal(err)
	}

	repo := NewAgeRatingRepository(db, cache.NewRamCache())
	err := repo.UpdateBookAgeRatingAndWarnings(ctx, "b1", "R18", []string{"cw-violence"})
	if err == nil {
		t.Fatal("setup broken: clear step did not fail; trigger not firing")
	}

	var ageRating string
	if err := db.QueryRow(`SELECT age_rating FROM books WHERE id = 'b1'`).Scan(&ageRating); err != nil {
		t.Fatal(err)
	}
	if ageRating != "R18" {
		t.Fatalf("no partial write to prove: age_rating = %q after failed update (transaction rollback may exist)", ageRating)
	}
}
