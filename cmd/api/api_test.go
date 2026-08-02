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

// The point of routing the rate limit through RuntimeLimits is that an admin
// change applies to the very next request, with no restart. This drives that
// path the way an operator would: sign in, PUT the setting, then get 429'd.
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

	health := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("health request failed: %v", err)
		}
		return resp.StatusCode
	}

	// Default budget is roomy, so a handful of calls must all succeed.
	for i := range 5 {
		if code := health(); code != http.StatusOK {
			t.Fatalf("request %d before tightening: expected 200, got %d", i, code)
		}
	}

	settingsBody := []byte(`{"limits.rate_limit_api":10,"limits.rate_limit_api_window_seconds":3600}`)
	settingsReq := httptest.NewRequest(http.MethodPut, "/api/v1/settings/", bytes.NewReader(settingsBody))
	settingsReq.Header.Set("Content-Type", "application/json")
	settingsReq.Header.Set("Authorization", "Bearer "+token)
	settingsResp, err := app.Test(settingsReq)
	if err != nil {
		t.Fatalf("settings request failed: %v", err)
	}
	if settingsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(settingsResp.Body)
		t.Fatalf("expected settings status 200, got %d: %s", settingsResp.StatusCode, string(body))
	}

	// No restart between the PUT above and the calls below.
	sawTooMany := false
	for i := range 40 {
		if health() == http.StatusTooManyRequests {
			sawTooMany = true
			break
		}
		if i == 39 {
			t.Fatal("rate limit never engaged after lowering it to 10/hour")
		}
	}
	if !sawTooMany {
		t.Fatal("expected a 429 once the tightened limit took effect")
	}
}

// A user who never set a goal must get usable defaults, not a 404 — the analytics
// page always needs a denominator. Round-tripping then proves the upsert persists
// past the repository cache.
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

	putBody := []byte(`{"targetWordsPerDay":2500,"targetBooksPerYear":40}`)
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

	// Zero must be rejected: a 0 words/day target would divide by zero on the client.
	badReq := httptest.NewRequest(http.MethodPut, "/api/v1/reader/goals/", bytes.NewReader([]byte(`{"targetWordsPerDay":0,"targetBooksPerYear":40}`)))
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

// rule_json is replayed into a library URL on the client, so it is a trust
// boundary: only the seven known filter fields may survive a round trip, and a
// caller must not be able to read another user's saved searches.
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

	// "evil" is not a rule field; it must be dropped, not stored and replayed.
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

	// Bob must not see alice's saved search, nor be able to delete it.
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

	// Alice can delete her own.
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
