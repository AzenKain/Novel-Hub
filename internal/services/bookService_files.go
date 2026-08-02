package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/convert"
)

func (s *bookService) GetBookFilePath(ctx context.Context, bookID string) (string, error) {
	file, err := s.GetBookFile(ctx, bookID, "")
	if err != nil {
		return "", err
	}
	return file.Path, nil
}

func (s *bookService) GetBookFile(ctx context.Context, bookID string, fileID string) (*models.BookFileEntity, error) {
	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found for book %s", bookID)
	}
	if fileID != "" {
		for _, file := range files {
			if file.ID == fileID {
				return file, nil
			}
		}
		return nil, fmt.Errorf("file %s not found for book %s", fileID, bookID)
	}
	return s.preferReadableFile(files), nil
}

func (s *bookService) ListBookFiles(ctx context.Context, bookID string) ([]*models.BookFileEntity, error) {
	if _, err := s.bookRepo.GetBook(ctx, bookID); err != nil {
		return nil, err
	}
	return s.bookRepo.GetFilesByBookId(ctx, bookID)
}

func (s *bookService) UploadBookFiles(ctx context.Context, bookID string, files []*multipart.FileHeader) (*models.BookFileUploadResult, error) {
	if _, err := s.bookRepo.GetBook(ctx, bookID); err != nil {
		return nil, err
	}
	successCount := 0
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !isAllowedBookFormat(ext) {
			continue
		}
		src, err := file.Open()
		if err != nil {
			continue
		}
		saved, saveErr := s.fileRepo.SaveBook(ctx, bookID, file.Filename, src)
		closeErr := src.Close()
		if saveErr != nil || closeErr != nil {
			continue
		}
		fileID := uuid.Must(uuid.NewV7()).String()
		state := "managed"
		hash, _ := s.fileRepo.HashSHA256(ctx, saved.Path)
		hashPtr := &hash
		if hash == "" {
			hashPtr = nil
		}
		if err := s.bookRepo.CreateBookFile(ctx, sqlc.CreateBookFileParams{
			ID:        fileID,
			BookID:    bookID,
			Path:      saved.Path,
			Format:    saved.Format,
			SizeBytes: saved.SizeBytes,
			ModTime:   saved.ModTime,
			Hash:      convert.StrPtrToNullString(hashPtr),
			State:     convert.StrPtrToNullString(&state),
		}); err != nil {
			continue
		}

		successCount++
	}
	currentFiles, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil {
		return nil, err
	}
	return &models.BookFileUploadResult{Uploaded: successCount, Total: len(files), Files: currentFiles}, nil
}

func (s *bookService) ProcessSingleLocalFile(ctx context.Context, bookID string, filename string, localFilePath string) error {
	if _, err := s.GetBook(ctx, bookID); err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !isAllowedBookFormat(ext) {
		return fmt.Errorf("unsupported format")
	}

	src, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	saved, saveErr := s.fileRepo.SaveBook(ctx, bookID, filename, src)
	closeErr := src.Close()
	if saveErr != nil || closeErr != nil {
		return fmt.Errorf("failed to save book")
	}

	fileID := uuid.Must(uuid.NewV7()).String()
	state := "managed"
	hash, _ := s.fileRepo.HashSHA256(ctx, saved.Path)
	hashPtr := &hash
	if hash == "" {
		hashPtr = nil
	}

	if err := s.bookRepo.CreateBookFile(ctx, sqlc.CreateBookFileParams{
		ID:        fileID,
		BookID:    bookID,
		Path:      saved.Path,
		Format:    saved.Format,
		SizeBytes: saved.SizeBytes,
		ModTime:   saved.ModTime,
		Hash:      convert.StrPtrToNullString(hashPtr),
		State:     convert.StrPtrToNullString(&state),
	}); err != nil {
		return err
	}

	return nil
}

func (s *bookService) DeleteBookFile(ctx context.Context, fileID string) error {
	fileRecord, err := s.bookRepo.GetBookFileById(ctx, fileID)
	if err != nil {
		return err
	}

	bookID := fileRecord.BookID

	if fileRecord.Path != "" {
		if err := os.Remove(fileRecord.Path); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", fileRecord.Path).Msg("failed to remove physical book file")
		}
	}

	if err := s.bookRepo.DeleteFile(ctx, fileID); err != nil {
		return err
	}

	count, err := s.bookRepo.CountFilesForBook(ctx, bookID)
	if err == nil && count == 0 {
		log.Info().Str("book_id", bookID).Msg("book has no remaining files, deleting book entity and folder")
		if err := s.DeleteBook(ctx, bookID); err != nil {
			log.Error().Err(err).Str("book_id", bookID).Msg("failed to delete book entity after removing last file")
		}
	}

	return nil
}
