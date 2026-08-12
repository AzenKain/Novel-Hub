package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
)

type MagicCodeService interface {
	RequestMagicCode(ctx context.Context, dto *request.RequestMagicCodeDto, baseURL string) (*response.RequestMagicCodeResponse, error)
	ActivateMagicCode(ctx context.Context, userID string, dto *request.ActivateMagicCodeDto) (*response.AuthResponse, error)
	PollMagicCode(ctx context.Context, dto *request.PollMagicCodeDto) (*response.PollMagicCodeResponse, error)
}

type magicCodeService struct {
	magicCodeRepo repositories.MagicCodeRepository
	userRepo      repositories.UserRepository
	authService   AuthService
}

func NewMagicCodeService(magicCodeRepo repositories.MagicCodeRepository, userRepo repositories.UserRepository, authService AuthService) MagicCodeService {
	return &magicCodeService{
		magicCodeRepo: magicCodeRepo,
		userRepo:      userRepo,
		authService:   authService,
	}
}

func generate6DigitCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func (s *magicCodeService) RequestMagicCode(ctx context.Context, dto *request.RequestMagicCodeDto, baseURL string) (*response.RequestMagicCodeResponse, error) {
	deviceInfo := "eReader / Device"
	if dto != nil && strings.TrimSpace(dto.DeviceInfo) != "" {
		deviceInfo = strings.TrimSpace(dto.DeviceInfo)
	}

	code := generate6DigitCode()
	formattedCode := fmt.Sprintf("%s-%s", code[:3], code[3:])
	id := uuid.New().String()
	pollToken := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	if err := s.magicCodeRepo.Create(ctx, id, code, pollToken, deviceInfo, expiresAt); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to request magic code")
	}

	activateURL := fmt.Sprintf("%s/activate?code=%s", strings.TrimRight(baseURL, "/"), code)

	return &response.RequestMagicCodeResponse{
		Code:             formattedCode,
		PollToken:        pollToken,
		ActivateURL:      activateURL,
		ExpiresInSeconds: 300,
	}, nil
}

func (s *magicCodeService) ActivateMagicCode(ctx context.Context, userID string, dto *request.ActivateMagicCodeDto) (*response.AuthResponse, error) {
	if dto == nil || strings.TrimSpace(dto.Code) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Magic code is required")
	}

	cleanCode := strings.ReplaceAll(strings.TrimSpace(dto.Code), "-", "")
	cleanCode = strings.ReplaceAll(cleanCode, " ", "")

	record, err := s.magicCodeRepo.GetByCode(ctx, cleanCode)
	if err != nil || record == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Magic code expired or invalid")
	}

	if record.Status != "pending" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Magic code is already activated or expired")
	}

	if time.Now().After(record.ExpiresAt) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Magic code has expired")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	authResp, err := s.authService.GenToken(user)
	if err != nil {
		return nil, err
	}

	if err := s.magicCodeRepo.Activate(ctx, cleanCode, userID, authResp.AccessToken); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to activate magic code")
	}

	return authResp, nil
}

func (s *magicCodeService) PollMagicCode(ctx context.Context, dto *request.PollMagicCodeDto) (*response.PollMagicCodeResponse, error) {
	if dto == nil || strings.TrimSpace(dto.PollToken) == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Poll token is required")
	}

	pollToken := strings.TrimSpace(dto.PollToken)
	record, err := s.magicCodeRepo.GetByPollToken(ctx, pollToken)
	if err != nil || record == nil {
		return &response.PollMagicCodeResponse{Status: "expired"}, nil
	}

	if time.Now().After(record.ExpiresAt) {
		return &response.PollMagicCodeResponse{Status: "expired"}, nil
	}

	if record.Status == "active" {
		_ = s.magicCodeRepo.MarkUsed(ctx, pollToken)
		return &response.PollMagicCodeResponse{
			Status: "active",
			AuthResponse: &response.AuthResponse{
				AccessToken: record.JWTToken,
			},
		}, nil
	}

	return &response.PollMagicCodeResponse{
		Status: record.Status,
	}, nil
}
