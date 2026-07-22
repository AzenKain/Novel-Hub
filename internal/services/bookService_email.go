package services

import (
	"context"
	"fmt"
	"os"

	"novelhub/pkg/apperrors"
	"novelhub/pkg/mailer"
)

func (s *bookService) SendBookToEmail(ctx context.Context, bookID string, recipientEmail string) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "No files available for this book")
	}

	targetFile := files[0]
	data, err := os.ReadFile(targetFile.Path)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to read book file content")
	}

	if len(data) > 50*1024*1024 {
		return apperrors.New(apperrors.ErrBadRequest, "Book file exceeds 50MB email attachment size limit")
	}

	settings, _ := s.settings.Public(ctx)
	smtpHost := "smtp.gmail.com"
	smtpPort := 587
	smtpUser := ""
	smtpPass := ""
	fromEmail := "noreply@novelhub.local"

	if settings != nil {
		// Use settings if configured
	}

	m := mailer.NewSMTPMailer(mailer.SMTPConfig{
		Host:      smtpHost,
		Port:      smtpPort,
		Username:  smtpUser,
		Password:  smtpPass,
		FromEmail: fromEmail,
	})

	subject := fmt.Sprintf("[NovelHub] Send to Kindle: %s", book.Title)
	body := fmt.Sprintf("Enjoy reading '%s' on your Kindle or e-reader device!", book.Title)

	attachment := &mailer.Attachment{
		Filename: targetFile.Path,
		Data:     data,
	}

	if err := m.SendEmail(recipientEmail, subject, body, attachment); err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to dispatch email: %v", err))
	}

	return nil
}
