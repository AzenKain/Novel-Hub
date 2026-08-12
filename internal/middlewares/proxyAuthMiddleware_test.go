package middlewares_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/middlewares"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/mailer"
	"novelhub/pkg/worker"
)

// MockSettingsService implements services.SettingsService for testing
type MockSettingsService struct {
	settings *models.AdminSettings
}

func (m *MockSettingsService) Reload(ctx context.Context) error { return nil }
func (m *MockSettingsService) Public(ctx context.Context) (*models.PublicSettings, error) {
	return &m.settings.PublicSettings, nil
}
func (m *MockSettingsService) Admin(ctx context.Context) (*models.AdminSettings, error) {
	return m.settings, nil
}
func (m *MockSettingsService) Limits() models.RuntimeLimits { return models.RuntimeLimits{} }
func (m *MockSettingsService) ServerURL() string            { return "" }
func (m *MockSettingsService) UpdateSettings(ctx context.Context, settings map[string]any) (*models.AdminSettings, error) {
	return nil, nil
}
func (m *MockSettingsService) GuestAllows(libraryID string) bool { return true }
func (m *MockSettingsService) SetupRequired(ctx context.Context) bool { return false }
func (m *MockSettingsService) SaveAsset(ctx context.Context, target string, fileData []byte, fileName string, urlStr string) (string, error) {
	return "", nil
}
func (m *MockSettingsService) SMTP(ctx context.Context) (mailer.SMTPConfig, error) {
	return mailer.SMTPConfig{}, nil
}
func (m *MockSettingsService) TestSMTP(ctx context.Context, dto *request.SMTPTestDto) error {
	return nil
}
func (m *MockSettingsService) OAuthProviderConfig(ctx context.Context, provider string) (*models.OAuthProviderConfig, error) {
	return nil, nil
}

// MockAuthService implements services.AuthService for testing
type MockAuthService struct {
	genTokenFunc func(user *models.UserEntity) (*response.AuthResponse, error)
}

func (m *MockAuthService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error) {
	return nil, nil
}
func (m *MockAuthService) SetTOTPService(service services.TOTPService) {}
func (m *MockAuthService) SetJobQueue(jobQueue *worker.Queue)          {}
func (m *MockAuthService) ExecuteSendOTPJob(ctx context.Context, payloadJSON string) error {
	return nil
}
func (m *MockAuthService) ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*response.JWTClaims, error) {
	return nil, nil
}
func (m *MockAuthService) Register(ctx context.Context, dto *request.RegisterDto) (*response.UserResponse, error) {
	return nil, nil
}
func (m *MockAuthService) SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.UserResponse, error) {
	return nil, nil
}
func (m *MockAuthService) RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, error) {
	return nil, nil
}
func (m *MockAuthService) Logout(ctx context.Context, userID string) error { return nil }
func (m *MockAuthService) RequestOTP(ctx context.Context, dto *request.RequestOTPDto) (*response.OTPRequestResponse, error) {
	return nil, nil
}
func (m *MockAuthService) VerifyOTP(ctx context.Context, dto *request.VerifyOTPDto) (*response.OTPVerifyResponse, error) {
	return nil, nil
}
func (m *MockAuthService) ResetPasswordWithOTP(ctx context.Context, dto *request.ResetPasswordWithOTPDto) error {
	return nil
}
func (m *MockAuthService) GenToken(user *models.UserEntity) (*response.AuthResponse, error) {
	if m.genTokenFunc != nil {
		return m.genTokenFunc(user)
	}
	return &response.AuthResponse{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
	}, nil
}
func (m *MockAuthService) SigninOrRegisterOAuth(ctx context.Context, provider string, email string, name string, avatarURL string, oauth2ID string) (*response.AuthResponse, error) {
	return nil, nil
}
func (m *MockAuthService) BuildOAuthURL(ctx context.Context, provider string, redirect string) (authURL string, stateUUID string, err error) {
	return "", "", nil
}
func (m *MockAuthService) HandleOAuthCallback(ctx context.Context, provider string, code string, stateParam string, cookieState string) (*response.OAuthCallbackResponse, error) {
	return nil, nil
}

func TestProxyAuthMiddleware(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh")

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	// Insert a default user role
	if _, err := db.Exec(`INSERT INTO roles (id, name, auto_assign) VALUES ('role-user', 'user', 1)`); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	userRepo := repositories.NewUserRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	txManager := database.NewTxManager(db)

	mockSettings := &models.AdminSettings{
		ProxyAuth: models.ProxyAuthSettings{
			Enabled:        true,
			HeaderNames:    []string{"X-Forwarded-User", "Remote-User"},
			TrustedProxies: []string{"127.0.0.1", "10.0.0.0/24", "0.0.0.0"},
			AutoCreate:     true,
		},
	}
	settingsSvc := &MockSettingsService{settings: mockSettings}
	authSvc := &MockAuthService{}

	app := fiber.New()
	app.Use(middlewares.ProxyAuth(settingsSvc, authSvc, userRepo, roleRepo, txManager))
	app.Get("/test", func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		return c.JSON(fiber.Map{
			"auth_header": authHeader,
		})
	})

	t.Run("Disabled proxy auth should skip", func(t *testing.T) {
		mockSettings.ProxyAuth.Enabled = false
		mockSettings.ProxyAuth.TrustedProxies = []string{"0.0.0.0"}
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-User", "test@example.com")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Untrusted proxy IP should ignore headers", func(t *testing.T) {
		mockSettings.ProxyAuth.Enabled = true
		mockSettings.ProxyAuth.TrustedProxies = []string{"127.0.0.1"} // 0.0.0.0 is not trusted
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-User", "test@example.com")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify user was NOT created in db
		u, err := userRepo.GetByEmail(context.Background(), "test@example.com")
		if err == nil && u != nil {
			t.Fatal("User was created even though proxy IP was untrusted")
		}
	})

	t.Run("Trusted proxy IP but missing header should skip", func(t *testing.T) {
		mockSettings.ProxyAuth.TrustedProxies = []string{"0.0.0.0"}
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Trusted proxy IP and valid header should login and auto-create", func(t *testing.T) {
		mockSettings.ProxyAuth.TrustedProxies = []string{"0.0.0.0"}
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-User", "test-auto@example.com")
		resp, _ := app.Test(req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify user was created in db
		u, err := userRepo.GetByEmail(context.Background(), "test-auto@example.com")
		if err != nil || u == nil {
			t.Fatal("User was not automatically created")
		}
	})
}
