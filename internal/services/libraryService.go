package services

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/worker"
)

type LibraryService interface {
	CreateLibrary(ctx context.Context, dto *request.CreateLibraryDto) (*response.LibraryResponse, error)
	GetLibrary(ctx context.Context, id string, claims *response.JWTClaims) (*response.LibraryResponse, error)
	ListLibraries(ctx context.Context, claims *response.JWTClaims) ([]*response.LibraryResponse, error)
	UpdateLibrary(ctx context.Context, id string, dto *request.UpdateLibraryDto) (*response.LibraryResponse, error)
	DeleteLibrary(ctx context.Context, id string) error
	UploadFiles(ctx context.Context, libraryID string, files []*multipart.FileHeader) (*response.LibraryUploadResultResponse, error)
	ProcessSingleLocalFile(ctx context.Context, libraryID string, filename string, localFilePath string) error
	StreamLibraryZip(ctx context.Context, libraryID string, w io.Writer) error
	ScanInbox(ctx context.Context) (int, error)
}

type libraryService struct {
	libraryRepo repositories.LibraryRepository
	bookRepo    repositories.BookDBRepository
	fileRepo    repositories.BookFileRepository
	parsers     *bookparser.Registry
	permissions PermissionCache
	jobQueue    *worker.Queue
}

func NewLibraryService(repo repositories.LibraryRepository, bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, parsers *bookparser.Registry, permissions PermissionCache, jobQueue *worker.Queue) LibraryService {
	return &libraryService{
		libraryRepo: repo,
		bookRepo:    bookRepo,
		fileRepo:    fileRepo,
		parsers:     parsers,
		permissions: permissions,
		jobQueue:    jobQueue,
	}
}

func (s *libraryService) CreateLibrary(ctx context.Context, dto *request.CreateLibraryDto) (*response.LibraryResponse, error) {
	lib := &models.LibraryEntity{
		ID:   uuid.Must(uuid.NewV7()).String(),
		Name: dto.Name,
	}
	if err := s.libraryRepo.CreateLibrary(ctx, lib); err != nil {
		return nil, err
	}
	return lib.ToResponse(), nil
}

func (s *libraryService) GetLibrary(ctx context.Context, id string, claims *response.JWTClaims) (*response.LibraryResponse, error) {
	lib, err := s.libraryRepo.GetLibrary(ctx, id)
	if err != nil {
		return nil, err
	}
	resolved := resolveClaims(claims)
	if !s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermLibraryRead, map[string]any{"library_id": lib.ID}) {
		return nil, apperrors.New(apperrors.ErrNotFound, "Library not found")
	}
	return lib.ToResponse(), nil
}

func (s *libraryService) ListLibraries(ctx context.Context, claims *response.JWTClaims) ([]*response.LibraryResponse, error) {
	libs, err := s.libraryRepo.ListLibraries(ctx)
	if err != nil {
		return nil, err
	}
	resolved := resolveClaims(claims)
	visible := make([]*models.LibraryEntity, 0, len(libs))
	for _, lib := range libs {
		if s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermLibraryRead, map[string]any{"library_id": lib.ID}) {
			visible = append(visible, lib)
		}
	}
	return models.LibraryEntitiesToResponse(visible), nil
}

func (s *libraryService) UpdateLibrary(ctx context.Context, id string, dto *request.UpdateLibraryDto) (*response.LibraryResponse, error) {
	lib, err := s.libraryRepo.GetLibrary(ctx, id)
	if err != nil {
		return nil, err
	}

	lib.Name = dto.Name

	if err := s.libraryRepo.UpdateLibrary(ctx, lib); err != nil {
		return nil, err
	}
	return lib.ToResponse(), nil
}

func (s *libraryService) DeleteLibrary(ctx context.Context, id string) error {
	return s.libraryRepo.DeleteLibrary(ctx, id)
}

func (s *libraryService) UploadFiles(ctx context.Context, libraryID string, files []*multipart.FileHeader) (*response.LibraryUploadResultResponse, error) {
	if _, err := s.libraryRepo.GetLibrary(ctx, libraryID); err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "Library not found")
		}
		return nil, err
	}
	if len(files) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "no files provided")
	}

	successCount := 0
	pendingJobs := make([]worker.Job, 0, len(files))
	for _, file := range files {
		bookID := uuid.Must(uuid.NewV7()).String()
		ext := filepath.Ext(file.Filename)

		src, err := file.Open()
		if err != nil {
			log.Error().Err(err).Str("file", file.Filename).Msg("failed to open uploaded file")
			continue
		}
		saved, saveErr := s.fileRepo.SaveBook(ctx, bookID, file.Filename, src)
		closeErr := src.Close()
		if saveErr != nil {
			_ = s.fileRepo.RemoveBookDir(ctx, bookID)
			log.Error().Err(saveErr).Str("file", file.Filename).Msg("failed to save uploaded file")
			continue
		}
		if closeErr != nil {
			_ = s.fileRepo.RemoveBookDir(ctx, bookID)
			log.Error().Err(closeErr).Str("file", file.Filename).Msg("failed to close uploaded file")
			continue
		}

		metaData := map[string]string{
			"original_filename": file.Filename,
			"upload_time":       time.Now().Format(time.RFC3339),
			"uuid":              bookID,
		}
		if err := s.fileRepo.WriteBookMeta(ctx, bookID, metaData); err != nil {
			_ = s.fileRepo.RemoveBookDir(ctx, bookID)
			log.Error().Err(err).Str("file", file.Filename).Msg("failed to write book metadata")
			continue
		}

		book := &models.BookEntity{
			ID:        bookID,
			LibraryID: libraryID,
			Title:     strings.TrimSuffix(file.Filename, ext),
			Status:    "processing",
		}

		fileID := uuid.Must(uuid.NewV7()).String()
		state := "managed"
		err = s.bookRepo.CreateBookWithFile(ctx, book, &sqlc.CreateBookFileParams{
			ID:        fileID,
			BookID:    bookID,
			Path:      saved.Path,
			Format:    saved.Format,
			SizeBytes: saved.SizeBytes,
			ModTime:   saved.ModTime,
			State:     convert.StrPtrToNullString(&state),
		})
		if err != nil {
			_ = s.fileRepo.RemoveBookDir(ctx, bookID)
			log.Error().Err(err).Str("file", file.Filename).Msg("failed to create book and file record")
			continue
		}

		pendingJobs = append(pendingJobs, worker.Job{
			ID:      uuid.Must(uuid.NewV7()).String(),
			Type:    "extract_metadata",
			Payload: bookID,
		}, worker.Job{
			ID:      uuid.Must(uuid.NewV7()).String(),
			Type:    "hash_file",
			Payload: fileID,
		})

		successCount++
	}

	if s.jobQueue != nil {
		for _, job := range pendingJobs {
			if err := s.jobQueue.Enqueue(ctx, job); err != nil {
				return nil, fmt.Errorf("queue post-processing: %w", err)
			}
		}
	}

	res := &models.LibraryUploadResult{Uploaded: successCount, Total: len(files)}
	return res.ToResponse(), nil
}

func (s *libraryService) ProcessSingleLocalFile(ctx context.Context, libraryID string, filename string, localFilePath string) error {
	if _, err := s.libraryRepo.GetLibrary(ctx, libraryID); err != nil {
		return err
	}

	bookID := uuid.Must(uuid.NewV7()).String()
	ext := filepath.Ext(filename)

	src, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	saved, saveErr := s.fileRepo.SaveBook(ctx, bookID, filename, src)
	closeErr := src.Close()
	if saveErr != nil {
		_ = s.fileRepo.RemoveBookDir(ctx, bookID)
		return fmt.Errorf("failed to save book: %w", saveErr)
	}
	if closeErr != nil {
		_ = s.fileRepo.RemoveBookDir(ctx, bookID)
		return fmt.Errorf("failed to close local file: %w", closeErr)
	}

	metaData := map[string]string{
		"original_filename": filename,
		"upload_time":       time.Now().Format(time.RFC3339),
		"uuid":              bookID,
	}
	if err := s.fileRepo.WriteBookMeta(ctx, bookID, metaData); err != nil {
		_ = s.fileRepo.RemoveBookDir(ctx, bookID)
		return fmt.Errorf("failed to write book metadata: %w", err)
	}

	book := &models.BookEntity{
		ID:        bookID,
		LibraryID: libraryID,
		Title:     strings.TrimSuffix(filename, ext),
		Status:    "processing",
	}

	fileID := uuid.Must(uuid.NewV7()).String()
	state := "managed"
	err = s.bookRepo.CreateBookWithFile(ctx, book, &sqlc.CreateBookFileParams{
		ID:        fileID,
		BookID:    bookID,
		Path:      saved.Path,
		Format:    saved.Format,
		SizeBytes: saved.SizeBytes,
		ModTime:   saved.ModTime,
		State:     convert.StrPtrToNullString(&state),
	})
	if err != nil {
		_ = s.fileRepo.RemoveBookDir(ctx, bookID)
		return fmt.Errorf("failed to create book and file record: %w", err)
	}

	pendingJobs := []worker.Job{
		{
			ID:      uuid.Must(uuid.NewV7()).String(),
			Type:    "extract_metadata",
			Payload: bookID,
		},
		{
			ID:      uuid.Must(uuid.NewV7()).String(),
			Type:    "hash_file",
			Payload: fileID,
		},
	}
	if s.jobQueue == nil {
		return fmt.Errorf("job queue is unavailable")
	}
	for _, job := range pendingJobs {
		if err := s.jobQueue.Enqueue(ctx, job); err != nil {
			return fmt.Errorf("queue post-processing: %w", err)
		}
	}
	return nil
}

func (s *libraryService) StreamLibraryZip(ctx context.Context, libraryID string, w io.Writer) (err error) {
	if _, err := s.libraryRepo.GetLibrary(ctx, libraryID); err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	defer func() {
		if closeErr := zw.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close ZIP archive: %w", closeErr)
		}
	}()

	var cursor *time.Time
	cursorID := ""
	for {
		books, searchErr := s.bookRepo.SearchBooks(ctx, &libraryID, nil, "", "", "", "", "", cursor, cursorID, 100)
		if searchErr != nil {
			return searchErr
		}
		if len(books) == 0 {
			return nil
		}

		for _, book := range books {
			files, filesErr := s.bookRepo.GetFilesByBookId(ctx, book.ID)
			if filesErr != nil {
				return fmt.Errorf("get files for book %s: %w", book.ID, filesErr)
			}
			for _, f := range files {
				if f.State != nil && *f.State == "deleted" {
					continue
				}

				src, openErr := os.Open(f.Path)
				if openErr != nil {
					return fmt.Errorf("open book file %s: %w", f.ID, openErr)
				}
				safeTitle := strings.ReplaceAll(book.Title, "/", "-")
				filename := fmt.Sprintf("%s.%s", safeTitle, f.Format)
				fw, createErr := zw.Create(filename)
				if createErr != nil {
					_ = src.Close()
					return fmt.Errorf("create ZIP entry %q: %w", filename, createErr)
				}
				if _, copyErr := io.Copy(fw, src); copyErr != nil {
					_ = src.Close()
					return fmt.Errorf("copy book file %s to ZIP: %w", f.ID, copyErr)
				}
				if closeErr := src.Close(); closeErr != nil {
					return fmt.Errorf("close book file %s: %w", f.ID, closeErr)
				}
			}
		}

		if len(books) < 100 {
			return nil
		}
		last := books[len(books)-1]
		cursor = &last.CreatedAt
		cursorID = last.ID
	}
}
