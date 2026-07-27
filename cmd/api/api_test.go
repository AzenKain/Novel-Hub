package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func setupTestAppWithDB(t *testing.T) (*fiber.App, *sql.DB, error) {
	t.Helper()
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "novelhub-test.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		return nil, nil, err
	}

	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		return nil, nil, err
	}

	ramCache := cache.NewRamCache()

	server := NewHTTPServer()
	server.SetupServer(db, ramCache)

	return server.App, db, nil
}

func TestHealthCheck(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var data response.CommonResponse
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if !data.Status {
		t.Errorf("expected status true, got false")
	}
}

func TestListBooksEmpty(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %v: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var data response.CommonResponse
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if !data.Status {
		t.Errorf("expected status true, got false")
	}
}

func TestRefreshPrefersRefreshCookieOverAuthorizationHeader(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	seedEmail := "refresh-test@example.com"
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, ?, 'Admin', ?, 'LOCAL', 1)
	`, userID, seedEmail, string(hash)); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT ?, id FROM roles WHERE name IN ('USER', 'ADMIN')
	`, userID); err != nil {
		t.Fatalf("failed to seed roles: %v", err)
	}

	signinBody := []byte(`{"email":"refresh-test@example.com","password":"password123"}`)
	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(signinBody))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin request failed: %v", err)
	}
	if signinResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(signinResp.Body)
		t.Fatalf("expected signin status 200, got %v: %s", signinResp.StatusCode, string(body))
	}

	var signinData struct {
		Status bool                  `json:"status"`
		Data   response.AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(signinResp.Body).Decode(&signinData); err != nil {
		t.Fatalf("failed to decode signin response: %v", err)
	}
	if !signinData.Status || signinData.Data.AccessToken == "" {
		t.Fatalf("signin did not return an access token")
	}

	var refreshCookie *http.Cookie
	for _, cookie := range signinResp.Cookies() {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
			break
		}
	}
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatalf("signin did not set refresh cookie")
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.Header.Set("Authorization", "Bearer "+signinData.Data.AccessToken)
	refreshReq.AddCookie(refreshCookie)
	refreshResp, err := app.Test(refreshReq)
	if err != nil {
		t.Fatalf("refresh request failed: %v", err)
	}
	if refreshResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(refreshResp.Body)
		t.Fatalf("expected refresh status 200, got %v: %s", refreshResp.StatusCode, string(body))
	}
}
