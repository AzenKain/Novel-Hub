package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/totp"
)

func setupAppTOTP(t *testing.T) (*fiber.App, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	t.Setenv("DB_ENCRYPTION_KEY", "totp-test-key")
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("SQLITE_DB_PATH", filepath.Join(dataDir, "totp.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db, "../../db/schema"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value_json) VALUES ('limits.rate_limit_auth', '200')
		ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
	`); err != nil {
		t.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("Sup3rSecret!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'reader@example.com', 'Reader', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'`, userID); err != nil {
		t.Fatal(err)
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())
	return server.App, userID
}

func signinTOTP(t *testing.T, app *fiber.App, body string) (int, string) {
	t.Helper()
	return postJSON(t, app, "/api/v1/auth/signin", body)
}

func accessTokenFrom(t *testing.T, body string) string {
	t.Helper()
	var decoded struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			TOTPRequired bool   `json:"totp_required"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode signin: %v (%s)", err, body)
	}
	return decoded.Data.AccessToken
}

func authedPost(t *testing.T, app *fiber.App, path, token, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s failed: %v", path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// Enrol, confirm, then sign in with a generated code — the whole loop over real HTTP.
func TestTOTPEnrollThenSigninRequiresACode(t *testing.T) {
	app, _ := setupAppTOTP(t)

	_, body := signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	token := accessTokenFrom(t, body)
	if token == "" {
		t.Fatalf("plain signin gave no token before enrolment: %s", body)
	}

	status, body := authedPost(t, app, "/api/v1/auth/totp/enroll", token, `{}`)
	if status != http.StatusOK {
		t.Fatalf("enroll returned %d: %s", status, body)
	}
	var enroll struct {
		Data struct {
			Secret          string `json:"secret"`
			ProvisioningURI string `json:"provisioning_uri"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &enroll); err != nil {
		t.Fatal(err)
	}
	if enroll.Data.Secret == "" || enroll.Data.ProvisioningURI == "" {
		t.Fatalf("enroll returned no secret or URI: %s", body)
	}

	_, plainBody := signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	if accessTokenFrom(t, plainBody) == "" {
		t.Fatal("an unconfirmed enrolment already blocked sign-in; a half-finished setup must not lock anyone out")
	}

	code := codeFor(t, enroll.Data.Secret)
	status, body = authedPost(t, app, "/api/v1/auth/totp/confirm", token, `{"code":"`+code+`"}`)
	if status != http.StatusOK {
		t.Fatalf("confirm returned %d: %s", status, body)
	}
	var confirmed struct {
		Data struct {
			Codes []string `json:"codes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &confirmed); err != nil {
		t.Fatal(err)
	}
	if len(confirmed.Data.Codes) != 10 {
		t.Fatalf("confirm returned %d recovery codes, want 10: %s", len(confirmed.Data.Codes), body)
	}

	status, body = signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	if status != http.StatusOK {
		t.Fatalf("signin returned %d: %s", status, body)
	}
	var gated struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			TOTPRequired bool   `json:"totp_required"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &gated); err != nil {
		t.Fatal(err)
	}
	if !gated.Data.TOTPRequired {
		t.Fatal("signin did not ask for a code after TOTP was confirmed")
	}
	if gated.Data.AccessToken != "" {
		t.Fatalf("signin handed out a token before the code: %s", body)
	}

	status, body = signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"000000"}`)
	if status == http.StatusOK && accessTokenFrom(t, body) != "" {
		t.Fatalf("a wrong code signed in: %s", body)
	}

	status, body = signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+codeInWindow(t, enroll.Data.Secret, 1)+`"}`)
	if status != http.StatusOK || accessTokenFrom(t, body) == "" {
		t.Fatalf("signin with a valid code returned %d: %s", status, body)
	}
}

// A code is accepted across three steps, so without burning the counter one code works twice.
func TestTOTPCodeCannotBeReplayed(t *testing.T) {
	app, secret := enrolledApp(t)

	code := codeInWindow(t, secret, 1)
	status, body := signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+code+`"}`)
	if status != http.StatusOK || accessTokenFrom(t, body) == "" {
		t.Fatalf("first use of the code failed: %d %s", status, body)
	}

	status, body = signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+code+`"}`)
	if status == http.StatusOK && accessTokenFrom(t, body) != "" {
		t.Fatalf("the same code signed in twice: %s", body)
	}
}

// The phone can be lost; a recovery code is the only way back, and only once each.
func TestTOTPRecoveryCodeWorksOnceThenIsSpent(t *testing.T) {
	app, _ := setupAppTOTP(t)
	_, body := signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	token := accessTokenFrom(t, body)

	_, body = authedPost(t, app, "/api/v1/auth/totp/enroll", token, `{}`)
	var enroll struct {
		Data struct{ Secret string } `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &enroll); err != nil {
		t.Fatal(err)
	}
	_, body = authedPost(t, app, "/api/v1/auth/totp/confirm", token, `{"code":"`+codeFor(t, enroll.Data.Secret)+`"}`)
	var confirmed struct {
		Data struct{ Codes []string } `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &confirmed); err != nil {
		t.Fatal(err)
	}
	if len(confirmed.Data.Codes) == 0 {
		t.Fatalf("no recovery codes issued: %s", body)
	}
	recovery := confirmed.Data.Codes[0]

	status, body := signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+recovery+`"}`)
	if status != http.StatusOK || accessTokenFrom(t, body) == "" {
		t.Fatalf("the recovery code did not work: %d %s", status, body)
	}

	status, body = signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+recovery+`"}`)
	if status == http.StatusOK && accessTokenFrom(t, body) != "" {
		t.Fatalf("a recovery code was reusable: %s", body)
	}
}

// OPDS/Kobo use Basic auth and cannot type a code, so the gate must not reach authenticate.
func TestTOTPDoesNotLockOutOPDSBasicAuth(t *testing.T) {
	app, _ := enrolledApp(t)

	credentials := base64.StdEncoding.EncodeToString([]byte("reader@example.com:Sup3rSecret!"))
	req := httptest.NewRequest(http.MethodGet, "/api/opds/v1/", nil)
	req.Header.Set("Authorization", "Basic "+credentials)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("TOTP locked an OPDS reader out; the gate leaked into authenticate")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("OPDS catalog returned %d, want 200", resp.StatusCode)
	}
}

// TOTP is per-user and needs no mail server, so a dead SMTP config cannot gate sign-in.
func TestTOTPIsIndependentOfSMTP(t *testing.T) {
	app, secret := enrolledApp(t)

	status, body := signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	if status != http.StatusOK {
		t.Fatalf("signin returned %d: %s", status, body)
	}
	var gated struct {
		Data struct {
			TOTPRequired bool `json:"totp_required"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &gated); err != nil {
		t.Fatal(err)
	}
	if !gated.Data.TOTPRequired {
		t.Fatal("TOTP did not gate sign-in even though no SMTP server is configured at all")
	}

	status, body = signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+codeInWindow(t, secret, 1)+`"}`)
	if status != http.StatusOK || accessTokenFrom(t, body) == "" {
		t.Fatalf("signin with a code failed without SMTP: %d %s", status, body)
	}
}

// Disabling must actually clear the gate, or a user who turns it off stays locked to a phone.
func TestTOTPDisableRestoresPlainSignin(t *testing.T) {
	app, secret := enrolledApp(t)

	_, body := signinTOTP(t, app,
		`{"email":"reader@example.com","password":"Sup3rSecret!","totp_code":"`+codeInWindow(t, secret, 1)+`"}`)
	token := accessTokenFrom(t, body)
	if token == "" {
		t.Fatalf("could not sign in to disable: %s", body)
	}

	status, body := authedPost(t, app, "/api/v1/auth/totp/disable", token, `{"code":"`+codeInWindow(t, secret, -1)+`"}`)
	if status != http.StatusOK {
		t.Fatalf("disable returned %d: %s", status, body)
	}

	status, body = signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	if status != http.StatusOK || accessTokenFrom(t, body) == "" {
		t.Fatalf("plain signin still blocked after disabling: %d %s", status, body)
	}
}

func enrolledApp(t *testing.T) (*fiber.App, string) {
	t.Helper()
	app, _ := setupAppTOTP(t)

	_, body := signinTOTP(t, app, `{"email":"reader@example.com","password":"Sup3rSecret!"}`)
	token := accessTokenFrom(t, body)

	_, body = authedPost(t, app, "/api/v1/auth/totp/enroll", token, `{}`)
	var enroll struct {
		Data struct{ Secret string } `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &enroll); err != nil {
		t.Fatal(err)
	}
	if status, confirmBody := authedPost(t, app, "/api/v1/auth/totp/confirm", token, `{"code":"`+codeFor(t, enroll.Data.Secret)+`"}`); status != http.StatusOK {
		t.Fatalf("confirm returned %d: %s", status, confirmBody)
	}
	return app, enroll.Data.Secret
}

func codeFor(t *testing.T, secret string) string {
	t.Helper()
	return codeInWindow(t, secret, 0)
}

// A used code burns its step counter, so a test that needs a second code cannot wait 30s for
// the window to roll. Shifting by one period is inside the accepted skew — it is exactly what
// a phone running slightly fast sends — and lands on a different counter.
func codeInWindow(t *testing.T, secret string, periods int) string {
	t.Helper()
	code, err := totp.Generate(secret, time.Now().Add(time.Duration(periods)*totp.Period))
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	return code
}
