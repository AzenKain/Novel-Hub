package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/ebookconv"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
)

const convertBookJobType = "convert_book"

type convertBookPayload struct {
	BookID       string `json:"book_id"`
	FileID       string `json:"file_id"`
	TargetFormat string `json:"target_format"`
}

func (s *bookService) ConvertBook(ctx context.Context, bookID string, fileID string, targetFormat string) (string, error) {
	target := bookparser.NormalizeFormat(targetFormat)
	if !ebookconv.IsTargetSupported(target) {
		return "", apperrors.New(apperrors.ErrBadRequest, "target format not supported")
	}
	file, err := s.bookRepo.GetBookFileById(ctx, fileID)
	if err != nil || file == nil {
		return "", apperrors.New(apperrors.ErrNotFound, "source file not found")
	}
	if file.BookID != bookID {
		return "", apperrors.New(apperrors.ErrBadRequest, "source file does not belong to this book")
	}
	if bookparser.NormalizeFormat(file.Format) == target {
		return "", apperrors.New(apperrors.ErrBadRequest, "source and target format are the same")
	}

	payload, err := jsonx.MarshalString(convertBookPayload{BookID: bookID, FileID: fileID, TargetFormat: target})
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to marshal convert payload")
	}

	jobID := uuid.Must(uuid.NewV7()).String()
	if s.jobQueue != nil {
		if err := s.jobQueue.Enqueue(ctx, worker.Job{ID: jobID, Type: convertBookJobType, Payload: payload}); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to enqueue convert job")
		}
		return jobID, nil
	}
	return jobID, s.ExecuteConvertBookJob(ctx, payload)
}

func (s *bookService) ExecuteConvertBookJob(ctx context.Context, payloadJSON string) error {
	var payload convertBookPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid convert job payload")
	}
	target := bookparser.NormalizeFormat(payload.TargetFormat)
	if !ebookconv.IsTargetSupported(target) {
		return apperrors.New(apperrors.ErrBadRequest, "target format not supported")
	}
	file, err := s.bookRepo.GetBookFileById(ctx, payload.FileID)
	if err != nil || file == nil {
		return apperrors.New(apperrors.ErrNotFound, "source file not found")
	}
	if file.BookID != payload.BookID {
		return apperrors.New(apperrors.ErrBadRequest, "source file does not belong to this book")
	}

	out, err := ebookconv.Convert(s.parsers, file.Format, file.Path, target)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to convert book: "+err.Error())
	}

	tmp, err := os.CreateTemp("", "novelhub-convert-*."+target)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to create temp output")
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	src, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	stem := strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
	saved, err := s.fileRepo.SaveBook(ctx, payload.BookID, stem+"."+target, src)
	_ = src.Close()
	_ = os.Remove(tmpPath)
	if err != nil {
		return err
	}

	return s.bookRepo.UpsertBookFile(ctx, sqlc.UpsertBookFileParams{
		ID:        uuid.Must(uuid.NewV7()).String(),
		BookID:    payload.BookID,
		Path:      saved.Path,
		Format:    target,
		SizeBytes: saved.SizeBytes,
		ModTime:   saved.ModTime,
		Hash:      sql.NullString{},
		State:     sql.NullString{},
	})
}