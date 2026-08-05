package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/mailer"
	"novelhub/pkg/worker"
)

type sendBookEmailPayload struct {
	BookID         string `json:"book_id"`
	RecipientEmail string `json:"recipient_email"`
}

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

	maxAttachmentMB := config.GetIntConfigWithDefault("MAX_EMAIL_ATTACHMENT_MB", 50)
	if info.Size() > int64(maxAttachmentMB)*1024*1024 {
		return apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("Book file exceeds %dMB email attachment size limit", maxAttachmentMB))
	}

	payload, err := jsonx.MarshalString(sendBookEmailPayload{
		BookID:         bookID,
		RecipientEmail: recipientEmail,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal email payload")
	}

	if s.jobQueue != nil {
		jobID := uuid.Must(uuid.NewV7()).String()
		if err := s.jobQueue.Enqueue(ctx, worker.Job{
			ID:      jobID,
			Type:    "send_book_email",
			Payload: payload,
		}); err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to enqueue email dispatch job")
		}
		return nil
	}

	return s.ExecuteSendBookEmailJob(ctx, payload)
}

func (s *bookService) ExecuteSendBookEmailJob(ctx context.Context, payloadJSON string) error {
	var payload sendBookEmailPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid email job payload")
	}

	book, err := s.bookRepo.GetBook(ctx, payload.BookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, payload.BookID)
	if err != nil || len(files) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "No files available for this book")
	}

	targetFile := files[0]
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

	if err := m.SendEmail(payload.RecipientEmail, subject, body, attachment); err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to dispatch email: %v", err))
	}

	return nil
}
