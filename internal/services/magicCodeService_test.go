package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/worker"
)

type mockAuthService struct{}

func (m *mockAuthService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error) {
	return nil, nil
}
func (m *mockAuthService) SetTOTPService(service TOTPService) {}
func (m *mockAuthService) SetJobQueue(jobQueue *worker.Queue) {}
func (m *mockAuthService) ExecuteSendOTPJob(ctx context.Context, payloadJSON string) error {
	return nil
}
func (m *mockAuthService) ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*response.JWTClaims, error) {
	return nil, nil
}
func (m *mockAuthService) Register(ctx context.Context, dto *request.RegisterDto) (*response.UserResponse, error) {
	return nil, nil
}
func (m *mockAuthService) SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.UserResponse, error) {
	return nil, nil
}
func (m *mockAuthService) RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, error) {
	return nil, nil
}
func (m *mockAuthService) Logout(ctx context.Context, userID string) error { return nil }
func (m *mockAuthService) RequestOTP(ctx context.Context, dto *request.RequestOTPDto) (*response.OTPRequestResponse, error) {
	return nil, nil
}
func (m *mockAuthService) VerifyOTP(ctx context.Context, dto *request.VerifyOTPDto) (*response.OTPVerifyResponse, error) {
	return nil, nil
}
func (m *mockAuthService) ResetPasswordWithOTP(ctx context.Context, dto *request.ResetPasswordWithOTPDto) error {
	return nil
}
func (m *mockAuthService) GenToken(user *models.UserEntity) (*response.AuthResponse, error) {
	return &response.AuthResponse{AccessToken: "mock-access-token", RefreshToken: "mock-refresh-token"}, nil
}
func (m *mockAuthService) SigninOrRegisterOAuth(ctx context.Context, provider string, email string, name string, avatarURL string, oauth2ID string) (*response.AuthResponse, error) {
	return nil, nil
}
func (m *mockAuthService) BuildOAuthURL(ctx context.Context, provider string, redirect string) (authURL string, stateUUID string, err error) {
	return "", "", nil
}
func (m *mockAuthService) HandleOAuthCallback(ctx context.Context, provider string, code string, stateParam string, cookieState string) (*response.OAuthCallbackResponse, error) {
	return nil, nil
}

func newTestMagicCodeService(t *testing.T) (MagicCodeService, repositories.UserRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "magic_code_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	userRepo := repositories.NewUserRepository(db, ramCache)
	magicRepo := repositories.NewMagicCodeRepository(db, ramCache)

	svc := NewMagicCodeService(magicRepo, userRepo, &mockAuthService{})
	return svc, userRepo, db
}

func TestMagicCodeFullLifecycle(t *testing.T) {
	svc, _, db := newTestMagicCodeService(t)
	ctx := context.Background()
	const baseURL = "http://localhost:3434"

	if _, err := db.Exec(`INSERT INTO users (id, email, full_name, password_hash) VALUES ('user-1', 'test@example.com', 'Test User', 'hash')`); err != nil {
		t.Fatal(err)
	}

	reqResp, err := svc.RequestMagicCode(ctx, &request.RequestMagicCodeDto{DeviceInfo: "Kindle Oasis"}, baseURL)
	if err != nil {
		t.Fatalf("RequestMagicCode failed: %v", err)
	}

	if len(reqResp.Code) != 7 || reqResp.Code[3] != '-' {
		t.Fatalf("Expected 6-digit code with hyphen (XXX-XXX), got: %s", reqResp.Code)
	}
	if reqResp.PollToken == "" {
		t.Fatal("Poll token should not be empty")
	}
	if reqResp.ExpiresInSeconds != 300 {
		t.Fatalf("ExpiresInSeconds = %d, want 300", reqResp.ExpiresInSeconds)
	}

	pollPending, err := svc.PollMagicCode(ctx, &request.PollMagicCodeDto{PollToken: reqResp.PollToken})
	if err != nil {
		t.Fatalf("PollMagicCode failed: %v", err)
	}
	if pollPending.Status != "pending" || pollPending.AuthResponse != nil {
		t.Fatalf("Expected pending status with no auth response, got status: %s", pollPending.Status)
	}

	authResp, err := svc.ActivateMagicCode(ctx, "user-1", &request.ActivateMagicCodeDto{Code: reqResp.Code})
	if err != nil {
		t.Fatalf("ActivateMagicCode failed: %v", err)
	}
	if authResp.AccessToken == "" {
		t.Fatal("Expected non-empty access token upon activation")
	}

	pollActive, err := svc.PollMagicCode(ctx, &request.PollMagicCodeDto{PollToken: reqResp.PollToken})
	if err != nil {
		t.Fatalf("PollMagicCode after activation failed: %v", err)
	}
	if pollActive.Status != "active" || pollActive.AuthResponse == nil || pollActive.AuthResponse.AccessToken == "" {
		t.Fatalf("Expected active status with access token, got: %+v", pollActive)
	}

	_, errSecondActivate := svc.ActivateMagicCode(ctx, "user-1", &request.ActivateMagicCodeDto{Code: reqResp.Code})
	if errSecondActivate == nil {
		t.Fatal("Second activation of the same code should fail")
	}
}

func TestActivateMagicCodeInvalidCode(t *testing.T) {
	svc, _, _ := newTestMagicCodeService(t)
	ctx := context.Background()

	_, err := svc.ActivateMagicCode(ctx, "user-1", &request.ActivateMagicCodeDto{Code: "000-000"})
	if err == nil {
		t.Fatal("Activating nonexistent magic code should return error")
	}
}

func TestPollMagicCodeInvalidToken(t *testing.T) {
	svc, _, _ := newTestMagicCodeService(t)
	ctx := context.Background()

	pollResp, err := svc.PollMagicCode(ctx, &request.PollMagicCodeDto{PollToken: "invalid-token"})
	if err != nil {
		t.Fatalf("PollMagicCode should not error on unknown token, got: %v", err)
	}
	if pollResp.Status != "expired" {
		t.Fatalf("Expected status 'expired' for unknown token, got: %s", pollResp.Status)
	}
}
