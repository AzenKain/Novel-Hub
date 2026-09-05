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
	"novelhub/pkg/apperrors"
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
		return nil, apperrors.New(apperrors.ErrNotFound, "This book has no files")
	}
	if fileID != "" {
		for _, file := range files {
			if file.ID == fileID {
				return file, nil
			}
		}
		return nil, apperrors.New(apperrors.ErrNotFound, "File not found for this book")
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
		return apperrors.New(apperrors.ErrBadRequest, "Unsupported file format")
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

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txBookRepo := s.bookRepo.WithTx(tx)

	survivors, err := txBookRepo.GetFilesByBookId(ctx, fileRecord.BookID)
	if err != nil {
		return err
	}
	keep := s.preferReadableFileExcept(survivors, fileID)
	if keep == nil {
		book, err := s.bookRepo.GetBook(ctx, fileRecord.BookID)
		if err != nil {
			return err
		}
		if err := txBookRepo.DeleteFTSBook(ctx, fileRecord.BookID); err != nil {
			return err
		}
		if err := txBookRepo.DeleteBook(ctx, fileRecord.BookID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		txBookRepo.FlushCache(ctx)

		if err := s.fileRepo.RemoveBookDir(ctx, fileRecord.BookID); err != nil {
			log.Warn().Err(err).Str("book_id", fileRecord.BookID).Msg("failed to remove book files")
		}
		if book != nil && s.webhookService != nil {
			s.webhookService.DispatchEvent(ctx, "book.deleted", BuildBookWebhookPayload(book))
		}
		return nil
	}
	if err := txBookRepo.RepointFileUserData(ctx, fileID, keep.ID); err != nil {
		return err
	}

	if err := txBookRepo.DeleteFile(ctx, fileID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	txBookRepo.FlushCache(ctx)

	if fileRecord.Path != "" {
		if err := os.Remove(fileRecord.Path); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", fileRecord.Path).Msg("failed to remove physical book file")
		}
	}
	return nil
}

func (s *bookService) preferReadableFileExcept(files []*models.BookFileEntity, excludeID string) *models.BookFileEntity {
	remaining := make([]*models.BookFileEntity, 0, len(files))
	for _, file := range files {
		if file != nil && file.ID != excludeID {
			remaining = append(remaining, file)
		}
	}
	if len(remaining) == 0 {
		return nil
	}
	return s.preferReadableFile(remaining)
}
