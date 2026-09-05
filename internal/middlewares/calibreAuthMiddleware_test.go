package middlewares

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	jwt "github.com/golang-jwt/jwt/v5"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
)

type mockCalibreAuthService struct {
	services.AuthService
	validEmail    string
	validPassword string
	bannedEmail   string
	claims        *response.JWTClaims
}

func (m *mockCalibreAuthService) ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*response.JWTClaims, error) {
	if m.bannedEmail != "" && dto.Email == m.bannedEmail {
		return nil, apperrors.New(apperrors.ErrForbidden, "User account is banned")
	}
	if dto.Email == m.validEmail && dto.Password == m.validPassword {
		return m.claims, nil
	}
	return nil, errors.New("invalid credentials")
}

type mockCalibreSettingsService struct {
	services.SettingsService
	guestRequired bool
}

func (m *mockCalibreSettingsService) Public(ctx context.Context) (*models.PublicSettings, error) {
	return &models.PublicSettings{GuestLoginRequired: m.guestRequired}, nil
}

func TestCalibreAuthMiddleware_BasicAuth_Success(t *testing.T) {
	authSvc := &mockCalibreAuthService{
		validEmail:    "reader@example.com",
		validPassword: "secretpassword",
		claims:        &response.JWTClaims{UId: "user-123", Roles: []constants.RoleType{constants.RoleTypeUser}},
	}
	settingsSvc := &mockCalibreSettingsService{guestRequired: true}

	app := fiber.New()
	app.Use(CalibreAuth(authSvc, settingsSvc))
	app.Get("/test", func(c fiber.Ctx) error {
		claims := c.Locals("user_claims").(*response.JWTClaims)
		return c.SendString("ok:" + claims.UId)
	})

	encoded := base64.StdEncoding.EncodeToString([]byte("reader@example.com:secretpassword"))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic "+encoded)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestCalibreAuthMiddleware_BasicAuth_Invalid(t *testing.T) {
	authSvc := &mockCalibreAuthService{
		validEmail:    "reader@example.com",
		validPassword: "secretpassword",
	}
	settingsSvc := &mockCalibreSettingsService{guestRequired: true}

	app := fiber.New()
	app.Use(CalibreAuth(authSvc, settingsSvc))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	encoded := base64.StdEncoding.EncodeToString([]byte("reader@example.com:wrongpass"))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic "+encoded)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
	if val := resp.Header.Get("WWW-Authenticate"); val == "" {
		t.Errorf("expected WWW-Authenticate header")
	}
}

func TestCalibreAuthMiddleware_GuestMode_Allowed(t *testing.T) {
	authSvc := &mockCalibreAuthService{}
	settingsSvc := &mockCalibreSettingsService{guestRequired: false}

	app := fiber.New()
	app.Use(CalibreAuth(authSvc, settingsSvc))
	app.Get("/test", func(c fiber.Ctx) error {
		claims := c.Locals("user_claims").(*response.JWTClaims)
		return c.SendString("ok:" + claims.UId)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestCalibreAuthMiddleware_GuestMode_Required_Rejects(t *testing.T) {
	authSvc := &mockCalibreAuthService{}
	settingsSvc := &mockCalibreSettingsService{guestRequired: true}

	app := fiber.New()
	app.Use(CalibreAuth(authSvc, settingsSvc))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCalibreAuthMiddleware_TokenAuth(t *testing.T) {
	secret := "calibre-test-jwt-secret-key-32bytes!"
	t.Setenv("JWT_SECRET", secret)

	validUUID := "018d4512-3456-789a-bcde-f0123456789a"
	claims := &response.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "novelhub",
			Subject:   validUUID,
			Audience:  jwt.ClaimStrings{"novelhub-access"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UId:          validUUID,
		Roles:        []constants.RoleType{constants.RoleTypeUser},
		TokenType:    "access",
		TokenVersion: 0,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	validToken, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	userRepo := &stubWebDAVUserRepo{
		user: &models.UserEntity{ID: validUUID, Email: "user@example.com", TokenVersion: 0},
	}

	authSvc := &mockCalibreAuthService{}
	settingsSvc := &mockCalibreSettingsService{guestRequired: true}

	app := fiber.New()
	app.Use(CalibreAuth(authSvc, settingsSvc, userRepo))
	app.Get("/test", func(c fiber.Ctx) error {
		claims := c.Locals("user_claims").(*response.JWTClaims)
		return c.SendString("ok:" + claims.UId)
	})

	reqBearer := httptest.NewRequest("GET", "/test", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+validToken)
	respBearer, err := app.Test(reqBearer)
	if err != nil || respBearer.StatusCode != 200 {
		t.Fatalf("expected 200 for Bearer token, got status %d, err %v", respBearer.StatusCode, err)
	}

	reqQuery := httptest.NewRequest("GET", "/test?token="+validToken, nil)
	respQuery, err := app.Test(reqQuery)
	if err != nil || respQuery.StatusCode != 200 {
		t.Fatalf("expected 200 for query token, got status %d, err %v", respQuery.StatusCode, err)
	}
}

func TestCalibreAuthMiddleware_BannedUser_Returns403(t *testing.T) {
	authSvc := &mockCalibreAuthService{
		bannedEmail: "banned@example.com",
	}
	settingsSvc := &mockCalibreSettingsService{guestRequired: true}

	app := fiber.New()
	app.Use(CalibreAuth(authSvc, settingsSvc))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	encoded := base64.StdEncoding.EncodeToString([]byte("banned@example.com:somepass"))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic "+encoded)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 Forbidden for banned user, got status %d", resp.StatusCode)
	}
}
