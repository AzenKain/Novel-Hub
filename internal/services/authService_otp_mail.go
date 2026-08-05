package services

import (
	"context"
	"fmt"

	"novelhub/pkg/apperrors"
	"novelhub/pkg/mailer"
)

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
	config, err := a.settings.SMTP(ctx)
	if err != nil {
		return err
	}
	subject, body := otpMailBody(purpose, code, minutes)
	if err := mailer.NewSMTPMailer(config).SendEmail(email, subject, body, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to send the verification email")
	}
	return nil
}
