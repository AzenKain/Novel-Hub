package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/jsonx"
)

func (s *bookService) ValidateBookEPUB(ctx context.Context, bookID string, fileID string, claims *response.JWTClaims) (*response.ValidationReportResponse, error) {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if !s.CanReadBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Forbidden to read this book")
	}

	file, err := s.GetBookFile(ctx, bookID, fileID)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(file.Path))
	if ext != ".epub" && file.Format != "epub" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Book Doctor currently supports EPUB files only")
	}

	report, err := epub.ValidateEPUB(file.Path)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to validate EPUB: %v", err))
	}

	issues := make([]response.ValidationIssueResponse, 0, len(report.Issues))
	for _, issue := range report.Issues {
		issues = append(issues, response.ValidationIssueResponse{
			Severity: issue.Severity,
			Code:     issue.Code,
			File:     issue.File,
			Message:  issue.Message,
			Fixable:  issue.Fixable,
			FixID:    issue.FixID,
		})
	}

	return &response.ValidationReportResponse{
		Valid:    report.Valid,
		Errors:   report.Errors,
		Warnings: report.Warnings,
		Infos:    report.Infos,
		Issues:   issues,
	}, nil
}

func (s *bookService) RepairBookEPUB(ctx context.Context, bookID string, fileID string, req *request.RepairBookRequest, claims *response.JWTClaims) (*response.BookRepairResponse, error) {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if !s.CanRepairBook(ctx, book, claims) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Forbidden to repair this book")
	}

	file, err := s.GetBookFile(ctx, bookID, fileID)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(file.Path))
	if ext != ".epub" && file.Format != "epub" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Book Doctor currently supports EPUB files only")
	}

	opts := epub.DefaultRepairOptions()
	if req != nil {
		if req.NormalizeMimetype != nil {
			opts.NormalizeMimetype = *req.NormalizeMimetype
		}
		if req.FixContainer != nil {
			opts.FixContainer = *req.FixContainer
		}
		if req.FixXHTML != nil {
			opts.FixXHTML = *req.FixXHTML
		}
		if req.ReconcileManifest != nil {
			opts.ReconcileManifest = *req.ReconcileManifest
		}
		if req.ReconcileSpine != nil {
			opts.ReconcileSpine = *req.ReconcileSpine
		}
		if req.FixTOC != nil {
			opts.FixTOC = *req.FixTOC
		}
		if req.CleanBrokenLinks != nil {
			opts.CleanBrokenLinks = *req.CleanBrokenLinks
		}
		if req.FixMetadata != nil {
			opts.FixMetadata = *req.FixMetadata
		}
	}

	// Perform in-place repair via atomic temp file
	tmpRepaired := file.Path + ".doctor_tmp"
	defer os.Remove(tmpRepaired)

	res, err := epub.RepairEPUB(file.Path, tmpRepaired, opts)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("EPUB repair failed: %v", err))
	}

	// Overwrite original file
	if err := os.Rename(tmpRepaired, file.Path); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to apply repaired EPUB: %v", err))
	}

	if s.fileRepo != nil {
		if hashStr, err := s.fileRepo.HashSHA256(ctx, file.Path); err == nil {
			_ = s.bookRepo.UpdateBookFileHash(ctx, file.ID, hashStr)
		}
	}

	// Re-extract metadata in background if needed
	if s.parsers != nil {
		_ = s.ExtractMetadata(ctx, bookID)
	}

	issues := make([]response.ValidationIssueResponse, 0, len(res.Report.Issues))
	for _, issue := range res.Report.Issues {
		issues = append(issues, response.ValidationIssueResponse{
			Severity: issue.Severity,
			Code:     issue.Code,
			File:     issue.File,
			Message:  issue.Message,
			Fixable:  issue.Fixable,
			FixID:    issue.FixID,
		})
	}

	return &response.BookRepairResponse{
		Success:    res.Success,
		FixedCount: res.FixedCount,
		Logs:       res.Logs,
		Report: response.ValidationReportResponse{
			Valid:    res.Report.Valid,
			Errors:   res.Report.Errors,
			Warnings: res.Report.Warnings,
			Infos:    res.Report.Infos,
			Issues:   issues,
		},
	}, nil
}

func (s *bookService) ExecuteBatchRepairBooksJob(ctx context.Context, payloadJSON string) error {
	var targetLibID string
	if strings.TrimSpace(payloadJSON) != "" {
		var payload struct {
			LibraryID string `json:"library_id"`
		}
		if err := jsonx.UnmarshalString(payloadJSON, &payload); err == nil {
			targetLibID = strings.TrimSpace(payload.LibraryID)
		} else {
			targetLibID = strings.TrimSpace(payloadJSON)
		}
	}

	var cursor *time.Time
	cursorID := ""
	limit := int64(100)
	opts := epub.DefaultRepairOptions()

	totalScanned := 0
	totalRepaired := 0

	for {
		ids, err := s.bookRepo.ListBookIDs(ctx, cursor, cursorID, limit)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}

		books, err := s.bookRepo.GetBooksByIDs(ctx, ids)
		if err != nil {
			return err
		}

		for _, book := range books {
			if book == nil {
				continue
			}
			if targetLibID != "" && book.LibraryID != targetLibID {
				continue
			}

			files, err := s.bookRepo.GetFilesByBookId(ctx, book.ID)
			if err != nil || len(files) == 0 {
				continue
			}

			for _, f := range files {
				if f == nil {
					continue
				}
				ext := strings.ToLower(filepath.Ext(f.Path))
				if ext != ".epub" && f.Format != "epub" {
					continue
				}

				totalScanned++
				report, err := epub.ValidateEPUB(f.Path)
				if err != nil || report.Valid {
					continue
				}

				tmpRepaired := f.Path + ".doctor_tmp"
				res, err := epub.RepairEPUB(f.Path, tmpRepaired, opts)
				if err != nil || !res.Success {
					_ = os.Remove(tmpRepaired)
					continue
				}

				if err := os.Rename(tmpRepaired, f.Path); err == nil {
					totalRepaired++
					if s.fileRepo != nil {
						if hashStr, err := s.fileRepo.HashSHA256(ctx, f.Path); err == nil {
							_ = s.bookRepo.UpdateBookFileHash(ctx, f.ID, hashStr)
						}
					}
					_ = s.ExtractMetadata(ctx, book.ID)
				} else {
					_ = os.Remove(tmpRepaired)
				}
			}
		}

		if len(ids) < int(limit) {
			break
		}
		lastBook := books[len(books)-1]
		if lastBook != nil {
			cursor = &lastBook.CreatedAt
			cursorID = lastBook.ID
		} else {
			break
		}
	}

	log.Info().Int("scanned", totalScanned).Int("repaired", totalRepaired).Msg("Batch EPUB repair job completed")
	return nil
}
