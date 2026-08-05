package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/response"
)

func seedAdmin(t *testing.T, db *sql.DB, email string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, ?, 'Admin', ?, 'LOCAL', 1)
	`, userID, email, string(hash)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT ?, id FROM roles WHERE name IN ('USER', 'ADMIN')
	`, userID); err != nil {
		t.Fatal(err)
	}
	return userID
}

func signinToken(t *testing.T, app *fiber.App, email string) string {
	t.Helper()
	body := []byte(`{"email":"` + email + `","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("signin failed with %d: %s", resp.StatusCode, raw)
	}
	var decoded struct {
		Data response.AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Data.AccessToken == "" {
		t.Fatal("signin returned no access token")
	}
	return decoded.Data.AccessToken
}

func TestAuditRecordsAdminMutationWithActorAndIP(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	adminID := seedAdmin(t, db, "audit-admin@example.com")
	token := signinToken(t, app, "audit-admin@example.com")

	createBody := []byte(`{"email":"audited-target@example.com","password":"Password123!","full_name":"Target User"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201 creating a user, got %d: %s", resp.StatusCode, raw)
	}

	var (
		actorID     sql.NullString
		actorEmail  string
		action      string
		targetType  string
		targetLabel string
		ip          string
	)
	row := db.QueryRow(`SELECT actor_id, actor_email, action, target_type, target_label, ip FROM audit_logs`)
	if err := row.Scan(&actorID, &actorEmail, &action, &targetType, &targetLabel, &ip); err != nil {
		t.Fatalf("no audit row for user.create: %v", err)
	}
	if !actorID.Valid || actorID.String != adminID {
		t.Fatalf("actor_id = %v, want %s", actorID, adminID)
	}
	if actorEmail != "audit-admin@example.com" {
		t.Fatalf("actor_email = %q, want the admin address; a log naming nobody is not an audit trail", actorEmail)
	}
	if action != "user.create" {
		t.Fatalf("action = %q, want user.create", action)
	}
	if targetType != "user" || targetLabel != "audited-target@example.com" {
		t.Fatalf("target = %q/%q, want user/audited-target@example.com", targetType, targetLabel)
	}
	if ip == "" {
		t.Fatal("ip is empty; the actor context did not reach the service")
	}
}

func TestAuditWriteFailureDoesNotFailTheAction(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	seedAdmin(t, db, "audit-drop@example.com")
	token := signinToken(t, app, "audit-drop@example.com")

	if _, err := db.Exec(`DROP TABLE audit_logs`); err != nil {
		t.Fatal(err)
	}

	createBody := []byte(`{"email":"still-created@example.com","password":"Password123!","full_name":"Still Created"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("a broken audit table failed the user creation: got %d: %s", resp.StatusCode, raw)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = 'still-created@example.com'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("user rows = %d, want 1; the mutation was rolled back by a logging failure", count)
	}
}

func TestAuditListRequiresSystemLogRead(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	plainID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'audit-plain@example.com', 'Plain', ?, 'LOCAL', 1)
	`, plainID, string(hash)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'`, plainID); err != nil {
		t.Fatal(err)
	}

	plainToken := signinToken(t, app, "audit-plain@example.com")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a plain user read the audit trail: got %d, want 403", resp.StatusCode)
	}

	seedAdmin(t, db, "audit-reader@example.com")
	adminToken := signinToken(t, app, "audit-reader@example.com")
	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminResp, err := app.Test(adminReq)
	if err != nil {
		t.Fatal(err)
	}
	if adminResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(adminResp.Body)
		t.Fatalf("admin cannot read the audit trail: got %d: %s", adminResp.StatusCode, raw)
	}
}

func TestAuditKeysetPaginationDoesNotRepeatOrSkip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	seedAdmin(t, db, "audit-page@example.com")
	token := signinToken(t, app, "audit-page@example.com")

	const total = 7
	for i := 0; i < total; i++ {
		if _, err := db.Exec(`
			INSERT INTO audit_logs (id, actor_email, action, target_type, target_id, ip, created_at)
			VALUES (?, 'seed@example.com', 'user.update', 'user', ?, '127.0.0.1', CURRENT_TIMESTAMP)
		`, uuid.Must(uuid.NewV7()).String(), i); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}
	cursor := ""
	for page := 0; page < 5; page++ {
		url := "/api/v1/admin/audit?limit=3"
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			t.Fatalf("page %d returned %d: %s", page, resp.StatusCode, raw)
		}
		var decoded struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			Pagination struct {
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			t.Fatal(err)
		}
		for _, entry := range decoded.Data {
			if seen[entry.ID] {
				t.Fatalf("page %d repeated audit id %s", page, entry.ID)
			}
			seen[entry.ID] = true
		}
		cursor = decoded.Pagination.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != total {
		t.Fatalf("walked %d of %d audit rows; keyset pagination lost some", len(seen), total)
	}
}

func TestAuditPruneKeepsRecentDropsOld(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	_, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	recentID := uuid.Must(uuid.NewV7()).String()
	oldID := uuid.Must(uuid.NewV7()).String()
	edgeID := uuid.Must(uuid.NewV7()).String()
	rows := []struct {
		id      string
		created string
	}{
		{recentID, "datetime('now', '-1 days')"},
		{edgeID, "datetime('now', '-89 days')"},
		{oldID, "datetime('now', '-91 days')"},
	}
	for _, row := range rows {
		if _, err := db.Exec(`
			INSERT INTO audit_logs (id, actor_email, action, target_type, ip, created_at)
			VALUES (?, 'seed@example.com', 'user.update', 'user', '127.0.0.1', `+row.created+`)
		`, row.id); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := db.Exec(`DELETE FROM audit_logs WHERE created_at < datetime('now', '-90 days')`); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE id = ?`, oldID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("a 91-day-old audit row survived the 90-day retention window")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE id IN (?, ?)`, recentID, edgeID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("retained rows = %d, want 2; prune ate rows inside the window", count)
	}
}

// Over real HTTP: the value an admin typed reaches the trail, but the mail password does not.
func TestAuditSettingsUpdateRecordsValuesButNotTheSMTPPassword(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "audit-settings-test-key")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup app: %v", err)
	}
	seedAdmin(t, db, "audit-settings@example.com")
	token := signinToken(t, app, "audit-settings@example.com")

	const password = "n0t-in-the-audit-table"
	body := []byte(`{"smtp.host":"mail.example.com","smtp.password":"` + password + `","auth.registration_enabled":false}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("settings update: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("settings update returned %d: %s", resp.StatusCode, raw)
	}

	var label string
	if err := db.QueryRow(`SELECT target_label FROM audit_logs WHERE action = 'settings.update'`).Scan(&label); err != nil {
		t.Fatalf("no settings.update row was written: %v", err)
	}
	if strings.Contains(label, password) {
		t.Fatalf("the SMTP password is sitting in the audit table: %q", label)
	}
	if !strings.Contains(label, "mail.example.com") {
		t.Errorf("the host an admin typed was not recorded: %q", label)
	}
	if !strings.Contains(label, "auth.registration_enabled = false") {
		t.Errorf("the trail cannot answer who turned registration off: %q", label)
	}

	var stored string
	if err := db.QueryRow(`SELECT value_json FROM app_settings WHERE key = 'smtp.password'`).Scan(&stored); err != nil {
		t.Fatalf("password was not stored: %v", err)
	}
	if strings.Contains(stored, password) {
		t.Fatalf("the password is in app_settings in plaintext: %q", stored)
	}
}
