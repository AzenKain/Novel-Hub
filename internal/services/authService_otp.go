package services

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

func otpPurposeFromString(value string) (OTPPurpose, error) {
	switch value {
	case string(OTPPurposeEmailVerify):
		return OTPPurposeEmailVerify, nil
	case string(OTPPurposePasswordReset):
		return OTPPurposePasswordReset, nil
	default:
		return "", apperrors.New(apperrors.ErrBadRequest, "Unsupported verification purpose")
	}
}

func (a *authService) emailFeatureEnabled(ctx context.Context, purpose OTPPurpose) error {
	settings, err := a.settings.Public(ctx)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to load settings")
	}
	switch purpose {
	case OTPPurposeEmailVerify:
		if !settings.RegistrationEnabled {
			return apperrors.New(apperrors.ErrForbidden, "Public registration is disabled")
		}
		if !settings.RequireEmailVerify {
			return apperrors.New(apperrors.ErrForbidden, "Email verification is disabled")
		}
	case OTPPurposePasswordReset:
		if !settings.PasswordResetEnabled {
			return apperrors.New(apperrors.ErrForbidden, "Password reset is disabled")
		}
	}
	return nil
}

// Reports the same success for unknown addresses, or this becomes a membership oracle.
func (a *authService) RequestOTP(ctx context.Context, dto *request.RequestOTPDto) (*response.OTPRequestResponse, error) {
	purpose, err := otpPurposeFromString(dto.Purpose)
	if err != nil {
		return nil, err
	}
	if err := a.emailFeatureEnabled(ctx, purpose); err != nil {
		return nil, err
	}
	email := dto.Email
	if !constants.EMAIL_REGEX.MatchString(email) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid email format")
	}
	if a.otp == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Verification codes are unavailable")
	}
	if _, err := a.settings.SMTP(ctx); err != nil {
		return nil, err
	}

	existing, err := a.userRepo.GetByEmail(ctx, email)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	payload := &response.OTPRequestResponse{
		ExpiresInSeconds: int(constants.OTPDuration / time.Second),
		CooldownSeconds:  int(constants.OTPCooldown / time.Second),
	}

	wanted := (purpose == OTPPurposeEmailVerify && existing == nil) ||
		(purpose == OTPPurposePasswordReset && existing != nil)
	if !wanted {
		return payload, nil
	}

	code, err := a.otp.Issue(ctx, purpose, email)
	if err != nil {
		return nil, err
	}
	if err := a.sendOTP(ctx, purpose, email, code, int(constants.OTPDuration/time.Minute)); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *authService) VerifyOTP(ctx context.Context, dto *request.VerifyOTPDto) (*response.OTPVerifyResponse, error) {
	purpose, err := otpPurposeFromString(dto.Purpose)
	if err != nil {
		return nil, err
	}
	if err := a.emailFeatureEnabled(ctx, purpose); err != nil {
		return nil, err
	}
	if a.otp == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Verification codes are unavailable")
	}
	ticket, err := a.otp.Verify(ctx, purpose, dto.Email, dto.Code)
	if err != nil {
		return nil, err
	}
	return &response.OTPVerifyResponse{
		OTPTicket:        ticket,
		ExpiresInSeconds: int(constants.OTPVerifiedDuration / time.Second),
	}, nil
}

func (a *authService) ResetPasswordWithOTP(ctx context.Context, dto *request.ResetPasswordWithOTPDto) error {
	if err := a.emailFeatureEnabled(ctx, OTPPurposePasswordReset); err != nil {
		return err
	}
	if err := constants.ValidatePassword(dto.NewPassword); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	if a.otp == nil {
		return apperrors.New(apperrors.ErrInternalError, "Verification codes are unavailable")
	}
	if err := a.otp.Consume(ctx, OTPPurposePasswordReset, dto.Email, dto.OTPTicket); err != nil {
		return err
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}
	id, err := convert.ParseID(user.ID)
	if err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid user ID")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}

	tx, err := a.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	userRepoTx := a.userRepo.WithTx(tx)

	if err := userRepoTx.UpdatePassword(ctx, id, string(hashed)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, id, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to clear refresh token")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit password reset")
	}
	a.userRepo.InvalidateUserCache(ctx, id, user.Email)
	return nil
}
