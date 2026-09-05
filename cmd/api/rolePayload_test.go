package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// The frontend evaluates permissions locally: hasPermission() walks role.permissions and sorts by role.position (web/src/utils/permission.ts).
func TestCurrentUserCarriesRoleGrants(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		role           string
		email          string
		wantAdmin      bool
		wantPermission string
	}{
		{role: "ADMIN", email: "admin@example.com", wantAdmin: true, wantPermission: "setting.manage"},
		{role: "USER", email: "user@example.com", wantAdmin: false, wantPermission: "book.read"},
	} {
		t.Run(tc.role, func(t *testing.T) {
			userID := uuid.Must(uuid.NewV7()).String()
			if _, err := db.Exec(`
				INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
				VALUES (?, ?, 'Test', ?, 'LOCAL', 1)`, userID, tc.email, string(hash)); err != nil {
				t.Fatalf("seed user: %v", err)
			}
			if _, err := db.Exec(`
				INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = ?`,
				userID, tc.role); err != nil {
				t.Fatalf("seed role: %v", err)
			}

			signinBody := []byte(`{"email":"` + tc.email + `","password":"password123"}`)
			signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(signinBody))
			signinReq.Header.Set("Content-Type", "application/json")
			signinResp, err := app.Test(signinReq)
			if err != nil {
				t.Fatalf("signin: %v", err)
			}
			var signin struct {
				Data struct {
					AccessToken string `json:"access_token"`
				} `json:"data"`
			}
			if err := json.NewDecoder(signinResp.Body).Decode(&signin); err != nil {
				t.Fatalf("decode signin: %v", err)
			}
			if signin.Data.AccessToken == "" {
				t.Fatal("signin returned no token")
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
			req.Header.Set("Authorization", "Bearer "+signin.Data.AccessToken)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("get current: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d: %s", resp.StatusCode, body)
			}

			var payload struct {
				Data struct {
					Roles []struct {
						Name        string `json:"name"`
						IsAdmin     bool   `json:"is_admin"`
						Permissions []struct {
							PermissionKey string `json:"permission_key"`
							Effect        string `json:"effect"`
						} `json:"permissions"`
					} `json:"roles"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode: %v (%s)", err, body)
			}
			if len(payload.Data.Roles) != 1 {
				t.Fatalf("roles = %d, want 1: %s", len(payload.Data.Roles), body)
			}
			role := payload.Data.Roles[0]
			if role.Name != tc.role {
				t.Errorf("name = %q, want %q", role.Name, tc.role)
			}
			if role.IsAdmin != tc.wantAdmin {
				t.Errorf("is_admin = %v, want %v", role.IsAdmin, tc.wantAdmin)
			}
			if len(role.Permissions) == 0 {
				t.Fatalf("role %s carries no permissions, so hasPermission() cannot succeed: %s", tc.role, body)
			}
			var found bool
			for _, p := range role.Permissions {
				if p.PermissionKey == tc.wantPermission && p.Effect == "allow" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("role %s is missing an allow for %q", tc.role, tc.wantPermission)
			}
		})
	}
}
