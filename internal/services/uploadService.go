package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/localfs"
)

type UploadService interface {
	InitUploadSession(ctx context.Context, dto *request.InitUploadDto, claims *response.JWTClaims) (string, error)
	SaveChunk(ctx context.Context, uploadID, chunkIndex string, file *multipart.FileHeader, claims *response.JWTClaims) error
	CommitUpload(ctx context.Context, uploadID string, dto *request.CommitUploadDto, claims *response.JWTClaims) error
}

type uploadManifest struct {
	OwnerID     string    `json:"owner_id"`
	Target      string    `json:"target"`
	TargetID    string    `json:"target_id"`
	LibraryID   string    `json:"library_id"`
	Filename    string    `json:"filename"`
	TotalBytes  int64     `json:"total_bytes"`
	TotalChunks int       `json:"total_chunks"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type uploadService struct {
	libraryService LibraryService
	bookService    BookService
	libraryRepo    repositories.LibraryRepository
	permissions    PermissionCache
	mu             sync.Mutex
}

func NewUploadService(libraryService LibraryService, bookService BookService, libraryRepo repositories.LibraryRepository, permissions PermissionCache) UploadService {
	return &uploadService{libraryService: libraryService, bookService: bookService, libraryRepo: libraryRepo, permissions: permissions}
}

func (s *uploadService) activeSessions(ownerID string) int {
	root := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads")
	entries, _ := os.ReadDir(root)
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name(), "session.json"))
		if err != nil {
			continue
		}
		var manifest uploadManifest
		if jsonx.Unmarshal(data, &manifest) == nil && manifest.OwnerID == ownerID && time.Now().Before(manifest.ExpiresAt) {
			count++
		}
	}
	return count
}

func (s *uploadService) InitUploadSession(ctx context.Context, dto *request.InitUploadDto, claims *response.JWTClaims) (string, error) {
	if claims == nil || claims.UId == "" || dto == nil {
		return "", apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSessions(claims.UId) >= constants.MaxUploadSessions {
		return "", apperrors.New(apperrors.ErrConflict, "Too many active upload sessions")
	}
	if dto.TotalBytes < 1 || dto.TotalBytes > constants.MaxUploadBytes || dto.TotalChunks < 1 || dto.TotalChunks > constants.MaxUploadChunks {
		return "", apperrors.New(apperrors.ErrBadRequest, "Upload exceeds configured limits")
	}
	filename := filepath.Base(dto.Filename)
	if filename == "." || filename == "" {
		return "", apperrors.New(apperrors.ErrBadRequest, "Invalid filename")
	}

	manifest := uploadManifest{OwnerID: claims.UId, Target: dto.Target, Filename: filename, TotalBytes: dto.TotalBytes, TotalChunks: dto.TotalChunks, ExpiresAt: time.Now().Add(constants.UploadSessionTTL)}
	switch dto.Target {
	case "library":
		if dto.LibraryID == "" {
			return "", apperrors.New(apperrors.ErrBadRequest, "Missing library ID")
		}
		if _, err := s.libraryRepo.GetLibrary(ctx, dto.LibraryID); err != nil {
			return "", apperrors.New(apperrors.ErrNotFound, "Library not found")
		}
		manifest.TargetID, manifest.LibraryID = dto.LibraryID, dto.LibraryID
	case "book":
		book, err := s.bookService.GetBook(ctx, dto.BookID)
		if err != nil || book == nil {
			return "", apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		manifest.TargetID, manifest.LibraryID = dto.BookID, book.LibraryID
	default:
		return "", apperrors.New(apperrors.ErrBadRequest, "Invalid upload target")
	}
	if !s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermBookUpload, map[string]any{"library_id": manifest.LibraryID}) {
		return "", apperrors.New(apperrors.ErrForbidden, "Upload permission denied")
	}

	uploadID := uuid.Must(uuid.NewV7()).String()
	uploadDir, err := localfs.SafeJoin(filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads"), uploadID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(uploadDir, 0700); err != nil {
		return "", fmt.Errorf("initialize upload: %w", err)
	}
	data, err := jsonx.Marshal(manifest)
	if err != nil {
		_ = os.RemoveAll(uploadDir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(uploadDir, "session.json"), data, 0600); err != nil {
		_ = os.RemoveAll(uploadDir)
		return "", fmt.Errorf("write upload manifest: %w", err)
	}
	return uploadID, nil
}

func (s *uploadService) loadManifest(uploadID string, claims *response.JWTClaims) (string, *uploadManifest, error) {
	if _, err := uuid.Parse(uploadID); err != nil || claims == nil {
		return "", nil, apperrors.New(apperrors.ErrBadRequest, "Invalid upload session")
	}
	uploadDir, err := localfs.SafeJoin(filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads"), uploadID)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(filepath.Join(uploadDir, "session.json"))
	if err != nil {
		return "", nil, apperrors.New(apperrors.ErrNotFound, "Upload session not found")
	}
	var manifest uploadManifest
	if err := jsonx.Unmarshal(data, &manifest); err != nil {
		return "", nil, apperrors.New(apperrors.ErrBadRequest, "Invalid upload session")
	}
	if manifest.OwnerID != claims.UId {
		return "", nil, apperrors.New(apperrors.ErrForbidden, "Upload session belongs to another user")
	}
	if time.Now().After(manifest.ExpiresAt) {
		_ = os.RemoveAll(uploadDir)
		return "", nil, apperrors.New(apperrors.ErrBadRequest, "Upload session expired")
	}
	return uploadDir, &manifest, nil
}

func (s *uploadService) SaveChunk(ctx context.Context, uploadID, chunkIndexStr string, file *multipart.FileHeader, claims *response.JWTClaims) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	uploadDir, manifest, err := s.loadManifest(uploadID, claims)
	if err != nil {
		return err
	}
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 || chunkIndex >= manifest.TotalChunks {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid chunk index")
	}
	if file == nil || file.Size < 1 || file.Size > constants.MaxUploadChunkBytes {
		return apperrors.New(apperrors.ErrBadRequest, "Chunk exceeds configured limit")
	}
	var stored int64
	chunks, _ := filepath.Glob(filepath.Join(uploadDir, "chunk_*"))
	for _, chunk := range chunks {
		if info, err := os.Stat(chunk); err == nil {
			stored += info.Size()
		}
	}
	if stored+file.Size > manifest.TotalBytes || stored+file.Size > constants.MaxUploadBytes {
		return apperrors.New(apperrors.ErrBadRequest, "Upload exceeds declared size")
	}
	chunkPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", chunkIndex))
	dst, err := os.OpenFile(chunkPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return apperrors.New(apperrors.ErrConflict, "Chunk already uploaded")
	}
	if err != nil {
		return err
	}
	defer dst.Close()
	src, err := file.Open()
	if err != nil {
		_ = os.Remove(chunkPath)
		return err
	}
	defer src.Close()
	written, err := io.Copy(dst, io.LimitReader(src, constants.MaxUploadChunkBytes+1))
	if err != nil || written > constants.MaxUploadChunkBytes {
		_ = os.Remove(chunkPath)
		return apperrors.New(apperrors.ErrBadRequest, "Invalid or oversized chunk")
	}
	select {
	case <-ctx.Done():
		_ = os.Remove(chunkPath)
		return ctx.Err()
	default:
	}
	return nil
}

func (s *uploadService) CommitUpload(ctx context.Context, uploadID string, dto *request.CommitUploadDto, claims *response.JWTClaims) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	uploadDir, manifest, err := s.loadManifest(uploadID, claims)
	if err != nil {
		return err
	}
	if dto != nil && ((dto.Target != "" && dto.Target != manifest.Target) || (dto.Filename != "" && filepath.Base(dto.Filename) != manifest.Filename) || (dto.TotalChunks != 0 && dto.TotalChunks != manifest.TotalChunks) || (dto.LibraryID != "" && dto.LibraryID != manifest.TargetID) || (dto.BookID != "" && dto.BookID != manifest.TargetID)) {
		return apperrors.New(apperrors.ErrBadRequest, "Upload metadata does not match session")
	}
	if !s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermBookUpload, map[string]any{"library_id": manifest.LibraryID}) {
		return apperrors.New(apperrors.ErrForbidden, "Upload permission denied")
	}
	marker, err := os.OpenFile(filepath.Join(uploadDir, ".committing"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return apperrors.New(apperrors.ErrConflict, "Upload is already being committed")
	}
	if err != nil {
		return err
	}
	_ = marker.Close()
	defer os.Remove(filepath.Join(uploadDir, ".committing"))

	var total int64
	for i := 0; i < manifest.TotalChunks; i++ {
		info, err := os.Stat(filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", i)))
		if err != nil {
			return apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("Missing chunk %d", i))
		}
		total += info.Size()
		if total > manifest.TotalBytes || total > constants.MaxUploadBytes {
			return apperrors.New(apperrors.ErrBadRequest, "Upload exceeds declared size")
		}
	}
	if total != manifest.TotalBytes {
		return apperrors.New(apperrors.ErrBadRequest, "Upload size does not match declaration")
	}
	finalPath := filepath.Join(uploadDir, "merged_file")
	out, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	for i := 0; i < manifest.TotalChunks; i++ {
		chunk, openErr := os.Open(filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", i)))
		if openErr != nil {
			_ = out.Close()
			_ = os.Remove(finalPath)
			return openErr
		}
		_, copyErr := io.Copy(out, chunk)
		_ = chunk.Close()
		if copyErr != nil {
			_ = out.Close()
			_ = os.Remove(finalPath)
			return copyErr
		}
		select {
		case <-ctx.Done():
			_ = out.Close()
			_ = os.Remove(finalPath)
			return ctx.Err()
		default:
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(finalPath)
		return err
	}

	if manifest.Target == "library" {
		err = s.libraryService.ProcessSingleLocalFile(ctx, manifest.TargetID, manifest.Filename, finalPath)
	} else {
		err = s.bookService.ProcessSingleLocalFile(ctx, manifest.TargetID, manifest.Filename, finalPath)
	}
	if err != nil {
		_ = os.Remove(finalPath)
		return err
	}
	return os.RemoveAll(uploadDir)
}
