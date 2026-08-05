package services

import (
	"context"

	"novelhub/internal/dtos/request"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/convert"
	"novelhub/pkg/mailer"
)

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

	config, err := u.settings.SMTP(ctx)
	if err != nil {
		return err
	}
	if err := mailer.NewSMTPMailer(config).SendEmail(user.Email, dto.Subject, dto.Body, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to send the email")
	}
	return nil
}
