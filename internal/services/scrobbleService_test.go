package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/netx"
)

func newTestScrobbleService(trackerRepo repositories.TrackerRepository, bookRepo repositories.BookDBRepository, config *models.HardcoverConfig, c cache.Cache) ScrobbleService {
	return NewScrobbleService(trackerRepo, bookRepo, &stubScrobbleSettings{config: config}, c, &stubPermissionCache{allow: true})
}

type stubPermissionCache struct {
	PermissionCache
	allow bool
}

func (p *stubPermissionCache) Can(ctx context.Context, userID string, permission string, attrs map[string]any) bool {
	return p.allow
}

func (p *stubPermissionCache) CanRoles(roleIDs []string, roles []constants.RoleType, permission string, attrs map[string]any) bool {
	return p.allow
}

type stubScrobbleSettings struct {
	SettingsService
	config *models.HardcoverConfig
	err    error
}

func (s *stubScrobbleSettings) HardcoverConfig(ctx context.Context) (*models.HardcoverConfig, error) {
	return s.config, s.err
}

type stubScrobbleTrackerRepo struct {
	repositories.TrackerRepository
	tracker *models.UserTrackerEntity
	err     error

	upsertedUserID     string
	upsertedProvider   string
	upsertedAccess     string
	upsertedRefresh    *string
	upsertedExpiresAt  *time.Time
}

func (s *stubScrobbleTrackerRepo) GetUserTracker(ctx context.Context, _ string, _ string) (*models.UserTrackerEntity, error) {
	return s.tracker, s.err
}

func (s *stubScrobbleTrackerRepo) UpsertUserTracker(ctx context.Context, userID string, provider string, accessToken string, refreshToken *string, expiresAt *time.Time) (*models.UserTrackerEntity, error) {
	s.upsertedUserID = userID
	s.upsertedProvider = provider
	s.upsertedAccess = accessToken
	s.upsertedRefresh = refreshToken
	s.upsertedExpiresAt = expiresAt
	return &models.UserTrackerEntity{ID: "trk-1", UserID: userID, Provider: provider, AccessToken: accessToken}, nil
}

type stubScrobbleBookRepo struct {
	repositories.BookDBRepository
	book *models.BookEntity
}

func (s *stubScrobbleBookRepo) GetBook(ctx context.Context, _ string) (*models.BookEntity, error) {
	return s.book, nil
}

func testHardcoverConfig() *models.HardcoverConfig {
	return &models.HardcoverConfig{Enabled: true, ClientID: "client-1", ClientSecret: "secret-1"}
}

func TestGetHardcoverAuthorizeURL(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	c := cache.NewRamCache()
	s := newTestScrobbleService(&stubScrobbleTrackerRepo{}, &stubScrobbleBookRepo{}, testHardcoverConfig(), c)

	url, err := s.GetHardcoverAuthorizeURL(context.Background(), "user-1", "http://localhost:8080/api/v1/scrobble/hardcover/callback")
	if err != nil {
		t.Fatalf("GetHardcoverAuthorizeURL: %v", err)
	}
	if !strings.HasPrefix(url, "https://hardcover.app/oauth/authorize?") {
		t.Fatalf("unexpected authorize URL: %s", url)
	}
	for _, want := range []string{"client_id=client-1", "redirect_uri=http%3A%2F%2Flocalhost", "scope=read%2Cwrite", "state="} {
		if !strings.Contains(url, want) {
			t.Errorf("authorize URL missing %q: %s", want, url)
		}
	}

	state := strings.SplitN(url, "state=", 2)[1]
	var userID string
	if err := c.Get(context.Background(), cache.BuildKey("hardcover_oauth", "state", state), &userID); err != nil || userID != "user-1" {
		t.Fatalf("state not stored in cache: err=%v userID=%q", err, userID)
	}
}

func TestGetHardcoverAuthorizeURLDisabled(t *testing.T) {
	s := newTestScrobbleService(&stubScrobbleTrackerRepo{}, &stubScrobbleBookRepo{}, &models.HardcoverConfig{Enabled: false}, cache.NewRamCache())
	_, err := s.GetHardcoverAuthorizeURL(context.Background(), "user-1", "http://localhost/x")
	if err == nil || !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestHandleHardcoverCallback(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "grant_type=authorization_code") {
			t.Errorf("missing grant_type: %s", body)
		}
		if !strings.Contains(string(body), "client_secret=secret-1") {
			t.Errorf("missing client_secret: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-1","refresh_token":"ref-1","expires_in":3600}`))
	}))
	defer server.Close()
	hardcoverTokenURL = server.URL + "/token"

	c := cache.NewRamCache()
	state := "state-abc"
	_ = c.Set(context.Background(), cache.BuildKey("hardcover_oauth", "state", state), "user-1", 10*time.Minute)

	trackerRepo := &stubScrobbleTrackerRepo{}
	s := newTestScrobbleService(trackerRepo, &stubScrobbleBookRepo{}, testHardcoverConfig(), c)

	if err := s.HandleHardcoverCallback(context.Background(), "code-1", state); err != nil {
		t.Fatalf("HandleHardcoverCallback: %v", err)
	}
	if trackerRepo.upsertedAccess != "tok-1" {
		t.Errorf("upserted access token = %q, want tok-1", trackerRepo.upsertedAccess)
	}
	if trackerRepo.upsertedRefresh == nil || *trackerRepo.upsertedRefresh != "ref-1" {
		t.Errorf("upserted refresh token = %v, want ref-1", trackerRepo.upsertedRefresh)
	}
	if trackerRepo.upsertedExpiresAt == nil || !trackerRepo.upsertedExpiresAt.After(time.Now().Add(30*time.Minute)) {
		t.Errorf("upserted expires_at = %v, want ~1h in future", trackerRepo.upsertedExpiresAt)
	}

	// state must be consumed
	var userID string
	_ = c.Get(context.Background(), cache.BuildKey("hardcover_oauth", "state", state), &userID)
	if userID != "" {
		t.Errorf("state not deleted after callback")
	}
}

func TestHandleHardcoverCallbackBadState(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	s := newTestScrobbleService(&stubScrobbleTrackerRepo{}, &stubScrobbleBookRepo{}, testHardcoverConfig(), cache.NewRamCache())
	err := s.HandleHardcoverCallback(context.Background(), "code-1", "unknown-state")
	if err == nil || !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest for unknown state, got %v", err)
	}
}

func TestSyncHardcoverProgress(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "SearchBook") {
			_, _ = w.Write([]byte(`{"data":{"books":[{"id":"42"}]}}`))
			return
		}
		if strings.Contains(string(body), "updateReadingState") {
			var payload struct {
				Variables map[string]any `json:"variables"`
			}
			_ = json.Unmarshal(body, &payload)
			if payload.Variables["bookId"].(float64) != 42 || payload.Variables["progress"].(float64) != 10 {
				t.Errorf("unexpected mutation variables: %v", payload.Variables)
			}
			_, _ = w.Write([]byte(`{"data":{"updateReadingState":{"id":42,"progress":10}}}`))
			return
		}
		t.Errorf("unexpected GraphQL query: %s", body)
	}))
	defer server.Close()
	hardcoverGraphqlURL = server.URL

	tracker := &models.UserTrackerEntity{
		ID:          "trk-1",
		UserID:      "user-1",
		Provider:    "hardcover",
		AccessToken: "tok-1",
	}
	book := &models.BookEntity{ID: "book-1", Title: "Test Book"}
	s := newTestScrobbleService(&stubScrobbleTrackerRepo{tracker: tracker}, &stubScrobbleBookRepo{book: book}, testHardcoverConfig(), cache.NewRamCache())

	if err := s.SyncHardcoverProgress(context.Background(), "user-1", "book-1", 10, &response.JWTClaims{}); err != nil {
		t.Fatalf("SyncHardcoverProgress: %v", err)
	}
}

func TestSyncHardcoverProgressNotConnected(t *testing.T) {
	s := newTestScrobbleService(&stubScrobbleTrackerRepo{err: apperrors.New(apperrors.ErrNotFound, "none")}, &stubScrobbleBookRepo{book: &models.BookEntity{ID: "book-1"}}, testHardcoverConfig(), cache.NewRamCache())
	err := s.SyncHardcoverProgress(context.Background(), "user-1", "book-1", 10, &response.JWTClaims{})
	if err == nil || !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest when not connected, got %v", err)
	}
}

func TestSyncHardcoverProgressRefreshesExpiredToken(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "grant_type=refresh_token") {
				t.Errorf("expected refresh_token grant, got: %s", body)
			}
			_, _ = w.Write([]byte(`{"access_token":"tok-fresh","refresh_token":"ref-new","expires_in":3600}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "SearchBook") {
			_, _ = w.Write([]byte(`{"data":{"books":[{"id":"42"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"updateReadingState":{"id":42,"progress":10}}}`))
	}))
	defer server.Close()
	hardcoverTokenURL = server.URL + "/token"
	hardcoverGraphqlURL = server.URL

	past := time.Now().Add(-time.Hour)
	refresh := "ref-old"
	tracker := &models.UserTrackerEntity{
		ID:           "trk-1",
		UserID:       "user-1",
		Provider:     "hardcover",
		AccessToken:  "tok-expired",
		RefreshToken: &refresh,
		ExpiresAt:    &past,
	}
	book := &models.BookEntity{ID: "book-1", Title: "Test Book"}

	trackerRepo := &stubScrobbleTrackerRepo{tracker: tracker}
	s := newTestScrobbleService(trackerRepo, &stubScrobbleBookRepo{book: book}, testHardcoverConfig(), cache.NewRamCache())

	if err := s.SyncHardcoverProgress(context.Background(), "user-1", "book-1", 10, &response.JWTClaims{}); err != nil {
		t.Fatalf("SyncHardcoverProgress: %v", err)
	}
	if trackerRepo.upsertedAccess != "tok-fresh" {
		t.Errorf("expected refreshed token persisted, got %q", trackerRepo.upsertedAccess)
	}
	if trackerRepo.upsertedRefresh == nil || *trackerRepo.upsertedRefresh != "ref-new" {
		t.Errorf("expected new refresh token persisted, got %v", trackerRepo.upsertedRefresh)
	}
}

func TestSyncHardcoverProgressForbidden(t *testing.T) {
	tracker := &models.UserTrackerEntity{
		ID:          "trk-1",
		UserID:      "user-1",
		Provider:    "hardcover",
		AccessToken: "tok-1",
	}
	book := &models.BookEntity{ID: "book-1", Title: "Test Book"}

	settings := &stubScrobbleSettings{config: testHardcoverConfig()}
	s := NewScrobbleService(
		&stubScrobbleTrackerRepo{tracker: tracker},
		&stubScrobbleBookRepo{book: book},
		settings,
		cache.NewRamCache(),
		&stubPermissionCache{allow: false},
	)

	err := s.SyncHardcoverProgress(context.Background(), "user-1", "book-1", 10, &response.JWTClaims{})
	if err == nil || !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when permission is denied, got %v", err)
	}
}