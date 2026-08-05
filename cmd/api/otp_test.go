package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func postJSON(t *testing.T, app *fiber.App, path string, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s failed: %v", path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// A service-level test cannot prove the route is wired.
func TestRegisterOverHTTPRefusesWithoutOTPWhenRequired(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "otp-http-test-key")
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("SQLITE_DB_PATH", filepath.Join(dataDir, "otp-http.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	smtpPort, received := captureSMTP(t)
	seed := map[string]string{
		"auth.require_email_verify":   "true",
		"auth.password_reset_enabled": "true",
		"smtp.enabled":                "true",
		"smtp.host":                   `"localhost"`,
		"smtp.port":                   strconv.Itoa(smtpPort),
		"smtp.from_email":             `"library@example.com"`,
		"smtp.tls_mode":               `"none"`,
		"smtp.allow_private_networks": "true",
		// Spends more than the default 5 auth requests; the limiter has its own test.
		"limits.rate_limit_auth": "50",
	}
	for key, value := range seed {
		if _, err := db.Exec(`
			INSERT INTO app_settings (key, value_json) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
		`, key, value); err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())
	app := server.App

	status, body := postJSON(t, app, "/api/v1/auth/register",
		`{"email":"newcomer@example.com","password":"Sup3rSecret!"}`)
	if status == http.StatusCreated {
		t.Fatalf("registration succeeded without email verification: %s", body)
	}

	status, body = postJSON(t, app, "/api/v1/auth/otp/request",
		`{"email":"newcomer@example.com","purpose":"email_verify"}`)
	if status != http.StatusOK {
		t.Fatalf("otp/request returned %d: %s", status, body)
	}

	var message string
	select {
	case message = <-received:
	default:
		t.Fatal("no code was mailed")
	}
	code := extractCode(t, message)

	status, body = postJSON(t, app, "/api/v1/auth/otp/verify",
		`{"email":"newcomer@example.com","purpose":"email_verify","code":"`+code+`"}`)
	if status != http.StatusOK {
		t.Fatalf("otp/verify returned %d: %s", status, body)
	}
	var verified struct {
		Data struct {
			OTPTicket string `json:"otp_ticket"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &verified); err != nil || verified.Data.OTPTicket == "" {
		t.Fatalf("otp/verify gave no ticket: %v (%s)", err, body)
	}

	status, body = postJSON(t, app, "/api/v1/auth/register",
		`{"email":"newcomer@example.com","password":"Sup3rSecret!","otp_ticket":"`+verified.Data.OTPTicket+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("verified registration returned %d: %s", status, body)
	}

	status, body = postJSON(t, app, "/api/v1/auth/register",
		`{"email":"another@example.com","password":"Sup3rSecret!","otp_ticket":"`+verified.Data.OTPTicket+`"}`)
	if status == http.StatusCreated {
		t.Fatalf("the same ticket registered a second account: %s", body)
	}

	status, body = postJSON(t, app, "/api/v1/auth/signin",
		`{"email":"newcomer@example.com","password":"Sup3rSecret!"}`)
	if status != http.StatusOK {
		t.Fatalf("the verified account cannot sign in: %d %s", status, body)
	}
}

func extractCode(t *testing.T, message string) string {
	t.Helper()
	for _, word := range strings.Fields(message) {
		trimmed := strings.Trim(word, ".,")
		if len(trimmed) != 6 {
			continue
		}
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed
		}
	}
	t.Fatalf("no 6-digit code in the mailed message:\n%s", message)
	return ""
}

func TestOTPEndpointsRejectUnknownPurpose(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "otp-http-test-key")
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatal(err)
	}

	status, body := postJSON(t, app, "/api/v1/auth/otp/request",
		`{"email":"reader@example.com","purpose":"admin_takeover"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("unknown purpose returned %d, want 400: %s", status, body)
	}

	status, body = postJSON(t, app, "/api/v1/auth/otp/verify",
		`{"email":"reader@example.com","purpose":"password_reset","code":"12"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("malformed code returned %d, want 400: %s", status, body)
	}
}

func TestSendUserEmailRequiresPermission(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "user-email-http-key")
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatal(err)
	}

	status, body := postJSON(t, app, "/api/v1/users/01920000-0000-7000-8000-0000000000a1/email",
		`{"subject":"Hi","body":"Hello"}`)
	if status != http.StatusUnauthorized && status != http.StatusForbidden {
		t.Fatalf("anonymous user email returned %d, want 401/403: %s", status, body)
	}
}
