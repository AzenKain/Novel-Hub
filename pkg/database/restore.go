package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"novelhub/pkg/config"
	"novelhub/pkg/jsonx"
)

type pendingRestore struct {
	StagingDir   string `json:"staging_dir"`
	IncludeBooks bool   `json:"include_books"`
	DatabaseHash string `json:"database_hash"`
}

func ApplyPendingRestore() error {
	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	marker := filepath.Join(dataDir, ".pending-restore.json")
	data, err := os.ReadFile(marker)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var pending pendingRestore
	if err := jsonx.Unmarshal(data, &pending); err != nil {
		return fmt.Errorf("read pending restore: %w", err)
	}
	expectedStaging, _ := filepath.Abs(filepath.Join(dataDir, ".restore-staging"))
	actualStaging, _ := filepath.Abs(pending.StagingDir)
	if actualStaging != expectedStaging {
		return fmt.Errorf("invalid restore staging directory")
	}
	stagedDB := filepath.Join(expectedStaging, "database", "novelhub.db")
	if info, err := os.Stat(stagedDB); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("staged database missing")
	}
	actualHash, err := restoreFileSHA256(stagedDB)
	if err != nil || actualHash != pending.DatabaseHash {
		return fmt.Errorf("staged database checksum mismatch")
	}

	safetyDir := filepath.Join(dataDir, "restore-safety", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(safetyDir, 0700); err != nil {
		return err
	}
	dbPath := config.GetConfigWithDefault("SQLITE_DB_PATH", filepath.Join(dataDir, "novelhub.db"))
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := copyIfExists(dbPath+suffix, filepath.Join(safetyDir, filepath.Base(dbPath)+suffix)); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return err
	}
	tempDB := dbPath + ".restore-tmp"
	_ = os.Remove(tempDB)
	if err := copyFile(stagedDB, tempDB); err != nil {
		return fmt.Errorf("stage restored database: %w", err)
	}
	booksSwapped := false
	currentBooks := filepath.Join(dataDir, "books")
	stagedBooks := filepath.Join(expectedStaging, "books")
	safetyBooks := filepath.Join(safetyDir, "books")
	if pending.IncludeBooks {
		if err := os.Rename(currentBooks, safetyBooks); err != nil && !os.IsNotExist(err) {
			_ = os.Remove(tempDB)
			return err
		}
		if err := os.Rename(stagedBooks, currentBooks); err != nil {
			_ = os.Rename(safetyBooks, currentBooks)
			_ = os.Remove(tempDB)
			return fmt.Errorf("apply staged books: %w", err)
		}
		booksSwapped = true
	}
	if err := os.Rename(tempDB, dbPath); err != nil {
		_ = os.Remove(tempDB)
		if booksSwapped {
			_ = os.Rename(currentBooks, stagedBooks)
			_ = os.Rename(safetyBooks, currentBooks)
		}
		return fmt.Errorf("apply staged database: %w", err)
	}
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	if err := os.Remove(marker); err != nil {
		return err
	}
	return os.RemoveAll(expectedStaging)
}

func restoreFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyIfExists(source, destination string) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return copyFile(source, destination)
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(destination)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(destination)
	}
	return closeErr
}
