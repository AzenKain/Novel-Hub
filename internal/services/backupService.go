package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"modernc.org/sqlite"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/localfs"
	"novelhub/pkg/systemgate"
)

const backupFormatVersion = 1

type backupManifest struct {
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	IncludeBooks bool      `json:"include_books"`
	DatabaseHash string    `json:"database_hash"`
}

type pendingRestore struct {
	StagingDir   string `json:"staging_dir"`
	IncludeBooks bool   `json:"include_books"`
	DatabaseHash string `json:"database_hash"`
}

type BackupService interface {
	List(ctx context.Context) ([]*response.BackupResponse, error)
	Create(ctx context.Context, includeBooks bool) (*response.BackupResponse, error)
	Delete(ctx context.Context, name string) error
	Path(name string) (string, error)
	StageRestore(ctx context.Context, name string) (*response.RestoreResponse, error)
}

type backupService struct {
	db          *sql.DB
	dataDir     string
	backupDir   string
	autoRestart bool
	requestStop func()
	gate        *systemgate.Gate
	mu          sync.Mutex
}

func NewBackupService(db *sql.DB, dataDir string, autoRestart bool, requestStop func(), gate *systemgate.Gate) BackupService {
	return &backupService{db: db, dataDir: dataDir, backupDir: filepath.Join(dataDir, "backups"), autoRestart: autoRestart, requestStop: requestStop, gate: gate}
}

func (s *backupService) Path(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || !strings.HasSuffix(name, ".tar.gz") || !strings.HasPrefix(name, "novelhub-") {
		return "", apperrors.New(apperrors.ErrBadRequest, "invalid backup name")
	}
	path, err := localfs.SafeJoin(s.backupDir, name)
	if err != nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "invalid backup path")
	}
	info, err := os.Lstat(path)
	if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", apperrors.New(apperrors.ErrNotFound, "backup not found")
	}
	return path, nil
}

func (s *backupService) List(ctx context.Context) ([]*response.BackupResponse, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*response.BackupResponse{}, nil
		}
		return nil, err
	}
	out := make([]*response.BackupResponse, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		manifest, _ := readBackupManifest(filepath.Join(s.backupDir, entry.Name()))
		out = append(out, &response.BackupResponse{Name: entry.Name(), SizeBytes: info.Size(), CreatedAt: info.ModTime(), IncludeBooks: manifest != nil && manifest.IncludeBooks})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *backupService) Create(ctx context.Context, includeBooks bool) (*response.BackupResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.backupDir, 0750); err != nil {
		return nil, err
	}
	id := uuid.Must(uuid.NewV7()).String()
	name := fmt.Sprintf("novelhub-%s-%s.tar.gz", time.Now().Format("20060102-150405"), id)
	tempDir, err := os.MkdirTemp(s.backupDir, ".backup-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	dbSnapshot := filepath.Join(tempDir, "novelhub.db")
	if err := sqliteSnapshot(ctx, s.db, dbSnapshot); err != nil {
		return nil, err
	}
	dbHash, err := fileSHA256(dbSnapshot)
	if err != nil {
		return nil, err
	}
	manifest := backupManifest{Version: backupFormatVersion, CreatedAt: time.Now().UTC(), IncludeBooks: includeBooks, DatabaseHash: dbHash}
	tmpArchive := filepath.Join(s.backupDir, "."+name+".tmp")
	if err := writeBackupArchive(ctx, tmpArchive, dbSnapshot, filepath.Join(s.dataDir, "books"), manifest); err != nil {
		_ = os.Remove(tmpArchive)
		return nil, err
	}
	finalPath := filepath.Join(s.backupDir, name)
	if err := os.Rename(tmpArchive, finalPath); err != nil {
		return nil, err
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		return nil, err
	}
	return &response.BackupResponse{Name: name, SizeBytes: info.Size(), CreatedAt: info.ModTime(), IncludeBooks: includeBooks}, nil
}

func sqliteSnapshot(ctx context.Context, db *sql.DB, destination string) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		backupConn, ok := driverConn.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		})
		if !ok {
			return fmt.Errorf("sqlite backup API unavailable")
		}
		backup, err := backupConn.NewBackup(destination)
		if err != nil {
			return err
		}
		if _, err := backup.Step(-1); err != nil {
			_ = backup.Finish()
			return err
		}
		return backup.Finish()
	})
}

func fileSHA256(path string) (string, error) {
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

func writeBackupArchive(ctx context.Context, destination, databasePath, booksDir string, manifest backupManifest) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	closeAll := func() error {
		if err := tarWriter.Close(); err != nil {
			return err
		}
		if err := gzipWriter.Close(); err != nil {
			return err
		}
		return file.Close()
	}
	manifestData, err := jsonx.Marshal(manifest)
	if err != nil {
		_ = file.Close()
		return err
	}
	if err := writeTarBytes(tarWriter, "manifest.json", manifestData, 0600); err != nil {
		_ = file.Close()
		return err
	}
	if err := writeTarFile(tarWriter, databasePath, "database/novelhub.db"); err != nil {
		_ = file.Close()
		return err
	}
	if manifest.IncludeBooks {
		err = filepath.WalkDir(booksDir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if path == booksDir || entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(booksDir, path)
			if err != nil || !filepath.IsLocal(rel) {
				return fmt.Errorf("invalid book path")
			}
			return writeTarFile(tarWriter, path, filepath.ToSlash(filepath.Join("books", rel)))
		})
		if err != nil && !os.IsNotExist(err) {
			_ = file.Close()
			return err
		}
	}
	return closeAll()
}

func writeTarBytes(writer *tar.Writer, name string, data []byte, mode int64) error {
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), ModTime: time.Now()}); err != nil {
		return err
	}
	_, err := writer.Write(data)
	return err
}

func writeTarFile(writer *tar.Writer, source, name string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: info.Size(), ModTime: info.ModTime()}); err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}

func readBackupManifest(path string) (*backupManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Name == "manifest.json" {
			data, err := io.ReadAll(io.LimitReader(tarReader, 64<<10))
			if err != nil {
				return nil, err
			}
			var manifest backupManifest
			if err := jsonx.Unmarshal(data, &manifest); err != nil {
				return nil, err
			}
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("backup manifest missing")
}

func (s *backupService) Delete(ctx context.Context, name string) error {
	path, err := s.Path(name)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Remove(path)
}

func (s *backupService) StageRestore(ctx context.Context, name string) (*response.RestoreResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.Path(name)
	if err != nil {
		return nil, err
	}
	stagingRoot := filepath.Join(s.dataDir, ".restore-staging")
	if err := os.RemoveAll(stagingRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(stagingRoot, 0700); err != nil {
		return nil, err
	}
	manifest, err := extractBackup(ctx, path, stagingRoot)
	if err != nil {
		_ = os.RemoveAll(stagingRoot)
		return nil, err
	}
	dbHash, err := fileSHA256(filepath.Join(stagingRoot, "database", "novelhub.db"))
	if err != nil || dbHash != manifest.DatabaseHash {
		_ = os.RemoveAll(stagingRoot)
		return nil, apperrors.New(apperrors.ErrBadRequest, "backup database checksum mismatch")
	}
	if err := validateStagedDatabase(ctx, filepath.Join(stagingRoot, "database", "novelhub.db")); err != nil {
		_ = os.RemoveAll(stagingRoot)
		return nil, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	pending := pendingRestore{StagingDir: stagingRoot, IncludeBooks: manifest.IncludeBooks, DatabaseHash: manifest.DatabaseHash}
	data, err := jsonx.Marshal(pending)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, ".pending-restore.json"), data, 0600); err != nil {
		return nil, err
	}
	if s.gate != nil {
		s.gate.Enable()
	}
	if s.autoRestart && s.requestStop != nil {
		time.AfterFunc(500*time.Millisecond, s.requestStop)
	}
	return &response.RestoreResponse{RestartRequired: true, AutoRestart: s.autoRestart}, nil
}

func validateStagedDatabase(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("invalid backup database: %w", err)
	}
	if _, err := sqlc.New(db).DatabaseHealthCheck(ctx); err != nil {
		return fmt.Errorf("backup is not a NovelHub database: %w", err)
	}
	return nil
}

func extractBackup(ctx context.Context, source, destination string) (*backupManifest, error) {
	file, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var manifest *backupManifest
	files, total := 0, int64(0)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg || !filepath.IsLocal(filepath.FromSlash(header.Name)) {
			return nil, fmt.Errorf("unsafe backup entry")
		}
		files++
		total += header.Size
		if files > 1_000_000 || total > 1<<40 {
			return nil, fmt.Errorf("backup exceeds restore limits")
		}
		target, err := localfs.SafeJoin(destination, filepath.FromSlash(header.Name))
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.CopyN(out, tarReader, header.Size)
		closeErr := out.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if header.Name == "manifest.json" {
			data, err := os.ReadFile(target)
			if err != nil {
				return nil, err
			}
			manifest = &backupManifest{}
			if err := jsonx.Unmarshal(data, manifest); err != nil || manifest.Version != backupFormatVersion {
				return nil, fmt.Errorf("unsupported backup manifest")
			}
		}
	}
	if manifest == nil {
		return nil, fmt.Errorf("backup manifest missing")
	}
	if manifest.IncludeBooks {
		if err := os.MkdirAll(filepath.Join(destination, "books"), 0700); err != nil {
			return nil, err
		}
	}
	return manifest, nil
}
