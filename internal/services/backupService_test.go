package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBackupServiceCreatesDatabaseArchive(t *testing.T) {
	dataDir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "novelhub.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE sample (id INTEGER PRIMARY KEY, value TEXT); INSERT INTO sample(value) VALUES ('ok')"); err != nil {
		t.Fatal(err)
	}

	service := NewBackupService(db, dataDir, false, nil, nil)
	backup, err := service.Create(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	path, err := service.Path(backup.Name)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := readBackupManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != backupFormatVersion || manifest.IncludeBooks {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("invalid archive: %v", err)
	}
}
