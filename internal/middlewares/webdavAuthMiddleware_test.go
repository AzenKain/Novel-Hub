package middlewares

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"testing"

	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

type stubWebDAVAuthService struct {
	services.AuthService
	validEmail string
	validPass  string
	claims     *response.JWTClaims
}

func (s *stubWebDAVAuthService) ValidateCredentials(_ context.Context, dto *request.SignInDto) (*response.JWTClaims, error) {
	if dto.Email == s.validEmail && dto.Password == s.validPass {
		return s.claims, nil
	}
	return nil, errors.New("invalid credentials")
}

type stubWebDAVSettingsService struct {
	services.SettingsService
	guestRequired bool
}

func (s *stubWebDAVSettingsService) Public(_ context.Context) (*models.PublicSettings, error) {
	return &models.PublicSettings{
		GuestLoginRequired: s.guestRequired,
	}, nil
}

func TestWebDAVAuthMiddleware(t *testing.T) {
	claims := &response.JWTClaims{
		UId:   "user-1",
		Roles: []constants.RoleType{constants.RoleTypeUser},
	}

	authService := &stubWebDAVAuthService{
		validEmail: "reader@novelhub.local",
		validPass:  "secret123",
		claims:     claims,
	}

	settingsService := &stubWebDAVSettingsService{
		guestRequired: true,
	}

	app := fiber.New()
	app.Use(WebDAVAuth(authService, settingsService))
	app.Get("/webdav", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 1. Test Missing Auth with Guest Required -> 401
	req1 := httptest.NewRequest("GET", "/webdav", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	if resp1.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp1.StatusCode)
	}
	if resp1.Header.Get("WWW-Authenticate") != `Basic realm="NovelHub WebDAV"` {
		t.Fatalf("expected WWW-Authenticate header, got %q", resp1.Header.Get("WWW-Authenticate"))
	}

	// 2. Test Invalid Basic Auth -> 401
	req2 := httptest.NewRequest("GET", "/webdav", nil)
	req2.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("bad:pass")))
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	if resp2.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp2.StatusCode)
	}

	// 3. Test Valid Basic Auth -> 200 OK
	req3 := httptest.NewRequest("GET", "/webdav", nil)
	req3.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("reader@novelhub.local:secret123")))
	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("request 3 failed: %v", err)
	}
	if resp3.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp3.StatusCode)
	}
}

type stubWebDAVUserRepo struct {
	repositories.UserRepository
	user *models.UserEntity
}

func (r *stubWebDAVUserRepo) GetByID(_ context.Context, id string) (*models.UserEntity, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, errors.New("user not found")
}

func (r *stubWebDAVUserRepo) GetTokenVersion(_ context.Context, id string) (int32, error) {
	if r.user != nil && r.user.ID == id {
		return r.user.TokenVersion, nil
	}
	return 0, errors.New("user not found")
}

func TestWebDAVAuthMiddleware_TokenAuth(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-webdav-secret-32bytes-length!")

	uid := "01920000-0000-7000-8000-000000000001"
	user := &models.UserEntity{
		ID:           uid,
		TokenVersion: 0,
		Roles: []*models.RoleSimple{
			{ID: "role-1", Name: "USER"},
		},
	}
	userRepo := &stubWebDAVUserRepo{user: user}

	claims := &response.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "novelhub",
			Subject:   uid,
			Audience:  jwt.ClaimStrings{"novelhub-access"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UId:          uid,
		Roles:        []constants.RoleType{constants.RoleTypeUser},
		TokenType:    "access",
		TokenVersion: 0,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-webdav-secret-32bytes-length!"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	authService := &stubWebDAVAuthService{}
	settingsService := &stubWebDAVSettingsService{guestRequired: true}

	app := fiber.New()
	app.Use(WebDAVAuth(authService, settingsService, userRepo))
	app.Get("/webdav", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 1. Bearer Token in Authorization header -> 200 OK
	reqBearer := httptest.NewRequest("GET", "/webdav", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+tokenString)
	respBearer, err := app.Test(reqBearer)
	if err != nil {
		t.Fatalf("bearer request failed: %v", err)
	}
	if respBearer.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK for Bearer token, got %d", respBearer.StatusCode)
	}

	// 2. Query Token (?token=...) -> 200 OK
	reqQuery := httptest.NewRequest("GET", "/webdav?token="+tokenString, nil)
	respQuery, err := app.Test(reqQuery)
	if err != nil {
		t.Fatalf("query token request failed: %v", err)
	}
	if respQuery.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK for query token, got %d", respQuery.StatusCode)
	}

	// 3. Invalid token -> 401 Unauthorized
	reqInvalid := httptest.NewRequest("GET", "/webdav?token=invalid.jwt.token", nil)
	respInvalid, err := app.Test(reqInvalid)
	if err != nil {
		t.Fatalf("invalid token request failed: %v", err)
	}
	if respInvalid.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", respInvalid.StatusCode)
	}
}

