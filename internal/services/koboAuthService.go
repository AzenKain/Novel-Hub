package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"
)

// KoboAuthService manages the path token a Kobo device authenticates with.
//
// baseURL is passed in rather than read from config here because the controller already knows
// the scheme and host the request arrived on, and a request-derived value is right in more
// deployments than a single configured one.
type KoboAuthService interface {
	// EnsureSetup returns the caller's endpoint URL, creating a token if absent. Called when
	// the user opens the Kobo setup card, so it must be idempotent — regenerating on every
	// view would silently unpair a working device.
	EnsureSetup(ctx context.Context, userID, baseURL string) (*response.KoboSetupResponse, error)
	// RegenerateSetup replaces the token, which is how a user revokes a lost reader.
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
	// Matches calibre-web's hexlify(urandom(16)): 128 bits, hex-encoded. The token travels in a
	// URL path, so hex keeps it free of characters needing escaping.
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "failed to generate Kobo token")
	}

	record, err := s.repo.UpsertAuthToken(ctx, hex.EncodeToString(buf), userID)
	if err != nil {
		return nil, err
	}
	// A new token means the device must re-sync from scratch; clearing the synced-books
	// record keeps "NewEntitlement vs ChangedEntitlement" honest for the replacement device.
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

// setupResponse builds the api_endpoint value the user pastes into the device, and flags the one
// mistake that breaks this setup silently.
//
// The token segment comes before /v1/... because the device appends that itself. baseURL falls
// back to SERVER_URL when the caller has nothing better; a Kobo cannot resolve "localhost", so
// a loopback address is reported back rather than rejected — calibre-web warns about exactly
// this case when the setup page is opened over loopback.
func setupResponse(baseURL, token string) *response.KoboSetupResponse {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(config.GetConfigWithDefault("SERVER_URL", "")), "/")
	}
	endpoint := baseURL + "/kobo/" + token

	lower := strings.ToLower(endpoint)
	return &response.KoboSetupResponse{
		EndpointURL: endpoint,
		IsLocalAddress: strings.Contains(lower, "//localhost") ||
			strings.Contains(lower, "//127.0.0.1") ||
			strings.Contains(lower, "//[::1]"),
	}
}
