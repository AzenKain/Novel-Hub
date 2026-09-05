package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

var (
	hardcoverAuthorizeURL = "https://hardcover.app/oauth/authorize"
	hardcoverTokenURL     = "https://hardcover.app/oauth/token"
	hardcoverGraphqlURL   = "https://api.hardcover.app/v1/graphql"
)

type ScrobbleService interface {
	GetHardcoverAuthorizeURL(ctx context.Context, userID string, redirectURI string) (string, error)
	HandleHardcoverCallback(ctx context.Context, code string, state string) error
	SyncHardcoverProgress(ctx context.Context, userID string, bookID string, progress int, claims *response.JWTClaims) error
}

type scrobbleService struct {
	trackerRepo repositories.TrackerRepository
	bookRepo    repositories.BookDBRepository
	settings    SettingsService
	cache       cache.Cache
	permissions PermissionCache
	httpClient  *http.Client
}

func NewScrobbleService(trackerRepo repositories.TrackerRepository, bookRepo repositories.BookDBRepository, settings SettingsService, c cache.Cache, permissions PermissionCache) ScrobbleService {
	return &scrobbleService{
		trackerRepo: trackerRepo,
		bookRepo:    bookRepo,
		settings:    settings,
		cache:       c,
		permissions: permissions,
		httpClient:  netx.NewSafeHTTPClient(15 * time.Second),
	}
}

func (s *scrobbleService) hardcoverConfig(ctx context.Context) (*models.HardcoverConfig, error) {
	config, err := s.settings.HardcoverConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !config.Enabled || config.ClientID == "" || config.ClientSecret == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Hardcover scrobbling is not configured")
	}
	return config, nil
}

func (s *scrobbleService) GetHardcoverAuthorizeURL(ctx context.Context, userID string, redirectURI string) (string, error) {
	config, err := s.hardcoverConfig(ctx)
	if err != nil {
		return "", err
	}

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to generate OAuth state")
	}
	state := hex.EncodeToString(stateBytes)
	if s.cache != nil {
		_ = s.cache.Set(ctx, cache.BuildKey("hardcover_oauth", "state", state), userID, 10*time.Minute)
	}

	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", "read,write")
	params.Set("state", state)
	return hardcoverAuthorizeURL + "?" + params.Encode(), nil
}

func (s *scrobbleService) HandleHardcoverCallback(ctx context.Context, code string, state string) error {
	if s.cache != nil {
		var userID string
		if err := s.cache.Get(ctx, cache.BuildKey("hardcover_oauth", "state", state), &userID); err != nil || userID == "" {
			return apperrors.New(apperrors.ErrBadRequest, "Invalid or expired OAuth state")
		}
		_ = s.cache.Del(ctx, cache.BuildKey("hardcover_oauth", "state", state))

		config, err := s.hardcoverConfig(ctx)
		if err != nil {
			return err
		}
		token, err := s.exchangeCode(ctx, config, code)
		if err != nil {
			return err
		}
		_, err = s.trackerRepo.UpsertUserTracker(ctx, userID, "hardcover", token.accessToken, token.refreshToken, token.expiresAt)
		return err
	}
	return apperrors.New(apperrors.ErrInternalError, "OAuth state store unavailable")
}

type hardcoverTokenExchange struct {
	accessToken  string
	refreshToken *string
	expiresAt    *time.Time
}

func (s *scrobbleService) exchangeCode(ctx context.Context, config *models.HardcoverConfig, code string) (*hardcoverTokenExchange, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", config.ClientID)
	form.Set("client_secret", config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", hardcoverTokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Hardcover token exchange failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Hardcover token exchange failed with status %d", resp.StatusCode))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := jsonx.Unmarshal(body, &parsed); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Invalid Hardcover token response")
	}
	if parsed.AccessToken == "" {
		return nil, apperrors.New(apperrors.ErrInternalError, "Hardcover token response missing access_token")
	}

	result := &hardcoverTokenExchange{accessToken: parsed.AccessToken}
	if parsed.RefreshToken != "" {
		result.refreshToken = &parsed.RefreshToken
	}
	if parsed.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
		result.expiresAt = &expiresAt
	}
	return result, nil
}

func (s *scrobbleService) SyncHardcoverProgress(ctx context.Context, userID string, bookID string, progress int, claims *response.JWTClaims) error {
	tracker, err := s.trackerRepo.GetUserTracker(ctx, userID, "hardcover")
	if err != nil || tracker == nil || tracker.AccessToken == "" {
		return apperrors.New(apperrors.ErrBadRequest, "Hardcover integration not connected")
	}

	config, err := s.hardcoverConfig(ctx)
	if err != nil {
		return err
	}

	accessToken := tracker.AccessToken
	if tracker.ExpiresAt != nil && time.Now().After(*tracker.ExpiresAt) && tracker.RefreshToken != nil {
		refreshed, refreshErr := s.refreshAccessToken(ctx, config, *tracker.RefreshToken)
		if refreshErr != nil {
			return refreshErr
		}
		accessToken = refreshed.accessToken
		refreshToken := tracker.RefreshToken
		if refreshed.refreshToken != nil {
			refreshToken = refreshed.refreshToken
		}
		_, _ = s.trackerRepo.UpsertUserTracker(ctx, userID, "hardcover", accessToken, refreshToken, refreshed.expiresAt)
	}

	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book not found")
	}

	resolved := resolveClaims(claims)
	attrs := map[string]any{"library_id": book.LibraryID}
	if !s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermBookRead, attrs) {
		return apperrors.New(apperrors.ErrForbidden, "You do not have access to this book")
	}

	searchQuery := `query SearchBook($title: String!) {
		books(where: {title: {_ilike: $title}}, limit: 1) {
			id
		}
	}`
	var searchRes struct {
		Data struct {
			Books []struct {
				ID string `json:"id"`
			} `json:"books"`
		} `json:"data"`
	}
	if err := s.doGraphql(ctx, accessToken, searchQuery, map[string]any{"title": book.Title}, &searchRes); err != nil {
		return err
	}
	if len(searchRes.Data.Books) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "Book not found on Hardcover")
	}

	mutation := `mutation UpdateReadingState($bookId: Int!, $progress: Int!) {
		updateReadingState(bookId: $bookId, progress: $progress) {
			id
			progress
		}
	}`
	var bookIDInt int64
	_, _ = fmt.Sscanf(searchRes.Data.Books[0].ID, "%d", &bookIDInt)
	if bookIDInt == 0 {
		return apperrors.New(apperrors.ErrInternalError, "Invalid Hardcover book ID")
	}
	var res map[string]any
	if err := s.doGraphql(ctx, accessToken, mutation, map[string]any{"bookId": bookIDInt, "progress": progress}, &res); err != nil {
		return err
	}
	return nil
}

func (s *scrobbleService) refreshAccessToken(ctx context.Context, config *models.HardcoverConfig, refreshToken string) (*hardcoverTokenExchange, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", config.ClientID)
	form.Set("client_secret", config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", hardcoverTokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Hardcover token refresh failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, apperrors.New(apperrors.ErrInternalError, "Hardcover token refresh failed")
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := jsonx.Unmarshal(body, &parsed); err != nil || parsed.AccessToken == "" {
		return nil, apperrors.New(apperrors.ErrInternalError, "Invalid Hardcover refresh response")
	}

	result := &hardcoverTokenExchange{accessToken: parsed.AccessToken}
	if parsed.RefreshToken != "" {
		result.refreshToken = &parsed.RefreshToken
	}
	if parsed.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
		result.expiresAt = &expiresAt
	}
	return result, nil
}

func (s *scrobbleService) doGraphql(ctx context.Context, accessToken string, query string, variables map[string]any, out any) error {
	payload := map[string]any{"query": query, "variables": variables}
	bodyBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hardcoverGraphqlURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Hardcover GraphQL call failed: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Hardcover GraphQL returned status %d", resp.StatusCode))
	}

	if strings.Contains(string(respBody), `"errors"`) && !strings.Contains(string(respBody), `"data"`) {
		return apperrors.New(apperrors.ErrInternalError, "Hardcover GraphQL returned an error")
	}
	return jsonx.Unmarshal(respBody, out)
}
