package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/response"
)

// No credential is a guest; a bad credential must be 401 or the FE never refreshes.
func TestOptionalAuthDistinguishesNoTokenFromABadToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'optional-auth@example.com', 'Reader', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'
	`, userID); err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin",
		bytes.NewReader([]byte(`{"email":"optional-auth@example.com","password":"Password123!"}`)))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin failed: %v", err)
	}
	var auth struct {
		Data response.AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(signinResp.Body).Decode(&auth); err != nil {
		t.Fatalf("decode signin: %v", err)
	}
	if auth.Data.AccessToken == "" {
		t.Fatal("signin returned no access token")
	}

	status := func(header string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("books request failed: %v", err)
		}
		return resp.StatusCode
	}

	if code := status(""); code != http.StatusOK {
		t.Errorf("no credential: got %d, want 200 — guest browsing is supposed to work", code)
	}
	if code := status("Bearer " + auth.Data.AccessToken); code != http.StatusOK {
		t.Errorf("valid token: got %d, want 200", code)
	}
	if code := status("Bearer not.a.jwt"); code != http.StatusUnauthorized {
		t.Errorf("garbage token: got %d, want 401 — the client is told nothing is wrong and never refreshes", code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+auth.Data.AccessToken)
	logoutResp, err := app.Test(logoutReq)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("logout: got %d, want 200", logoutResp.StatusCode)
	}

	if code := status("Bearer " + auth.Data.AccessToken); code != http.StatusUnauthorized {
		t.Errorf("revoked token: got %d, want 401", code)
	}
}

// The cookie is the path the browser actually uses.
func TestOptionalAuthRejectsABadCookieToo(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "stale.garbage.token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("books request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expired access_token cookie: got %d, want 401", resp.StatusCode)
	}
}
