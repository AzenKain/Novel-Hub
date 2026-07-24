package database

import (
	"os"
	"path/filepath"
	"testing"

	"novelhub/pkg/jsonx"
)

func TestApplyPendingRestoreSwapsDatabaseAndKeepsSafetyCopy(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "active.db")
	if err := os.WriteFile(dbPath, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(dataDir, ".restore-staging")
	stagedDB := filepath.Join(staging, "database", "novelhub.db")
	if err := os.MkdirAll(filepath.Dir(stagedDB), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stagedDB, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, err := restoreFileSHA256(stagedDB)
	if err != nil {
		t.Fatal(err)
	}
	marker, err := jsonx.Marshal(pendingRestore{StagingDir: staging, DatabaseHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, ".pending-restore.json"), marker, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("SQLITE_DB_PATH", dbPath)
	if err := ApplyPendingRestore(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dbPath)
	if err != nil || string(data) != "new" {
		t.Fatalf("restored database = %q, %v", data, err)
	}
	matches, err := filepath.Glob(filepath.Join(dataDir, "restore-safety", "*", "active.db"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("safety backup missing: %#v, %v", matches, err)
	}
	old, err := os.ReadFile(matches[0])
	if err != nil || string(old) != "old" {
		t.Fatalf("safety database = %q, %v", old, err)
	}
}
