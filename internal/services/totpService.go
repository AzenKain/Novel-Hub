package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/crypto"
	"novelhub/pkg/totp"
)

const (
	recoveryCodeCount  = 10
	recoveryCodeLength = 10
	totpIssuer         = "NovelHub"
)

type TOTPService interface {
	Status(ctx context.Context, userID string) (*response.TOTPStatusResponse, error)
	BeginEnrollment(ctx context.Context, userID, email string) (*response.TOTPEnrollResponse, error)
	ConfirmEnrollment(ctx context.Context, userID, code string) (*response.TOTPRecoveryCodesResponse, error)
	Disable(ctx context.Context, userID, code string) error
	RegenerateRecoveryCodes(ctx context.Context, userID, code string) (*response.TOTPRecoveryCodesResponse, error)
	Enabled(ctx context.Context, userID string) bool
	VerifyLogin(ctx context.Context, userID, code string) error
}

type totpService struct {
	repo  repositories.TOTPRepository
	cache cache.Cache
}

func NewTOTPService(repo repositories.TOTPRepository, c cache.Cache) TOTPService {
	return &totpService{repo: repo, cache: c}
}

func recoveryCodeHash(code string) string {
	sum := sha256.Sum256([]byte(normalizeRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}

func normalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

func generateRecoveryCode() (string, error) {
	const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	buf := make([]byte, recoveryCodeLength)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		buf[i] = alphabet[n.Int64()]
	}
	return string(buf[:5]) + "-" + string(buf[5:]), nil
}

func (s *totpService) confirmedSecret(ctx context.Context, userID string) (string, error) {
	entity, err := s.repo.Get(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return "", apperrors.New(apperrors.ErrBadRequest, "Two-factor authentication is not enabled")
		}
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to load two-factor settings")
	}
	if !entity.IsConfirmed() {
		return "", apperrors.New(apperrors.ErrBadRequest, "Two-factor authentication is not enabled")
	}
	secret, err := crypto.DecryptAES(entity.Secret)
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to read the two-factor secret")
	}
	return secret, nil
}

func (s *totpService) Enabled(ctx context.Context, userID string) bool {
	entity, err := s.repo.Get(ctx, userID)
	return err == nil && entity.IsConfirmed()
}

func (s *totpService) Status(ctx context.Context, userID string) (*response.TOTPStatusResponse, error) {
	entity, err := s.repo.Get(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return &response.TOTPStatusResponse{}, nil
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load two-factor settings")
	}
	if !entity.IsConfirmed() {
		return &response.TOTPStatusResponse{PendingEnrollment: true}, nil
	}
	remaining, err := s.repo.CountUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to count recovery codes")
	}
	return &response.TOTPStatusResponse{
		Enabled:                true,
		ConfirmedAt:            entity.ConfirmedAt,
		RecoveryCodesRemaining: int(remaining),
	}, nil
}

func (s *totpService) BeginEnrollment(ctx context.Context, userID, email string) (*response.TOTPEnrollResponse, error) {
	if existing, err := s.repo.Get(ctx, userID); err == nil && existing.IsConfirmed() {
		return nil, apperrors.New(apperrors.ErrConflict, "Two-factor authentication is already enabled")
	}
	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to generate a secret")
	}
	encrypted, err := crypto.EncryptAES(secret)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to protect the secret")
	}
	if _, err := s.repo.Upsert(ctx, userID, encrypted); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to store the secret")
	}
	return &response.TOTPEnrollResponse{
		Secret:          secret,
		ProvisioningURI: totp.ProvisioningURI(totpIssuer, email, secret),
	}, nil
}

func (s *totpService) ConfirmEnrollment(ctx context.Context, userID, code string) (*response.TOTPRecoveryCodesResponse, error) {
	entity, err := s.repo.Get(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Start the setup before confirming a code")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load two-factor settings")
	}
	if entity.IsConfirmed() {
		return nil, apperrors.New(apperrors.ErrConflict, "Two-factor authentication is already enabled")
	}
	secret, err := crypto.DecryptAES(entity.Secret)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to read the two-factor secret")
	}
	if err := s.checkCode(ctx, userID, secret, code); err != nil {
		return nil, err
	}
	if _, err := s.repo.Confirm(ctx, userID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to enable two-factor authentication")
	}
	return s.issueRecoveryCodes(ctx, userID)
}

func (s *totpService) Disable(ctx context.Context, userID, code string) error {
	secret, err := s.confirmedSecret(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.checkCodeOrRecovery(ctx, userID, secret, code); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, userID); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to disable two-factor authentication")
	}
	return nil
}

func (s *totpService) RegenerateRecoveryCodes(ctx context.Context, userID, code string) (*response.TOTPRecoveryCodesResponse, error) {
	secret, err := s.confirmedSecret(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, userID, secret, code); err != nil {
		return nil, err
	}
	return s.issueRecoveryCodes(ctx, userID)
}

func (s *totpService) VerifyLogin(ctx context.Context, userID, code string) error {
	secret, err := s.confirmedSecret(ctx, userID)
	if err != nil {
		return err
	}
	return s.checkCodeOrRecovery(ctx, userID, secret, code)
}

func (s *totpService) issueRecoveryCodes(ctx context.Context, userID string) (*response.TOTPRecoveryCodesResponse, error) {
	codes := make([]string, 0, recoveryCodeCount)
	hashes := make([]string, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to generate recovery codes")
		}
		codes = append(codes, code)
		hashes = append(hashes, recoveryCodeHash(code))
	}
	if err := s.repo.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to store recovery codes")
	}
	return &response.TOTPRecoveryCodesResponse{Codes: codes}, nil
}

// A TOTP code stays valid across three steps, so without burning the counter a code read over
// someone's shoulder works again for up to 90 seconds.
func (s *totpService) checkCode(ctx context.Context, userID, secret, code string) error {
	counter, ok := totp.ValidateWithCounter(secret, code, time.Now())
	if !ok {
		return apperrors.New(apperrors.ErrBadRequest, "The code is invalid or has expired")
	}
	if s.cache != nil {
		key := cache.BuildKey("totp", "used", userID, fmt.Sprintf("%d", counter))
		if used, _ := s.cache.Exists(ctx, key); used {
			return apperrors.New(apperrors.ErrBadRequest, "That code has already been used")
		}
		_ = s.cache.Set(ctx, key, true, constants.TOTPReplayWindow)
	}
	return nil
}

func (s *totpService) checkCodeOrRecovery(ctx context.Context, userID, secret, code string) error {
	trimmed := strings.TrimSpace(code)
	if len(trimmed) == totp.Digits {
		return s.checkCode(ctx, userID, secret, trimmed)
	}
	consumed, err := s.repo.ConsumeRecoveryCode(ctx, userID, recoveryCodeHash(trimmed))
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to check the recovery code")
	}
	if !consumed {
		return apperrors.New(apperrors.ErrBadRequest, "The code is invalid or has expired")
	}
	return nil
}
