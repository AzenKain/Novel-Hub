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

type stubWebDAVPermCache struct {
	services.PermissionCache
	allowGuest bool
}

func (p *stubWebDAVPermCache) CanRoles(_ []string, roles []constants.RoleType, permission string, _ map[string]any) bool {
	for _, r := range roles {
		if r == constants.RoleTypeGuest && permission == constants.PermWebDAVRead {
			return p.allowGuest
		}
	}
	return false
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

	permCache := &stubWebDAVPermCache{allowGuest: false}

	app := fiber.New()
	app.Use(WebDAVAuth(authService, settingsService, permCache))
	app.Get("/webdav", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Add([]string{"OPTIONS"}, "/webdav", func(c fiber.Ctx) error {
		return c.SendString("OPTIONS OK")
	})

	// 1. Unauthenticated request when guest has no perm -> 401 with WWW-Authenticate
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
	if resp1.Header.Get("DAV") != "1, 2" {
		t.Fatalf("expected DAV header, got %q", resp1.Header.Get("DAV"))
	}

	// 2. Bad credentials -> 401
	req2 := httptest.NewRequest("GET", "/webdav", nil)
	req2.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("bad:pass")))
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	if resp2.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp2.StatusCode)
	}

	// 3. Valid credentials -> 200 OK
	req3 := httptest.NewRequest("GET", "/webdav", nil)
	req3.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("reader@novelhub.local:secret123")))
	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("request 3 failed: %v", err)
	}
	if resp3.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp3.StatusCode)
	}

	// 4. OPTIONS request -> should pass through unauthenticated (200 OK)
	reqOpt := httptest.NewRequest("OPTIONS", "/webdav", nil)
	respOpt, err := app.Test(reqOpt)
	if err != nil {
		t.Fatalf("options request failed: %v", err)
	}
	if respOpt.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK for OPTIONS, got %d", respOpt.StatusCode)
	}

	// 5. Unauthenticated request when guestLoginRequired is false but Guest does NOT have webdav:read -> 401
	settingsService.guestRequired = false
	permCache.allowGuest = false
	req5 := httptest.NewRequest("GET", "/webdav", nil)
	resp5, err := app.Test(req5)
	if err != nil {
		t.Fatalf("request 5 failed: %v", err)
	}
	if resp5.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 challenge when Guest lacks webdav:read, got %d", resp5.StatusCode)
	}

	// 6. Unauthenticated request when guestLoginRequired is false and Guest DOES have webdav:read -> 200 OK
	permCache.allowGuest = true
	req6 := httptest.NewRequest("GET", "/webdav", nil)
	resp6, err := app.Test(req6)
	if err != nil {
		t.Fatalf("request 6 failed: %v", err)
	}
	if resp6.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK when Guest has webdav:read, got %d", resp6.StatusCode)
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
	app.Use(WebDAVAuth(authService, settingsService, nil, userRepo))
	app.Get("/webdav", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	reqBearer := httptest.NewRequest("GET", "/webdav", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+tokenString)
	respBearer, err := app.Test(reqBearer)
	if err != nil {
		t.Fatalf("bearer request failed: %v", err)
	}
	if respBearer.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK for Bearer token, got %d", respBearer.StatusCode)
	}

	reqQuery := httptest.NewRequest("GET", "/webdav?token="+tokenString, nil)
	respQuery, err := app.Test(reqQuery)
	if err != nil {
		t.Fatalf("query token request failed: %v", err)
	}
	if respQuery.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK for query token, got %d", respQuery.StatusCode)
	}

	reqInvalid := httptest.NewRequest("GET", "/webdav?token=invalid.jwt.token", nil)
	respInvalid, err := app.Test(reqInvalid)
	if err != nil {
		t.Fatalf("invalid token request failed: %v", err)
	}
	if respInvalid.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", respInvalid.StatusCode)
	}
}
