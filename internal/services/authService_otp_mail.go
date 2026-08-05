package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"novelhub/pkg/apperrors"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/mailer"
	"novelhub/pkg/worker"
)

type sendOTPPayload struct {
	Purpose OTPPurpose `json:"purpose"`
	Email   string     `json:"email"`
	Code    string     `json:"code"`
	Minutes int        `json:"minutes"`
}

func otpMailBody(purpose OTPPurpose, code string, minutes int) (string, string) {
	switch purpose {
	case OTPPurposePasswordReset:
		return "[NovelHub] Password reset code",
			fmt.Sprintf("Your password reset code is %s.\n\nIt expires in %d minutes. If you did not request it, ignore this email and your password stays unchanged.", code, minutes)
	default:
		return "[NovelHub] Verify your email address",
			fmt.Sprintf("Your verification code is %s.\n\nIt expires in %d minutes. If you did not request it, ignore this email.", code, minutes)
	}
}

func (a *authService) sendOTP(ctx context.Context, purpose OTPPurpose, email string, code string, minutes int) error {
	payload, err := jsonx.MarshalString(sendOTPPayload{
		Purpose: purpose,
		Email:   email,
		Code:    code,
		Minutes: minutes,
	})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to marshal OTP payload")
	}

	if a.jobQueue != nil {
		jobID := uuid.Must(uuid.NewV7()).String()
		if err := a.jobQueue.Enqueue(ctx, worker.Job{
			ID:      jobID,
			Type:    "send_otp_email",
			Payload: payload,
		}); err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to enqueue OTP email job")
		}
		return nil
	}

	return a.ExecuteSendOTPJob(ctx, payload)
}

func (a *authService) ExecuteSendOTPJob(ctx context.Context, payloadJSON string) error {
	var payload sendOTPPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid OTP job payload")
	}

	config, err := a.settings.SMTP(ctx)
	if err != nil {
		return err
	}
	subject, body := otpMailBody(payload.Purpose, payload.Code, payload.Minutes)
	if err := mailer.NewSMTPMailer(config).SendEmail(payload.Email, subject, body, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to send the verification email")
	}
	return nil
}
