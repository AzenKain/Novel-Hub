package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser"
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

type uploadLimitsProvider interface {
	Limits() models.RuntimeLimits
}

type uploadManifest struct {
	OwnerID           string    `json:"owner_id"`
	Target            string    `json:"target"`
	TargetID          string    `json:"target_id"`
	LibraryID         string    `json:"library_id"`
	Filename          string    `json:"filename"`
	TotalBytes        int64     `json:"total_bytes"`
	TotalChunks       int       `json:"total_chunks"`
	ExpiresAt         time.Time `json:"expires_at"`
	UploadChunkBytes  int64     `json:"upload_chunk_bytes"`
	UploadChunks      int       `json:"upload_chunks"`
	UploadSessions    int       `json:"upload_sessions"`
	UploadBytes       int64     `json:"upload_bytes"`
	SessionTTLSeconds int64     `json:"upload_session_ttl_seconds"`
}

type uploadSession struct {
	mu           sync.Mutex
	manifest     uploadManifest
	storedBytes  int64
	storedChunks int
	removed      bool
}

type uploadService struct {
	libraryService LibraryService
	bookService    BookService
	libraryRepo    repositories.LibraryRepository
	permissions    PermissionCache
	limits         uploadLimitsProvider
	root           string
	mu             sync.Mutex
	sessions       map[string]*uploadSession
	ownerCounts    map[string]int
}

func NewUploadService(libraryService LibraryService, bookService BookService, libraryRepo repositories.LibraryRepository, permissions PermissionCache, limits ...uploadLimitsProvider) UploadService {
	root, err := localfs.SafeJoin(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads")
	if err != nil {
		root = filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "uploads")
	}
	s := &uploadService{
		libraryService: libraryService,
		bookService:    bookService,
		libraryRepo:    libraryRepo,
		permissions:    permissions,
		root:           root,
		sessions:       make(map[string]*uploadSession),
		ownerCounts:    make(map[string]int),
	}
	if len(limits) > 0 {
		s.limits = limits[0]
	}
	s.recoverSessions()
	return s
}

func (s *uploadService) effectiveLimits() models.RuntimeLimits {
	if s.limits != nil {
		return s.limits.Limits()
	}
	return models.RuntimeLimits{
		UploadChunkBytes:        constants.MaxUploadChunkBytes,
		UploadChunks:            constants.MaxUploadChunks,
		UploadSessions:          constants.MaxUploadSessions,
		UploadBytes:             constants.MaxUploadBytes,
		UploadSessionTTLSeconds: int64(constants.UploadSessionTTL / time.Second),
	}
}

func (s *uploadService) recoverSessions() {
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir, err := localfs.SafeJoin(s.root, entry.Name())
		if err != nil {
			continue
		}
		manifest, storedBytes, storedChunks, err := s.readSession(dir, entry.Name())
		if err != nil || !now.Before(manifest.ExpiresAt) {
			_ = os.RemoveAll(dir)
			continue
		}
		s.sessions[entry.Name()] = &uploadSession{manifest: manifest, storedBytes: storedBytes, storedChunks: storedChunks}
		s.ownerCounts[manifest.OwnerID]++
	}
}

func (s *uploadService) readSession(dir, uploadID string) (uploadManifest, int64, int, error) {
	var manifest uploadManifest
	if _, err := uuid.Parse(uploadID); err != nil {
		return manifest, 0, 0, err
	}
	manifestPath, err := localfs.SafeJoin(dir, "session.json")
	if err != nil {
		return manifest, 0, 0, err
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil || jsonx.Unmarshal(data, &manifest) != nil || manifest.OwnerID == "" || manifest.TotalBytes < 1 || manifest.TotalChunks < 1 {
		return manifest, 0, 0, fmt.Errorf("invalid upload manifest")
	}
	if manifest.UploadChunkBytes < 1 || manifest.UploadChunks < 1 || manifest.UploadSessions < 1 || manifest.UploadBytes < 1 || manifest.SessionTTLSeconds < 1 || manifest.TotalChunks > manifest.UploadChunks || manifest.TotalBytes > manifest.UploadBytes {
		return manifest, 0, 0, fmt.Errorf("invalid upload limits")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return manifest, 0, 0, err
	}
	var storedBytes int64
	storedChunks := 0
	seen := make(map[int]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "chunk_") {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), "chunk_"))
		if err != nil || index < 0 || index >= manifest.TotalChunks {
			return manifest, 0, 0, fmt.Errorf("invalid upload chunk")
		}
		if _, exists := seen[index]; exists {
			return manifest, 0, 0, fmt.Errorf("duplicate upload chunk")
		}
		seen[index] = struct{}{}
		info, err := entry.Info()
		if err != nil || info.Size() < 1 || info.Size() > manifest.UploadChunkBytes {
			return manifest, 0, 0, fmt.Errorf("invalid upload chunk")
		}
		storedBytes += info.Size()
		storedChunks++
	}
	if storedBytes > manifest.TotalBytes || storedChunks > manifest.TotalChunks {
		return manifest, 0, 0, fmt.Errorf("invalid upload contents")
	}
	return manifest, storedBytes, storedChunks, nil
}

func (s *uploadService) cleanupExpired(ownerID string) {
	now := time.Now()
	s.mu.Lock()
	candidates := make(map[string]*uploadSession)
	for id, session := range s.sessions {
		if session.manifest.OwnerID == ownerID && !now.Before(session.manifest.ExpiresAt) {
			candidates[id] = session
		}
	}
	s.mu.Unlock()
	for id, session := range candidates {
		session.mu.Lock()
		if !session.removed && !now.Before(session.manifest.ExpiresAt) {
			s.removeSession(id, session)
		}
		session.mu.Unlock()
	}
}

func (s *uploadService) InitUploadSession(ctx context.Context, dto *request.InitUploadDto, claims *response.JWTClaims) (string, error) {
	if claims == nil || claims.UId == "" || dto == nil {
		return "", apperrors.New(apperrors.ErrUnauthorized, "Authentication required")
	}
	limits := s.effectiveLimits()
	if limits.UploadChunkBytes < 1 || limits.UploadChunks < 1 || limits.UploadSessions < 1 || limits.UploadBytes < 1 || limits.UploadSessionTTLSeconds < 1 {
		return "", apperrors.New(apperrors.ErrInternalError, "Invalid upload limits")
	}
	if dto.TotalBytes < 1 || dto.TotalBytes > limits.UploadBytes || dto.TotalChunks < 1 || dto.TotalChunks > limits.UploadChunks {
		return "", apperrors.New(apperrors.ErrBadRequest, "Upload exceeds configured limits")
	}
	filename := filepath.Base(dto.Filename)
	if filename == "." || filename == "" {
		return "", apperrors.New(apperrors.ErrBadRequest, "Invalid filename")
	}
	if !bookparser.IsAllowedBookFormat(filename) {
		return "", apperrors.New(apperrors.ErrBadRequest, "Unsupported file format: only valid book and audiobook files are allowed")
	}

	manifest := uploadManifest{
		OwnerID: claims.UId, Target: dto.Target, Filename: filename, TotalBytes: dto.TotalBytes, TotalChunks: dto.TotalChunks,
		ExpiresAt: time.Now().Add(time.Duration(limits.UploadSessionTTLSeconds) * time.Second), UploadChunkBytes: limits.UploadChunkBytes,
		UploadChunks: limits.UploadChunks, UploadSessions: limits.UploadSessions, UploadBytes: limits.UploadBytes,
		SessionTTLSeconds: limits.UploadSessionTTLSeconds,
	}
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

	s.cleanupExpired(claims.UId)
	s.mu.Lock()
	if s.ownerCounts[claims.UId] >= limits.UploadSessions {
		s.mu.Unlock()
		return "", apperrors.New(apperrors.ErrConflict, "Too many active upload sessions")
	}
	s.ownerCounts[claims.UId]++
	s.mu.Unlock()
	registered := false
	defer func() {
		if !registered {
			s.mu.Lock()
			s.ownerCounts[claims.UId]--
			s.mu.Unlock()
		}
	}()

	uploadID := uuid.Must(uuid.NewV7()).String()
	uploadDir, err := localfs.SafeJoin(s.root, uploadID)
	if err != nil {
		return "", err
	}
	if err := os.Mkdir(uploadDir, 0700); err != nil {
		return "", fmt.Errorf("initialize upload: %w", err)
	}
	data, err := jsonx.Marshal(manifest)
	if err != nil {
		_ = os.RemoveAll(uploadDir)
		return "", err
	}
	manifestPath, err := localfs.SafeJoin(uploadDir, "session.json")
	if err != nil {
		_ = os.RemoveAll(uploadDir)
		return "", err
	}
	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		_ = os.RemoveAll(uploadDir)
		return "", fmt.Errorf("write upload manifest: %w", err)
	}
	s.mu.Lock()
	s.sessions[uploadID] = &uploadSession{manifest: manifest}
	s.mu.Unlock()
	registered = true
	return uploadID, nil
}

func (s *uploadService) lockSession(uploadID string, claims *response.JWTClaims) (string, *uploadSession, error) {
	if _, err := uuid.Parse(uploadID); err != nil || claims == nil {
		return "", nil, apperrors.New(apperrors.ErrBadRequest, "Invalid upload session")
	}
	s.mu.Lock()
	session := s.sessions[uploadID]
	s.mu.Unlock()
	if session == nil {
		return "", nil, apperrors.New(apperrors.ErrNotFound, "Upload session not found")
	}
	session.mu.Lock()
	if session.removed {
		session.mu.Unlock()
		return "", nil, apperrors.New(apperrors.ErrNotFound, "Upload session not found")
	}
	if session.manifest.OwnerID != claims.UId {
		session.mu.Unlock()
		return "", nil, apperrors.New(apperrors.ErrForbidden, "Upload session belongs to another user")
	}
	if time.Now().After(session.manifest.ExpiresAt) {
		s.removeSession(uploadID, session)
		session.mu.Unlock()
		return "", nil, apperrors.New(apperrors.ErrBadRequest, "Upload session expired")
	}
	uploadDir, err := localfs.SafeJoin(s.root, uploadID)
	if err != nil {
		session.mu.Unlock()
		return "", nil, err
	}
	return uploadDir, session, nil
}

func (s *uploadService) removeSession(uploadID string, session *uploadSession) {
	s.mu.Lock()
	if s.sessions[uploadID] == session {
		session.removed = true
		delete(s.sessions, uploadID)
		s.ownerCounts[session.manifest.OwnerID]--
	}
	s.mu.Unlock()
	dir, err := localfs.SafeJoin(s.root, uploadID)
	if err == nil {
		_ = os.RemoveAll(dir)
	}
}

func (s *uploadService) SaveChunk(ctx context.Context, uploadID, chunkIndexStr string, file *multipart.FileHeader, claims *response.JWTClaims) error {
	uploadDir, session, err := s.lockSession(uploadID, claims)
	if err != nil {
		return err
	}
	defer session.mu.Unlock()
	manifest := &session.manifest
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 || chunkIndex >= manifest.TotalChunks {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid chunk index")
	}
	if file == nil || file.Size < 1 || file.Size > manifest.UploadChunkBytes || session.storedBytes+file.Size > manifest.TotalBytes || session.storedChunks >= manifest.TotalChunks {
		return apperrors.New(apperrors.ErrBadRequest, "Chunk exceeds configured limit")
	}
	chunkPath, err := localfs.SafeJoin(uploadDir, fmt.Sprintf("chunk_%d", chunkIndex))
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(chunkPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return apperrors.New(apperrors.ErrConflict, "Chunk already uploaded")
	}
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		_ = dst.Close()
		_ = os.Remove(chunkPath)
		return err
	}
	written, copyErr := io.Copy(dst, io.LimitReader(src, manifest.UploadChunkBytes+1))
	closeErr := dst.Close()
	_ = src.Close()
	if copyErr != nil || closeErr != nil || written < 1 || written > manifest.UploadChunkBytes || written != file.Size || session.storedBytes+written > manifest.TotalBytes {
		_ = os.Remove(chunkPath)
		return apperrors.New(apperrors.ErrBadRequest, "Invalid or oversized chunk")
	}
	select {
	case <-ctx.Done():
		_ = os.Remove(chunkPath)
		return ctx.Err()
	default:
	}
	session.storedBytes += written
	session.storedChunks++
	return nil
}

func (s *uploadService) CommitUpload(ctx context.Context, uploadID string, dto *request.CommitUploadDto, claims *response.JWTClaims) error {
	uploadDir, session, err := s.lockSession(uploadID, claims)
	if err != nil {
		return err
	}
	defer session.mu.Unlock()
	manifest := &session.manifest
	if dto != nil && ((dto.Target != "" && dto.Target != manifest.Target) || (dto.Filename != "" && filepath.Base(dto.Filename) != manifest.Filename) || (dto.TotalChunks != 0 && dto.TotalChunks != manifest.TotalChunks) || (dto.LibraryID != "" && dto.LibraryID != manifest.TargetID) || (dto.BookID != "" && dto.BookID != manifest.TargetID)) {
		return apperrors.New(apperrors.ErrBadRequest, "Upload metadata does not match session")
	}
	if !bookparser.IsAllowedBookFormat(manifest.Filename) {
		return apperrors.New(apperrors.ErrBadRequest, "Unsupported file format")
	}
	if !s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermBookUpload, map[string]any{"library_id": manifest.LibraryID}) {
		return apperrors.New(apperrors.ErrForbidden, "Upload permission denied")
	}
	markerPath, err := localfs.SafeJoin(uploadDir, ".committing")
	if err != nil {
		return err
	}
	marker, err := os.OpenFile(markerPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return apperrors.New(apperrors.ErrConflict, "Upload is already being committed")
	}
	if err != nil {
		return err
	}
	_ = marker.Close()
	defer os.Remove(markerPath)

	var total int64
	for i := 0; i < manifest.TotalChunks; i++ {
		chunkPath, _ := localfs.SafeJoin(uploadDir, fmt.Sprintf("chunk_%d", i))
		info, err := os.Stat(chunkPath)
		if err != nil {
			return apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("Missing chunk %d", i))
		}
		total += info.Size()
		if info.Size() < 1 || info.Size() > manifest.UploadChunkBytes || total > manifest.TotalBytes || total > manifest.UploadBytes {
			return apperrors.New(apperrors.ErrBadRequest, "Upload exceeds declared size")
		}
	}
	if total != manifest.TotalBytes || session.storedChunks != manifest.TotalChunks || session.storedBytes != total {
		return apperrors.New(apperrors.ErrBadRequest, "Upload size does not match declaration")
	}
	finalPath, _ := localfs.SafeJoin(uploadDir, "merged_file")
	out, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	for i := 0; i < manifest.TotalChunks; i++ {
		chunkPath, _ := localfs.SafeJoin(uploadDir, fmt.Sprintf("chunk_%d", i))
		chunk, openErr := os.Open(chunkPath)
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
	s.removeSession(uploadID, session)
	return nil
}
