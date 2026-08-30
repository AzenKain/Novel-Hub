package middlewares

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
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
