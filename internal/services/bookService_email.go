package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/mailer"
)

func (s *bookService) SendBookToEmail(ctx context.Context, bookID string, recipientEmail string, claims *response.JWTClaims) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	if !s.CanDownloadBook(ctx, book, claims) {
		return apperrors.New(apperrors.ErrForbidden, "Download permission denied")
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "No files available for this book")
	}

	targetFile := files[0]
	info, err := os.Stat(targetFile.Path)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to inspect book file")
	}

	if info.Size() > 50*1024*1024 {
		return apperrors.New(apperrors.ErrBadRequest, "Book file exceeds 50MB email attachment size limit")
	}

	attachment := &mailer.Attachment{
		Filename: filepath.Base(targetFile.Path),
		Path:     targetFile.Path,
	}

	smtpConfig, err := s.settings.SMTP(ctx)
	if err != nil {
		return err
	}
	m := mailer.NewSMTPMailer(smtpConfig)

	subject := fmt.Sprintf("[NovelHub] Send to Kindle: %s", book.Title)
	body := fmt.Sprintf("Enjoy reading '%s' on your Kindle or e-reader device!", book.Title)

	if err := m.SendEmail(recipientEmail, subject, body, attachment); err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to dispatch email: %v", err))
	}

	return nil
}
