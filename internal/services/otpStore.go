package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/google/uuid"

	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
)

type OTPPurpose string

const (
	OTPPurposeEmailVerify   OTPPurpose = "email_verify"
	OTPPurposePasswordReset OTPPurpose = "password_reset"
)

type otpEntry struct {
	Digest   string `json:"digest"`
	Attempts int    `json:"attempts"`
}

type OTPStore struct {
	cache cache.Cache
}

func NewOTPStore(c cache.Cache) *OTPStore { return &OTPStore{cache: c} }

func otpDigest(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func otpKey(purpose OTPPurpose, email string) string {
	return cache.BuildKey("otp", string(purpose), email)
}

func otpCooldownKey(purpose OTPPurpose, email string) string {
	return cache.BuildKey("otp_cooldown", string(purpose), email)
}

func otpVerifiedKey(purpose OTPPurpose, email string, ticket string) string {
	return cache.BuildKey("otp_verified", string(purpose), email, ticket)
}

func (s *OTPStore) Issue(ctx context.Context, purpose OTPPurpose, email string) (string, error) {
	if s.cache == nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Verification codes are unavailable")
	}
	cooldown := otpCooldownKey(purpose, email)
	if exists, _ := s.cache.Exists(ctx, cooldown); exists {
		return "", apperrors.New(apperrors.ErrTooManyRequests, "A code was already sent, please wait before requesting another")
	}

	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to generate a verification code")
	}
	code := fmt.Sprintf("%06d", number.Int64())

	entry := otpEntry{Digest: otpDigest(code)}
	if err := s.cache.Set(ctx, otpKey(purpose, email), entry, constants.OTPDuration); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to store the verification code")
	}
	if err := s.cache.Set(ctx, cooldown, true, constants.OTPCooldown); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to store the verification code")
	}
	return code, nil
}

func (s *OTPStore) Verify(ctx context.Context, purpose OTPPurpose, email string, code string) (string, error) {
	if s.cache == nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Verification codes are unavailable")
	}
	key := otpKey(purpose, email)
	var entry otpEntry
	if err := s.cache.Get(ctx, key, &entry); err != nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "The code is invalid or has expired")
	}
	if entry.Attempts >= constants.OTPMaxAttempts {
		_ = s.cache.Del(ctx, key)
		return "", apperrors.New(apperrors.ErrTooManyRequests, "Too many incorrect attempts, request a new code")
	}
	if subtle.ConstantTimeCompare([]byte(entry.Digest), []byte(otpDigest(code))) != 1 {
		entry.Attempts++
		_ = s.cache.Set(ctx, key, entry, constants.OTPDuration)
		return "", apperrors.New(apperrors.ErrBadRequest, "The code is invalid or has expired")
	}

	_ = s.cache.Del(ctx, key)
	ticket := uuid.Must(uuid.NewV7()).String()
	if err := s.cache.Set(ctx, otpVerifiedKey(purpose, email, ticket), true, constants.OTPVerifiedDuration); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to record the verification")
	}
	return ticket, nil
}

func (s *OTPStore) Consume(ctx context.Context, purpose OTPPurpose, email string, ticket string) error {
	if s.cache == nil {
		return apperrors.New(apperrors.ErrInternalError, "Verification codes are unavailable")
	}
	key := otpVerifiedKey(purpose, email, ticket)
	exists, err := s.cache.Exists(ctx, key)
	if err != nil || !exists {
		return apperrors.New(apperrors.ErrBadRequest, "Email verification is missing or has expired")
	}
	_ = s.cache.Del(ctx, key)
	return nil
}
