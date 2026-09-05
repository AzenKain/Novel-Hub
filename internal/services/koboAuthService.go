package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
)

// KoboAuthService manages the path token a Kobo device authenticates with.
type KoboAuthService interface {
	EnsureSetup(ctx context.Context, userID, baseURL string) (*response.KoboSetupResponse, error)
	RegenerateSetup(ctx context.Context, userID, baseURL string) (*response.KoboSetupResponse, error)
	RevokeToken(ctx context.Context, userID string) error
}

type koboAuthService struct {
	repo repositories.KoboRepository
}

func NewKoboAuthService(repo repositories.KoboRepository) KoboAuthService {
	return &koboAuthService{repo: repo}
}

func (s *koboAuthService) EnsureSetup(ctx context.Context, userID, baseURL string) (*response.KoboSetupResponse, error) {
	existing, err := s.repo.GetAuthTokenByUser(ctx, userID)
	if err == nil && existing != nil {
		return setupResponse(baseURL, existing.Token), nil
	}
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, err
	}
	return s.RegenerateSetup(ctx, userID, baseURL)
}

func (s *koboAuthService) RegenerateSetup(ctx context.Context, userID, baseURL string) (*response.KoboSetupResponse, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "failed to generate Kobo token")
	}

	record, err := s.repo.UpsertAuthToken(ctx, hex.EncodeToString(buf), userID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ResetSyncedBooks(ctx, userID); err != nil {
		return nil, err
	}
	return setupResponse(baseURL, record.Token), nil
}

func (s *koboAuthService) RevokeToken(ctx context.Context, userID string) error {
	if err := s.repo.DeleteAuthToken(ctx, userID); err != nil {
		return err
	}
	return s.repo.ResetSyncedBooks(ctx, userID)
}

func setupResponse(baseURL, token string) *response.KoboSetupResponse {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpoint := baseURL + "/kobo/" + token

	lower := strings.ToLower(endpoint)
	return &response.KoboSetupResponse{
		EndpointURL: endpoint,
		IsLocalAddress: strings.Contains(lower, "//localhost") ||
			strings.Contains(lower, "//127.0.0.1") ||
			strings.Contains(lower, "//[::1]"),
	}
}
