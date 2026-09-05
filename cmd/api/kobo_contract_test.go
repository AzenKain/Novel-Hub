package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

type koboFixture struct {
	app    *fiber.App
	db     *sql.DB
	token  string
	userID string
	bookID string
	libID  string
}

func setupKoboFixture(t *testing.T, seed ...func(*testing.T, *sql.DB, koboSeed)) koboFixture {
	t.Helper()
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "novelhub-kobo-test.db"))

	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	ids := seedKoboData(t, db)
	for _, hook := range seed {
		hook(t, db, ids)
	}

	server := NewHTTPServer()
	server.SetupServer(db, cache.NewRamCache())

	return koboFixture{app: server.App, db: db, token: ids.Token, userID: ids.UserID, bookID: ids.BookID, libID: ids.LibraryID}
}

type koboSeed struct {
	UserID    string
	LibraryID string
	BookID    string
	Token     string
}

func seedKoboData(t *testing.T, db *sql.DB) koboSeed {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, 'kobo-test@example.com', 'Kobo User', ?, 'LOCAL', 1)
	`, userID, string(hash)); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'
	`, userID); err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	libID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES (?, 'Main')`, libID); err != nil {
		t.Fatalf("seed library: %v", err)
	}

	bookID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO books (id, library_id, title, description, status, metadata_json, cover_url)
		VALUES (?, ?, 'Kobo Test Book', 'A description.', 'active',
		        '{"publisher":"NovelHub Press","language":"vi","series":"Kobo Series","series_index":3}',
		        '/storage/books/' || ? || '/cover.jpg')
	`, bookID, libID, bookID); err != nil {
		t.Fatalf("seed book: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time)
		VALUES (?, ?, ?, 'EPUB', 4096, CURRENT_TIMESTAMP)
	`, uuid.Must(uuid.NewV7()).String(), bookID, "/tmp/kobo-test-"+bookID+".epub"); err != nil {
		t.Fatalf("seed book file: %v", err)
	}

	token := "0123456789abcdef0123456789abcdef"
	if _, err := db.Exec(`INSERT INTO kobo_auth_tokens (token, user_id) VALUES (?, ?)`, token, userID); err != nil {
		t.Fatalf("seed kobo token: %v", err)
	}

	return koboSeed{UserID: userID, LibraryID: libID, BookID: bookID, Token: token}
}

func (f koboFixture) get(t *testing.T, path string, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/kobo/"+f.token+path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, into any) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("decode %T: %v\nbody: %s", into, err, body)
	}
}

// A Kobo reader sends no Authorization header — it has one configurable setting, api_endpoint.
func TestKoboAuthenticatesFromPathTokenWithoutHeaders(t *testing.T) {
	f := setupKoboFixture(t)
	resp := f.get(t, "/v1/initialization", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("initialization with path token = %d, want 200: %s", resp.StatusCode, body)
	}
}

func TestKoboRejectsUnknownToken(t *testing.T) {
	f := setupKoboFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/kobo/deadbeefdeadbeefdeadbeefdeadbeef/v1/initialization", nil)
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unknown token = %d, want 401", resp.StatusCode)
	}
}

func TestKoboRejectsTokenOfDeletedUser(t *testing.T) {
	f := setupKoboFixture(t)
	if _, err := f.db.Exec(`UPDATE users SET is_deleted = 1 WHERE id = ?`, f.userID); err != nil {
		t.Fatalf("soft-delete user: %v", err)
	}
	resp := f.get(t, "/v1/initialization", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("token of deleted user = %d, want 401", resp.StatusCode)
	}
}

// The device derives every URL it calls from this map, so a short map silently disables features (covers above all).
func TestKoboInitializationReturnsFullResourceMap(t *testing.T) {
	f := setupKoboFixture(t)
	resp := f.get(t, "/v1/initialization", nil)

	var payload struct {
		Resources map[string]any `json:"Resources"`
	}
	decodeJSON(t, resp, &payload)

	if len(payload.Resources) != 147 {
		t.Errorf("Resources has %d keys, want 147", len(payload.Resources))
	}
	sync, _ := payload.Resources["library_sync"].(string)
	if want := "/kobo/" + f.token + "/v1/library/sync"; !bytes.Contains([]byte(sync), []byte(want)) {
		t.Errorf("library_sync = %q, want it to contain %q", sync, want)
	}
	tmpl, _ := payload.Resources["image_url_template"].(string)
	for _, ph := range []string{"{ImageId}", "{width}", "{height}"} {
		if !bytes.Contains([]byte(tmpl), []byte(ph)) {
			t.Errorf("image_url_template = %q, missing %s", tmpl, ph)
		}
	}
	if got, _ := payload.Resources["account_page"].(string); got != "https://www.kobo.com/account/settings" {
		t.Errorf("account_page = %q, want the upstream URL", got)
	}
	if got := resp.Header.Get("x-kobo-apitoken"); got != "e30=" {
		t.Errorf("x-kobo-apitoken = %q, want e30=", got)
	}
}

// calibre-web returns a throwaway response here purely so the device's login step succeeds.
func TestKoboAuthDeviceReturnsDummyShape(t *testing.T) {
	f := setupKoboFixture(t)
	body := []byte(`{"UserKey":"device-user-key","AffiliateName":"Kobo","DeviceId":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/kobo/"+f.token+"/v1/auth/device", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("auth request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("auth/device = %d, want 200: %s", resp.StatusCode, raw)
	}

	var got map[string]any
	decodeJSON(t, resp, &got)
	for _, key := range []string{"AccessToken", "RefreshToken", "TokenType", "TrackingId", "UserKey"} {
		if _, ok := got[key]; !ok {
			t.Errorf("auth response missing %s", key)
		}
	}
	if got["TokenType"] != "Bearer" {
		t.Errorf("TokenType = %v, want Bearer", got["TokenType"])
	}
	if got["UserKey"] != "device-user-key" {
		t.Errorf("UserKey = %v, want it echoed from the request", got["UserKey"])
	}
	req2 := httptest.NewRequest(http.MethodPost, "/kobo/"+f.token+"/v1/auth/refresh", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := f.app.Test(req2)
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}
	var got2 map[string]any
	decodeJSON(t, resp2, &got2)
	if got["AccessToken"] == got2["AccessToken"] {
		t.Error("AccessToken repeated across calls; it must be freshly random")
	}
}

// The sync body is a bare JSON array, not an object and never null: the device aborts parsing otherwise.
func TestKoboSyncReturnsNewEntitlements(t *testing.T) {
	f := setupKoboFixture(t)
	resp := f.get(t, "/v1/library/sync", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("sync = %d, want 200: %s", resp.StatusCode, body)
	}

	var items []map[string]json.RawMessage
	decodeJSON(t, resp, &items)
	if len(items) != 1 {
		t.Fatalf("sync returned %d items, want 1", len(items))
	}
	raw, ok := items[0]["NewEntitlement"]
	if !ok {
		t.Fatalf("first sync must use NewEntitlement, got keys %v", items[0])
	}

	var entitlement struct {
		BookEntitlement map[string]any `json:"BookEntitlement"`
		BookMetadata    map[string]any `json:"BookMetadata"`
	}
	if err := json.Unmarshal(raw, &entitlement); err != nil {
		t.Fatalf("decode entitlement: %v", err)
	}

	for _, key := range []string{
		"Accessibility", "ActivePeriod", "Created", "CrossRevisionId", "Id", "IsRemoved",
		"IsHiddenFromArchive", "IsLocked", "LastModified", "OriginCategory", "RevisionId", "Status",
	} {
		if _, ok := entitlement.BookEntitlement[key]; !ok {
			t.Errorf("BookEntitlement missing %s", key)
		}
	}
	if entitlement.BookEntitlement["Id"] != f.bookID {
		t.Errorf("BookEntitlement.Id = %v, want the book UUID %s", entitlement.BookEntitlement["Id"], f.bookID)
	}

	for _, key := range []string{
		"CoverImageId", "CurrentDisplayPrice", "Description", "DownloadUrls", "EntitlementId",
		"Language", "Publisher", "Title", "WorkId",
	} {
		if _, ok := entitlement.BookMetadata[key]; !ok {
			t.Errorf("BookMetadata missing %s", key)
		}
	}
	downloads, _ := entitlement.BookMetadata["DownloadUrls"].([]any)
	if len(downloads) == 0 {
		t.Fatal("DownloadUrls is empty; the device would show the book but fail to open it")
	}
	first, _ := downloads[0].(map[string]any)
	url, _ := first["Url"].(string)
	if !bytes.Contains([]byte(url), []byte("/kobo/"+f.token+"/download/"+f.bookID)) {
		t.Errorf("download URL = %q, want it token-scoped for this book", url)
	}
	if entitlement.BookMetadata["Language"] != "vi" {
		t.Errorf("Language = %v, want vi from metadata_json", entitlement.BookMetadata["Language"])
	}
	series, _ := entitlement.BookMetadata["Series"].(map[string]any)
	if series == nil || series["Name"] != "Kobo Series" {
		t.Errorf("Series = %#v, want the series from metadata_json", series)
	}

	if resp.Header.Get("x-kobo-synctoken") == "" {
		t.Error("x-kobo-synctoken missing; the device would re-sync everything forever")
	}
	if got := resp.Header.Get("x-kobo-sync"); got == "continue" {
		t.Error("x-kobo-sync: continue set for a single-book library")
	}
}

// Second sync must not resend a book the device already has, otherwise every sync re-downloads the library.
func TestKoboSyncIsIncremental(t *testing.T) {
	f := setupKoboFixture(t)

	first := f.get(t, "/v1/library/sync", nil)
	token := first.Header.Get("x-kobo-synctoken")
	if token == "" {
		t.Fatal("first sync returned no token")
	}
	var firstItems []map[string]json.RawMessage
	decodeJSON(t, first, &firstItems)
	if len(firstItems) != 1 {
		t.Fatalf("first sync returned %d items, want 1", len(firstItems))
	}

	second := f.get(t, "/v1/library/sync", map[string]string{"x-kobo-synctoken": token})
	var secondItems []map[string]json.RawMessage
	decodeJSON(t, second, &secondItems)
	if len(secondItems) != 0 {
		t.Errorf("second sync returned %d items, want 0 — the book was already synced", len(secondItems))
	}
}

// An empty result must serialise as [] — a null body has been observed to abort device parsing.
func TestKoboSyncEmptyResultIsArrayNotNull(t *testing.T) {
	f := setupKoboFixture(t)
	if _, err := f.db.Exec(`DELETE FROM book_files`); err != nil {
		t.Fatalf("clear files: %v", err)
	}
	if _, err := f.db.Exec(`DELETE FROM books`); err != nil {
		t.Fatalf("clear books: %v", err)
	}
	resp := f.get(t, "/v1/library/sync", nil)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if got := string(bytes.TrimSpace(body)); got != "[]" {
		t.Errorf("empty sync body = %q, want []", got)
	}
}

// An unopened book must arrive with no ReadingState at all.
func TestKoboSyncOmitsReadingStateForUnopenedBook(t *testing.T) {
	f := setupKoboFixture(t)

	var items []map[string]json.RawMessage
	decodeJSON(t, f.get(t, "/v1/library/sync", nil), &items)
	if len(items) != 1 {
		t.Fatalf("sync returned %d items, want 1", len(items))
	}
	var entitlement map[string]json.RawMessage
	if err := json.Unmarshal(items[0]["NewEntitlement"], &entitlement); err != nil {
		t.Fatalf("decode entitlement: %v", err)
	}
	if raw, ok := entitlement["ReadingState"]; ok {
		t.Errorf("an unopened book was synced with a ReadingState: %s", raw)
	}
}

// The mirror of the test above: once the book has been opened, the state must be there.
func TestKoboSyncCarriesReadingStateForOpenedBook(t *testing.T) {
	f := setupKoboFixture(t)
	if _, err := f.db.Exec(`
		INSERT INTO reading_progress (user_id, book_id, chapter_ref, progress_percent, opened_count)
		VALUES (?, ?, 'chapter-1', 42.0, 1)
	`, f.userID, f.bookID); err != nil {
		t.Fatalf("seed progress: %v", err)
	}

	var items []map[string]json.RawMessage
	decodeJSON(t, f.get(t, "/v1/library/sync", nil), &items)
	if len(items) != 1 {
		t.Fatalf("sync returned %d items, want 1", len(items))
	}
	var entitlement map[string]json.RawMessage
	if err := json.Unmarshal(items[0]["NewEntitlement"], &entitlement); err != nil {
		t.Fatalf("decode entitlement: %v", err)
	}
	raw, ok := entitlement["ReadingState"]
	if !ok {
		t.Fatal("an opened book was synced without its ReadingState; the device loses the position")
	}
	if !bytes.Contains(raw, []byte("42")) {
		t.Errorf("ReadingState carries no progress: %s", raw)
	}
}

func TestKoboBookMetadataEndpoint(t *testing.T) {
	f := setupKoboFixture(t)
	resp := f.get(t, "/v1/library/"+f.bookID+"/metadata", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("metadata = %d, want 200: %s", resp.StatusCode, body)
	}
	var got []map[string]any
	decodeJSON(t, resp, &got)
	if len(got) != 1 {
		t.Fatalf("metadata returned %d entries, want 1", len(got))
	}
	if got[0]["EntitlementId"] != f.bookID {
		t.Errorf("EntitlementId = %v, want %s", got[0]["EntitlementId"], f.bookID)
	}
}

func TestKoboMetadataForUnknownBookIs404(t *testing.T) {
	f := setupKoboFixture(t)
	resp := f.get(t, "/v1/library/"+uuid.Must(uuid.NewV7()).String()+"/metadata", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown book metadata = %d, want 404", resp.StatusCode)
	}
}

// GET state must answer for a book that has never been opened; calibre-web creates an empty row rather than 404ing, and a device that gets an error here stops syncing that book.
func TestKoboReadingStateForUnreadBook(t *testing.T) {
	f := setupKoboFixture(t)
	resp := f.get(t, "/v1/library/"+f.bookID+"/state", nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("state = %d, want 200: %s", resp.StatusCode, body)
	}

	var states []map[string]any
	decodeJSON(t, resp, &states)
	if len(states) != 1 {
		t.Fatalf("state returned %d entries, want 1", len(states))
	}
	state := states[0]
	for _, key := range []string{"EntitlementId", "Created", "LastModified", "PriorityTimestamp", "StatusInfo", "Statistics", "CurrentBookmark"} {
		if _, ok := state[key]; !ok {
			t.Errorf("ReadingState missing %s", key)
		}
	}
	status, _ := state["StatusInfo"].(map[string]any)
	if status["Status"] != "ReadyToRead" {
		t.Errorf("Status = %v, want ReadyToRead for an unopened book", status["Status"])
	}
	if state["PriorityTimestamp"] != state["LastModified"] {
		t.Errorf("PriorityTimestamp %v != LastModified %v", state["PriorityTimestamp"], state["LastModified"])
	}
}

// The body below is the shape a device PUTs.
func TestKoboPutReadingStatePersistsProgress(t *testing.T) {
	f := setupKoboFixture(t)

	body := []byte(`{
	  "ReadingStates": [{
	    "CurrentBookmark": {
	      "ProgressPercent": 63,
	      "ContentSourceProgressPercent": 61,
	      "Location": {"Value": "kobo.11.2", "Type": "KoboSpan", "Source": "kepub"}
	    },
	    "Statistics": {"SpentReadingMinutes": 42, "RemainingTimeMinutes": 18},
	    "StatusInfo": {"Status": "Reading"}
	  }]
	}`)
	req := httptest.NewRequest(http.MethodPut, "/kobo/"+f.token+"/v1/library/"+f.bookID+"/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("put state: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("put state = %d, want 200: %s", resp.StatusCode, raw)
	}

	var putResp struct {
		RequestResult string `json:"RequestResult"`
		UpdateResults []struct {
			EntitlementID         string            `json:"EntitlementId"`
			CurrentBookmarkResult map[string]string `json:"CurrentBookmarkResult"`
			StatisticsResult      map[string]string `json:"StatisticsResult"`
			StatusInfoResult      map[string]string `json:"StatusInfoResult"`
			LastModified          string            `json:"LastModified"`
			PriorityTimestamp     string            `json:"PriorityTimestamp"`
		} `json:"UpdateResults"`
	}
	decodeJSON(t, resp, &putResp)

	if putResp.RequestResult != "Success" {
		t.Errorf("RequestResult = %q, want Success", putResp.RequestResult)
	}
	if len(putResp.UpdateResults) != 1 {
		t.Fatalf("UpdateResults has %d entries, want 1", len(putResp.UpdateResults))
	}
	result := putResp.UpdateResults[0]
	if result.EntitlementID != f.bookID {
		t.Errorf("EntitlementId = %q, want %s", result.EntitlementID, f.bookID)
	}
	for name, sub := range map[string]map[string]string{
		"CurrentBookmarkResult": result.CurrentBookmarkResult,
		"StatisticsResult":      result.StatisticsResult,
		"StatusInfoResult":      result.StatusInfoResult,
	} {
		if sub["Result"] != "Success" {
			t.Errorf("%s = %v, want {Result: Success}", name, sub)
		}
	}
	if result.LastModified == "" || result.PriorityTimestamp == "" {
		t.Error("LastModified/PriorityTimestamp must be set in the PUT response")
	}

	getResp := f.get(t, "/v1/library/"+f.bookID+"/state", nil)
	var states []map[string]any
	decodeJSON(t, getResp, &states)
	bookmark, _ := states[0]["CurrentBookmark"].(map[string]any)
	if bookmark == nil {
		t.Fatal("CurrentBookmark missing after PUT")
	}
	if pct, _ := bookmark["ProgressPercent"].(float64); pct != 63 {
		t.Errorf("ProgressPercent read back as %v, want 63", bookmark["ProgressPercent"])
	}
	location, _ := bookmark["Location"].(map[string]any)
	if location == nil || location["Value"] != "kobo.11.2" {
		t.Errorf("Location read back as %#v, want the stored value", location)
	}
	status, _ := states[0]["StatusInfo"].(map[string]any)
	if status["Status"] != "Reading" {
		t.Errorf("Status = %v, want Reading at 63%%", status["Status"])
	}
}

func TestKoboPutReadingStateRejectsMalformedBody(t *testing.T) {
	f := setupKoboFixture(t)
	req := httptest.NewRequest(http.MethodPut, "/kobo/"+f.token+"/v1/library/"+f.bookID+"/state", bytes.NewReader([]byte(`{"ReadingStates":[]}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("put state: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty ReadingStates = %d, want 400", resp.StatusCode)
	}
}

// Both cover URL shapes the device builds from the two templates must route.
func TestKoboCoverRoutesBothTemplateVariants(t *testing.T) {
	f := setupKoboFixture(t)
	for _, path := range []string{
		"/" + f.bookID + "/400/600/false/image.jpg",
		"/" + f.bookID + "/400/600/85/false/image.jpg",
	} {
		resp := f.get(t, path, nil)
		if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusInternalServerError {
			t.Errorf("cover %s = %d, want the route to exist", path, resp.StatusCode)
		}
	}
}

// A token grants only its own user's access.
func TestKoboRequiresKoboSyncPermission(t *testing.T) {
	f := setupKoboFixture(t, func(t *testing.T, db *sql.DB, ids koboSeed) {
		if _, err := db.Exec(`
			DELETE FROM role_permissions
			WHERE permission_key = 'kobo.sync'
			  AND role_id IN (SELECT role_id FROM user_roles WHERE user_id = ?)
		`, ids.UserID); err != nil {
			t.Fatalf("revoke permission: %v", err)
		}
	})

	resp := f.get(t, "/v1/library/sync", nil)
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("sync without kobo.sync = %d, want 403: %s", resp.StatusCode, body)
	}
}

// A banned user's device stops syncing, same as the JWT path.
func TestKoboRejectsBannedUser(t *testing.T) {
	f := setupKoboFixture(t, func(t *testing.T, db *sql.DB, ids koboSeed) {
		if _, err := db.Exec(`
			INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'BANNED'
		`, ids.UserID); err != nil {
			t.Fatalf("assign banned role: %v", err)
		}
	})
	resp := f.get(t, "/v1/initialization", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("banned user = %d, want 403", resp.StatusCode)
	}
}

// Touching the token updates last_used_at, which is the only signal the settings UI has for "is this device still in use".
func TestKoboTouchesTokenLastUsed(t *testing.T) {
	f := setupKoboFixture(t)
	f.get(t, "/v1/initialization", nil)

	var lastUsed sql.NullString
	if err := f.db.QueryRow(`SELECT last_used_at FROM kobo_auth_tokens WHERE token = ?`, f.token).Scan(&lastUsed); err != nil {
		t.Fatalf("read last_used_at: %v", err)
	}
	if !lastUsed.Valid || lastUsed.String == "" {
		t.Error("last_used_at not set after a device request")
	}
}

// A library larger than one sync page must still sync completely.
func TestKoboSyncPagesBeyondFirstPage(t *testing.T) {
	const total = 150
	f := setupKoboFixture(t, func(t *testing.T, db *sql.DB, ids koboSeed) {
		for i := 1; i < total; i++ {
			bookID := uuid.Must(uuid.NewV7()).String()
			if _, err := db.Exec(`
				INSERT INTO books (id, library_id, title, status, created_at, updated_at)
				VALUES (?, ?, ?, 'active', datetime('now', ?), datetime('now', ?))
			`, bookID, ids.LibraryID, "Bulk Book", "-"+strconv.Itoa(total-i)+" seconds", "-"+strconv.Itoa(total-i)+" seconds"); err != nil {
				t.Fatalf("seed book %d: %v", i, err)
			}
			if _, err := db.Exec(`
				INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time)
				VALUES (?, ?, ?, 'EPUB', 2048, CURRENT_TIMESTAMP)
			`, uuid.Must(uuid.NewV7()).String(), bookID, "/tmp/kobo-bulk-"+bookID+".epub"); err != nil {
				t.Fatalf("seed file %d: %v", i, err)
			}
		}
	})

	seen := 0
	token := ""
	for round := 1; round <= 5; round++ {
		headers := map[string]string{}
		if token != "" {
			headers["x-kobo-synctoken"] = token
		}
		resp := f.get(t, "/v1/library/sync", headers)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("sync round %d = %d: %s", round, resp.StatusCode, body)
		}

		var items []map[string]json.RawMessage
		decodeJSON(t, resp, &items)
		seen += len(items)
		token = resp.Header.Get("x-kobo-synctoken")

		wantsMore := resp.Header.Get("x-kobo-sync") == "continue"
		if len(items) == 0 && wantsMore {
			t.Fatalf("round %d returned nothing but asked the device to continue — infinite loop", round)
		}
		if !wantsMore {
			break
		}
	}

	if seen != total {
		t.Errorf("synced %d of %d books; the rest can never reach the device", seen, total)
	}
}

// The browser-facing setup endpoint.
func TestKoboSetupEndpointShape(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")
	f := setupKoboFixture(t)

	signinBody := []byte(`{"email":"kobo-test@example.com","password":"password123"}`)
	signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(signinBody))
	signinReq.Header.Set("Content-Type", "application/json")
	signinResp, err := f.app.Test(signinReq)
	if err != nil {
		t.Fatalf("signin: %v", err)
	}
	var signin struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	decodeJSON(t, signinResp, &signin)
	if signin.Data.AccessToken == "" {
		t.Fatal("signin returned no access token")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/kobo/setup", nil)
	req.Header.Set("Authorization", "Bearer "+signin.Data.AccessToken)
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("get setup: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("get setup = %d, want 200: %s", resp.StatusCode, body)
	}

	var payload struct {
		Status bool `json:"status"`
		Data   struct {
			EndpointURL    string `json:"endpoint_url"`
			IsLocalAddress bool   `json:"is_local_address"`
		} `json:"data"`
	}
	decodeJSON(t, resp, &payload)

	if !payload.Status {
		t.Error("status must be true on success")
	}
	if !bytes.Contains([]byte(payload.Data.EndpointURL), []byte("/kobo/"+f.token)) {
		t.Errorf("endpoint_url = %q, want it to carry the existing token %s", payload.Data.EndpointURL, f.token)
	}
	if payload.Data.IsLocalAddress {
		t.Errorf("is_local_address = true for %q", payload.Data.EndpointURL)
	}
}

// server.url replaces the old SERVER_URL env var.
func TestKoboSetupEndpointUsesConfiguredServerURL(t *testing.T) {
	const configured = "https://books.example.org"

	for _, tc := range []struct {
		name      string
		serverURL string
		wantorig  string
	}{
		{name: "configured wins over detected host", serverURL: configured, wantorig: configured},
		{name: "empty falls back to detected host", serverURL: "", wantorig: "http://example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "test-access-secret")
			t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

			f := setupKoboFixture(t, func(t *testing.T, db *sql.DB, _ koboSeed) {
				t.Helper()
				encoded, err := json.Marshal(tc.serverURL)
				if err != nil {
					t.Fatalf("encode server.url: %v", err)
				}
				if _, err := db.Exec(`
					INSERT INTO app_settings (key, value_json) VALUES ('server.url', ?)
					ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
				`, string(encoded)); err != nil {
					t.Fatalf("seed server.url: %v", err)
				}
			})

			signinBody := []byte(`{"email":"kobo-test@example.com","password":"password123"}`)
			signinReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", bytes.NewReader(signinBody))
			signinReq.Header.Set("Content-Type", "application/json")
			signinResp, err := f.app.Test(signinReq)
			if err != nil {
				t.Fatalf("signin: %v", err)
			}
			var signin struct {
				Data struct {
					AccessToken string `json:"access_token"`
				} `json:"data"`
			}
			decodeJSON(t, signinResp, &signin)
			if signin.Data.AccessToken == "" {
				t.Fatal("signin returned no access token")
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/kobo/setup", nil)
			req.Header.Set("Authorization", "Bearer "+signin.Data.AccessToken)
			resp, err := f.app.Test(req)
			if err != nil {
				t.Fatalf("get setup: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("get setup = %d, want 200: %s", resp.StatusCode, body)
			}

			var payload struct {
				Data struct {
					EndpointURL string `json:"endpoint_url"`
				} `json:"data"`
			}
			decodeJSON(t, resp, &payload)

			want := tc.wantorig + "/kobo/" + f.token
			if payload.Data.EndpointURL != want {
				t.Errorf("endpoint_url = %q, want %q", payload.Data.EndpointURL, want)
			}
		})
	}
}

// The same value drives every OPDS link, and a reader app resolves them to fetch files.
func TestOPDSCatalogUsesConfiguredServerURL(t *testing.T) {
	const configured = "https://books.example.org"

	f := setupKoboFixture(t, func(t *testing.T, db *sql.DB, _ koboSeed) {
		t.Helper()
		if _, err := db.Exec(`
			INSERT INTO app_settings (key, value_json) VALUES ('server.url', '"https://books.example.org"')
			ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
		`); err != nil {
			t.Fatalf("seed server.url: %v", err)
		}
	})

	resp, err := f.app.Test(httptest.NewRequest(http.MethodGet, "/api/opds/v1", nil))
	if err != nil {
		t.Fatalf("get opds: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get opds = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Contains(body, []byte(configured+"/api/opds/v1")) {
		t.Errorf("catalog carries no link on %s:\n%s", configured, body)
	}
	if bytes.Contains(body, []byte("http://example.com")) {
		t.Errorf("catalog still carries the detected host:\n%s", body)
	}
}
