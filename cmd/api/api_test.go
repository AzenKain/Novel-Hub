package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"path/filepath"
	"strings"
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

	if err := database.ApplySchema(db); err != nil {
		return nil, nil, err
	}

	ramCache := cache.NewRamCache()

	server := NewHTTPServer()
	server.SetupServer(db, ramCache)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return server.App, db, nil
}

func TestParseTrustProxy(t *testing.T) {
	tests := []struct {
		raw       string
		enabled   bool
		ranges    bool
		allowlist []string
	}{
		{raw: "false", enabled: false},
		{raw: "", enabled: false},
		{raw: "  ", enabled: false},
		{raw: "FALSE", enabled: false},
		{raw: " , , ", enabled: false},
		{raw: "true", enabled: true, ranges: true},
		{raw: "True", enabled: true, ranges: true},
		{raw: "173.245.48.0/20", enabled: true, allowlist: []string{"173.245.48.0/20"}},
		{raw: " 10.0.0.5 , 172.16.0.0/12 ", enabled: true, allowlist: []string{"10.0.0.5", "172.16.0.0/12"}},
	}

	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			enabled, cfg := parseTrustProxy(test.raw)
			if enabled != test.enabled {
				t.Fatalf("enabled = %v, want %v", enabled, test.enabled)
			}
			gotRanges := cfg.Loopback || cfg.Private || cfg.LinkLocal
			if gotRanges != test.ranges {
				t.Errorf("built-in ranges = %v, want %v", gotRanges, test.ranges)
			}
			if len(cfg.Proxies) != len(test.allowlist) {
				t.Fatalf("proxies = %#v, want %#v", cfg.Proxies, test.allowlist)
			}
			for i, want := range test.allowlist {
				if cfg.Proxies[i] != want {
					t.Errorf("proxies[%d] = %q, want %q", i, cfg.Proxies[i], want)
				}
			}
		})
	}
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

func TestSigninWorksAfterAFailedAttempt(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'typo@example.com', 'Typo', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'
	`, userID); err != nil {
		t.Fatalf("failed to seed roles: %v", err)
	}

	signin := func(password string) int {
		body := []byte(`{"email":"typo@example.com","password":"` + password + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("signin request failed: %v", err)
		}
		return resp.StatusCode
	}

	if code := signin("wrong-password"); code != http.StatusUnauthorized {
		t.Fatalf("wrong password: expected 401, got %d", code)
	}
	if code := signin("password123"); code != http.StatusOK {
		t.Fatalf("correct password after a typo: expected 200, got %d", code)
	}
}

func TestRateLimitAppliesWithoutRestart(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'ratelimit-admin@example.com', 'Admin', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT ?, id FROM roles WHERE name IN ('USER', 'ADMIN')
	`, userID); err != nil {
		t.Fatalf("failed to seed roles: %v", err)
	}

	signinBody := []byte(`{"email":"ratelimit-admin@example.com","password":"password123"}`)
	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(signinBody))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin request failed: %v", err)
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
	token := signinData.Data.AccessToken

	putLimit := func(max int, window int) {
		body := fmt.Appendf(nil, `{"limits.rate_limit_auth":%d,"limits.rate_limit_auth_window_seconds":%d}`, max, window)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("settings request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected settings status 200, got %d: %s", resp.StatusCode, string(respBody))
		}
	}

	attemptSignin := func() int {
		body := []byte(`{"email":"ratelimit-admin@example.com","password":"wrong-password"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("signin attempt failed: %v", err)
		}
		return resp.StatusCode
	}

	putLimit(1000, 3600)
	for i := range 5 {
		if code := attemptSignin(); code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d was rate limited while the budget was still 1000", i)
		}
	}

	putLimit(1, 3600)

	sawTooMany := false
	for range 40 {
		if attemptSignin() == http.StatusTooManyRequests {
			sawTooMany = true
			break
		}
	}
	if !sawTooMany {
		t.Fatal("expected a 429 once the sign-in budget was lowered to 1/hour")
	}
}

func TestReadingGoalDefaultsThenPersists(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'goal-user@example.com', 'Reader', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT ?, id FROM roles WHERE name IN ('USER', 'ADMIN')
	`, userID); err != nil {
		t.Fatalf("failed to seed roles: %v", err)
	}

	signinBody := []byte(`{"email":"goal-user@example.com","password":"password123"}`)
	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(signinBody))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin request failed: %v", err)
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
	token := signinData.Data.AccessToken

	getGoal := func() response.ReadingGoalResponse {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reader/goals/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("get goal failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected goal status 200, got %d: %s", resp.StatusCode, string(body))
		}
		var out struct {
			Status bool                         `json:"status"`
			Data   response.ReadingGoalResponse `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("failed to decode goal: %v", err)
		}
		return out.Data
	}

	if goal := getGoal(); goal.TargetWordsPerDay != 1000 || goal.TargetBooksPerYear != 12 {
		t.Fatalf("expected defaults 1000/12 for a user with no goal, got %d/%d", goal.TargetWordsPerDay, goal.TargetBooksPerYear)
	}

	putBody := []byte(`{"target_words_per_day":2500,"target_books_per_year":40}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/reader/goals/", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+token)
	putResp, err := app.Test(putReq)
	if err != nil {
		t.Fatalf("upsert goal failed: %v", err)
	}
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("expected upsert status 200, got %d: %s", putResp.StatusCode, string(body))
	}

	if goal := getGoal(); goal.TargetWordsPerDay != 2500 || goal.TargetBooksPerYear != 40 {
		t.Fatalf("expected persisted 2500/40, got %d/%d", goal.TargetWordsPerDay, goal.TargetBooksPerYear)
	}

	badReq := httptest.NewRequest(http.MethodPut, "/api/v1/reader/goals/", bytes.NewReader([]byte(`{"target_words_per_day":0,"target_books_per_year":40}`)))
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("Authorization", "Bearer "+token)
	badResp, err := app.Test(badReq)
	if err != nil {
		t.Fatalf("bad upsert request failed: %v", err)
	}
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a zero words/day goal, got %d", badResp.StatusCode)
	}
}

func TestSmartCollectionRoundTripAndIsolation(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	signIn := func(email string) string {
		userID := uuid.Must(uuid.NewV7()).String()
		if _, err := db.Exec(`
			INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
			VALUES (?, ?, 'Reader', ?, 'LOCAL', 1)
		`, userID, email, string(hash)); err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}
		if _, err := db.Exec(`
			INSERT INTO user_roles (user_id, role_id)
			SELECT ?, id FROM roles WHERE name IN ('USER', 'ADMIN')
		`, userID); err != nil {
			t.Fatalf("failed to seed roles: %v", err)
		}
		body := []byte(`{"email":"` + email + `","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("signin request failed: %v", err)
		}
		var out struct {
			Status bool                  `json:"status"`
			Data   response.AuthResponse `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("failed to decode signin: %v", err)
		}
		if !out.Status || out.Data.AccessToken == "" {
			t.Fatalf("signin did not return a token for %s", email)
		}
		return out.Data.AccessToken
	}

	list := func(token string) []response.SmartCollectionResponse {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/smart-collections/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("list request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected list status 200, got %d: %s", resp.StatusCode, string(body))
		}
		var out struct {
			Status bool                               `json:"status"`
			Data   []response.SmartCollectionResponse `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("failed to decode list: %v", err)
		}
		return out.Data
	}

	alice := signIn("smart-alice@example.com")
	bob := signIn("smart-bob@example.com")

	createBody := []byte(`{"name":"Unread sci-fi","rule":{"search":"dune","chip":"Unread","facet":"tag","facet_id":"scifi","evil":"<script>"}}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/smart-collections/", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+alice)
	createResp, err := app.Test(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected create status 200, got %d: %s", createResp.StatusCode, string(body))
	}

	items := list(alice)
	if len(items) != 1 {
		t.Fatalf("expected 1 smart collection for alice, got %d", len(items))
	}
	saved := items[0]
	if saved.Name != "Unread sci-fi" {
		t.Errorf("name = %q, want %q", saved.Name, "Unread sci-fi")
	}
	if saved.Rule.Search != "dune" || saved.Rule.Chip != "Unread" || saved.Rule.Facet != "tag" || saved.Rule.FacetID != "scifi" {
		t.Errorf("rule did not round trip: %#v", saved.Rule)
	}

	if got := list(bob); len(got) != 0 {
		t.Fatalf("bob sees %d of alice's smart collections, want 0", len(got))
	}
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/smart-collections/"+saved.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+bob)
	if _, err := app.Test(delReq); err != nil {
		t.Fatalf("bob delete request failed: %v", err)
	}
	if got := list(alice); len(got) != 1 {
		t.Fatalf("bob deleted alice's smart collection: alice now has %d", len(got))
	}

	ownDelReq := httptest.NewRequest(http.MethodDelete, "/api/v1/smart-collections/"+saved.ID, nil)
	ownDelReq.Header.Set("Authorization", "Bearer "+alice)
	ownDelResp, err := app.Test(ownDelReq)
	if err != nil {
		t.Fatalf("alice delete request failed: %v", err)
	}
	if ownDelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d", ownDelResp.StatusCode)
	}
	if got := list(alice); len(got) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(got))
	}
}

func TestPermissionDeniedUsesStandardEnvelope(t *testing.T) {
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
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'plain@example.com', 'Plain', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'`, userID); err != nil {
		t.Fatal(err)
	}

	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin",
		bytes.NewReader([]byte(`{"email":"plain@example.com","password":"password123"}`)))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := app.Test(signinReq)
	if err != nil {
		t.Fatal(err)
	}
	var auth struct {
		Data response.AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(signinResp.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Data.AccessToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for a user without user.manage, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var envelope response.CommonResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("403 body is not JSON at all: %v (%s)", err, body)
	}
	if envelope.Status {
		t.Fatalf("403 envelope reports status=true: %s", body)
	}
	if envelope.Message == "" {
		t.Fatalf("403 envelope has no message, frontend has nothing to show: %s", body)
	}
}

func TestBooksNextCursorSurvivesFilteredOutBooks(t *testing.T) {
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "novelhub-cursor-test.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	openLib := uuid.Must(uuid.NewV7()).String()
	hiddenLib := uuid.Must(uuid.NewV7()).String()
	for id, name := range map[string]string{openLib: "Open", hiddenLib: "Hidden"} {
		if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES (?, ?)`, id, name); err != nil {
			t.Fatalf("seed library %s: %v", name, err)
		}
	}
	if _, err := db.Exec(`
		UPDATE app_settings SET value_json = ? WHERE key = 'guest_access.mode'
	`, `"selected_libraries"`); err != nil {
		t.Fatalf("set guest mode: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE app_settings SET value_json = ? WHERE key = 'guest_access.library_ids'
	`, `["`+openLib+`"]`); err != nil {
		t.Fatalf("set guest libraries: %v", err)
	}

	const total = 6
	for i := 0; i < total; i++ {
		library := openLib
		if i%2 == 1 {
			library = hiddenLib
		}
		if _, err := db.Exec(`
			INSERT INTO books (id, library_id, title, status, created_at, updated_at)
			VALUES (?, ?, ?, 'active', datetime('now', ?), datetime('now', ?))
		`, uuid.Must(uuid.NewV7()).String(), library, fmt.Sprintf("Book %d", i),
			fmt.Sprintf("-%d seconds", total-i), fmt.Sprintf("-%d seconds", total-i)); err != nil {
			t.Fatalf("seed book %d: %v", i, err)
		}
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())
	app := server.App

	firstResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/books?limit=100", nil))
	if err != nil {
		t.Fatalf("probe request failed: %v", err)
	}
	var probe struct {
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	probeBody, _ := io.ReadAll(firstResp.Body)
	if err := json.Unmarshal(probeBody, &probe); err != nil {
		t.Fatalf("probe bad json: %v (%s)", err, probeBody)
	}
	if len(probe.Data) != 3 {
		t.Fatalf("guest sees %d books, want 3 — the permission filter is not applied, so this test cannot detect the cursor bug", len(probe.Data))
	}

	seen := map[string]bool{}
	cursor := ""
	for round := 0; round < total+2; round++ {
		url := "/api/v1/books?limit=3"
		if cursor != "" {
			url += "&cursor=" + neturl.QueryEscape(cursor)
		}
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, url, nil))
		if err != nil {
			t.Fatalf("round %d: request failed: %v", round, err)
		}
		var page struct {
			Status     bool   `json:"status"`
			Pagination struct {
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
			Data       []struct {
				Title string `json:"title"`
			} `json:"data"`
		}
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatalf("round %d: bad json: %v (%s)", round, err, body)
		}
		if !page.Status {
			t.Fatalf("round %d: status=false: %s", round, body)
		}
		for _, book := range page.Data {
			seen[book.Title] = true
		}
		if page.Pagination.NextCursor == "" {
			break
		}
		if page.Pagination.NextCursor == cursor {
			t.Fatalf("round %d: cursor did not advance (%q); the client would loop forever", round, cursor)
		}
		cursor = page.Pagination.NextCursor
	}

	for _, title := range []string{"Book 0", "Book 2", "Book 4"} {
		if !seen[title] {
			t.Errorf("%q was never returned: paging stopped early, so it is unreachable from the client", title)
		}
	}
}

// A missing row used to reach the client as 500 with "sql: no rows in result set" as the message.
// Repositories return sql.ErrNoRows raw by convention — no repository imports apperrors — and
// HandleError only recognised its own AppError kinds, so every "book not found" was a 500 that
// also put the storage engine's wording on the wire. Fixed in HandleError rather than in each
// repository: one guard covers every caller, including the ones nobody has audited.
func TestMissingRowIsNotFoundNotServerError(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	const missing = "00000000-0000-7000-8000-000000000000"
	for _, path := range []string{
		"/api/v1/books/" + missing,
		"/api/v1/books/" + missing + "/download",
		"/api/v1/libraries/" + missing,
		// These four read the book then check permission. They used to collapse both into one
		// 403 "You do not have access to this book", which is the wrong status and the wrong
		// story: it implies the book exists. GetBootstrap always split them; these diverged.
		"/api/v1/reader/" + missing + "/bootstrap",
		"/api/v1/reader/" + missing + "/chapter/x",
		"/api/v1/reader/" + missing + "/file",
		"/api/v1/reader/" + missing + "/images",
	} {
		t.Run(path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("status = %d, want 404: %s", resp.StatusCode, body)
			}

			var envelope response.CommonResponse
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatalf("body is not the standard envelope: %v (%s)", err, body)
			}
			if envelope.Status {
				t.Error("status must be false on an error")
			}
			// The driver's wording names the storage engine and tells a client nothing actionable.
			if strings.Contains(envelope.Message, "sql:") || strings.Contains(envelope.Message, "no rows") {
				t.Errorf("message leaks driver text: %q", envelope.Message)
			}
		})
	}
}
