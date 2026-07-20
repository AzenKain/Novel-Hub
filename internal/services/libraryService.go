package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/convert"
	"novelhub/pkg/worker"
)

type LibraryService interface {
	CreateLibrary(ctx context.Context, dto *request.CreateLibraryDto) (*models.LibraryEntity, error)
	GetLibrary(ctx context.Context, id string) (*models.LibraryEntity, error)
	ListLibraries(ctx context.Context) ([]*models.LibraryEntity, error)
	UpdateLibrary(ctx context.Context, id string, dto *request.UpdateLibraryDto) (*models.LibraryEntity, error)
	DeleteLibrary(ctx context.Context, id string) error
	UploadFiles(ctx context.Context, libraryID string, files []*multipart.FileHeader) (*models.LibraryUploadResult, error)
}

type libraryService struct {
	libraryRepo repositories.LibraryRepository
	bookRepo    repositories.BookDBRepository
	fileRepo    repositories.BookFileRepository
	jobQueue    *worker.Queue
}

func NewLibraryService(repo repositories.LibraryRepository, bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, jobQueue *worker.Queue) LibraryService {
	return &libraryService{
		libraryRepo: repo,
		bookRepo:    bookRepo,
		fileRepo:    fileRepo,
		jobQueue:    jobQueue,
	}
}

func (s *libraryService) CreateLibrary(ctx context.Context, dto *request.CreateLibraryDto) (*models.LibraryEntity, error) {
	lib := &models.LibraryEntity{
		ID:   uuid.Must(uuid.NewV7()).String(),
		Name: dto.Name,
	}
	if err := s.libraryRepo.CreateLibrary(ctx, lib); err != nil {
		return nil, err
	}
	return lib, nil
}

func (s *libraryService) GetLibrary(ctx context.Context, id string) (*models.LibraryEntity, error) {
	return s.libraryRepo.GetLibrary(ctx, id)
}

func (s *libraryService) ListLibraries(ctx context.Context) ([]*models.LibraryEntity, error) {
	return s.libraryRepo.ListLibraries(ctx)
}

func (s *libraryService) UpdateLibrary(ctx context.Context, id string, dto *request.UpdateLibraryDto) (*models.LibraryEntity, error) {
	lib, err := s.libraryRepo.GetLibrary(ctx, id)
	if err != nil {
		return nil, err
	}

	lib.Name = dto.Name

	if err := s.libraryRepo.UpdateLibrary(ctx, lib); err != nil {
		return nil, err
	}
	return lib, nil
}

func (s *libraryService) DeleteLibrary(ctx context.Context, id string) error {
	return s.libraryRepo.DeleteLibrary(ctx, id)
}

func (s *libraryService) UploadFiles(ctx context.Context, libraryID string, files []*multipart.FileHeader) (*models.LibraryUploadResult, error) {
	if _, err := s.libraryRepo.GetLibrary(ctx, libraryID); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	successCount := 0
	pendingJobs := make([]worker.Job, 0, len(files)*2)
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
			s.jobQueue.Enqueue(job)
		}
	}

	return &models.LibraryUploadResult{Uploaded: successCount, Total: len(files)}, nil
}
