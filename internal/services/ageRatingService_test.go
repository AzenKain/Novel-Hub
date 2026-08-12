package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func newTestAgeRatingService(t *testing.T) (AgeRatingService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "age_rating_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	repo := repositories.NewAgeRatingRepository(db, ramCache)
	svc := NewAgeRatingService(repo)

	return svc, db
}

func TestAgeRatingHierarchy(t *testing.T) {
	svc, _ := newTestAgeRatingService(t)

	if !svc.IsAgeAllowed(constants.AgeRatingG, constants.AgeRatingPG13) {
		t.Fatal("G rating should be allowed for PG-13 max rating")
	}
	if !svc.IsAgeAllowed(constants.AgeRatingPG13, constants.AgeRatingR18) {
		t.Fatal("PG-13 rating should be allowed for R18+ max rating")
	}
	if svc.IsAgeAllowed(constants.AgeRatingR18, constants.AgeRatingPG13) {
		t.Fatal("R18+ rating should NOT be allowed for PG-13 max rating")
	}
	if svc.IsAgeAllowed(constants.AgeRatingR17, constants.AgeRatingG) {
		t.Fatal("R17+ rating should NOT be allowed for G max rating")
	}
}

func TestKidsMode6DigitPinLifecycle(t *testing.T) {
	svc, db := newTestAgeRatingService(t)
	ctx := context.Background()

	// 1. Insert test user
	if _, err := db.Exec(`INSERT INTO users (id, email, full_name) VALUES ('user-kids-1', 'kids@example.com', 'Kids User')`); err != nil {
		t.Fatal(err)
	}

	// 2. Reject non-6-digit PINs
	err4Digit := svc.SetKidsModePin(ctx, "user-kids-1", &request.SetKidsModePinDto{Pin: "1234"})
	if err4Digit == nil {
		t.Fatal("4-digit PIN should be rejected, exactly 6 digits required")
	}
	errAlpha := svc.SetKidsModePin(ctx, "user-kids-1", &request.SetKidsModePinDto{Pin: "12345a"})
	if errAlpha == nil {
		t.Fatal("Alpha-numeric PIN should be rejected, exactly 6 numeric digits required")
	}

	// 3. Set valid 6-digit PIN (bcrypt hashed)
	if err := svc.SetKidsModePin(ctx, "user-kids-1", &request.SetKidsModePinDto{Pin: "849102"}); err != nil {
		t.Fatalf("SetKidsModePin 6-digit failed: %v", err)
	}

	// 4. Enable Kids Mode
	if err := svc.ToggleKidsMode(ctx, "user-kids-1", &request.ToggleKidsModeDto{Enable: true}); err != nil {
		t.Fatalf("ToggleKidsMode enable failed: %v", err)
	}

	info, err := svc.GetKidsModeInfo(ctx, "user-kids-1")
	if err != nil {
		t.Fatalf("GetKidsModeInfo failed: %v", err)
	}
	if !info.IsKidsMode {
		t.Fatal("Expected IsKidsMode = true")
	}

	// 5. Attempt exit Kids Mode with incorrect 6-digit PIN -> Error
	errWrongPin := svc.ToggleKidsMode(ctx, "user-kids-1", &request.ToggleKidsModeDto{Enable: false, Pin: "111111"})
	if errWrongPin == nil {
		t.Fatal("Exiting Kids Mode with incorrect PIN should return error")
	}

	// 6. Exit Kids Mode with correct 6-digit PIN -> Success
	if err := svc.ToggleKidsMode(ctx, "user-kids-1", &request.ToggleKidsModeDto{Enable: false, Pin: "849102"}); err != nil {
		t.Fatalf("Exiting Kids Mode with correct PIN failed: %v", err)
	}

	infoAfter, err := svc.GetKidsModeInfo(ctx, "user-kids-1")
	if err != nil {
		t.Fatalf("GetKidsModeInfo after toggle failed: %v", err)
	}
	if infoAfter.IsKidsMode {
		t.Fatal("Expected IsKidsMode = false after disabling")
	}
}

func TestContentWarningsAndAgeRating(t *testing.T) {
	svc, db := newTestAgeRatingService(t)
	ctx := context.Background()

	// 1. Check seeded content warnings via Cache-by-IDs
	cws, err := svc.GetContentWarnings(ctx)
	if err != nil {
		t.Fatalf("GetContentWarnings failed: %v", err)
	}
	if len(cws) == 0 {
		t.Fatal("Expected non-empty content warnings list")
	}

	// 2. Create test book
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main Lib')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title) VALUES ('book-1', 'lib-1', 'Action Manga')`); err != nil {
		t.Fatal(err)
	}

	// 3. Update book age rating & content warnings
	errUpdate := svc.UpdateBookAgeRating(ctx, "book-1", &request.UpdateBookAgeRatingDto{
		AgeRating:          constants.AgeRatingR17,
		ContentWarningIDs: []string{"cw-violence", "cw-language"},
	})
	if errUpdate != nil {
		t.Fatalf("UpdateBookAgeRating failed: %v", errUpdate)
	}

	bookCWs, err := svc.GetBookContentWarnings(ctx, "book-1")
	if err != nil {
		t.Fatalf("GetBookContentWarnings failed: %v", err)
	}
	if len(bookCWs) != 2 {
		t.Fatalf("Expected 2 content warnings attached to book, got %d", len(bookCWs))
	}
}
