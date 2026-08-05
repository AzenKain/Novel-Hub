package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/mailer"
	"novelhub/pkg/worker"
)

type sendUserEmailPayload struct {
	UserID  string `json:"user_id"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (u *userService) SendEmail(ctx context.Context, userID string, dto *request.SendUserEmailDto) error {
	if u.settings == nil {
		return apperrors.New(apperrors.ErrInternalError, "Email delivery is not available")
	}
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil && !apperrors.IsNotFound(err) {
		return apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	payload, err := jsonx.MarshalString(sendUserEmailPayload{
		UserID:  userID,
		Subject: dto.Subject,
		Body:    dto.Body,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal email payload")
	}

	if u.jobQueue != nil {
		jobID := uuid.Must(uuid.NewV7()).String()
		if err := u.jobQueue.Enqueue(ctx, worker.Job{
			ID:      jobID,
			Type:    "send_user_email",
			Payload: payload,
		}); err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to enqueue user email job")
		}
		return nil
	}

	return u.ExecuteSendUserEmailJob(ctx, payload)
}

func (u *userService) ExecuteSendUserEmailJob(ctx context.Context, payloadJSON string) error {
	var payload sendUserEmailPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid email job payload")
	}

	id, ferr := convert.ParseID(payload.UserID)
	if ferr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	config, err := u.settings.SMTP(ctx)
	if err != nil {
		return err
	}
	if err := mailer.NewSMTPMailer(config).SendEmail(user.Email, payload.Subject, payload.Body, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to send email: %v", err))
	}
	return nil
}
