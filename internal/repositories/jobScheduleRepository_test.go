package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/database"
)

func TestJobScheduleRepositoryClaimsDueSchedule(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db, filepath.Join("..", "..", "db", "schema")); err != nil {
		t.Fatal(err)
	}
	repo := NewJobScheduleRepository(db, nil)
	now := time.Now().UTC().Truncate(time.Second)
	created, err := repo.Create(context.Background(), sqlc.CreateJobScheduleParams{
		ID: "schedule-1", Name: "daily", TaskType: "maintenance", IntervalMinutes: 1440,
		Enabled: 1, NextRunAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	due, err := repo.ListDue(context.Background(), now)
	if err != nil || len(due) != 1 || due[0].ID != created.ID {
		t.Fatalf("unexpected due schedules: %#v, %v", due, err)
	}
	claimed, err := repo.Claim(context.Background(), created.ID, "job-1", now, now.Add(24*time.Hour))
	if err != nil || !claimed {
		t.Fatalf("claim failed: %v", err)
	}
	claimedAgain, err := repo.Claim(context.Background(), created.ID, "job-2", now, now.Add(24*time.Hour))
	if err != nil || claimedAgain {
		t.Fatalf("duplicate claim result: %v, %v", claimedAgain, err)
	}
}
