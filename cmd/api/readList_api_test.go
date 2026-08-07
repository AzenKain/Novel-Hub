package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Plain USER, not seedAdmin: ADMIN holds every permission, so an admin round trip would pass even
// if the route group were gated on the wrong key. USER holds book.collection and nothing extra.
func seedReadListUser(t *testing.T, db *sql.DB, email string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.Must(uuid.NewV7()).String()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, full_name, password_hash, auth_provider, token_version)
		VALUES (?, ?, 'Reader', ?, 'LOCAL', 1)
	`, userID, email, string(hash)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE name = 'USER'
	`, userID); err != nil {
		t.Fatal(err)
	}
	return userID
}

func readListCall(t *testing.T, app *fiber.App, method, path, token, body string) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func readListBookOrder(t *testing.T, app *fiber.App, token, listID string) []string {
	t.Helper()
	status, body := readListCall(t, app, http.MethodGet, "/api/v1/read-lists/"+listID+"/books", token, "")
	if status != http.StatusOK {
		t.Fatalf("GET books returned %d: %s", status, body)
	}
	var decoded struct {
		Data []struct {
			Position int64 `json:"position"`
			Book     struct {
				ID string `json:"id"`
			} `json:"book"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode books: %v (%s)", err, body)
	}
	ids := make([]string, 0, len(decoded.Data))
	for i, entry := range decoded.Data {
		if entry.Position != int64(i) {
			t.Errorf("entry %d reports position %d", i, entry.Position)
		}
		ids = append(ids, entry.Book.ID)
	}
	return ids
}

func seedReadListBooks(t *testing.T, db *sql.DB, count int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES (?, 'lib', ?, 'active')`,
			fmt.Sprintf("bk-%d", i), fmt.Sprintf("Book %d", i)); err != nil {
			t.Fatal(err)
		}
	}
}

// The whole promise of the feature in one request chain: create a list, stack issues into it, drag
// them into story order, and have /next walk that order back out.
func TestReadListRoundTripOverHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	seedReadListBooks(t, db, 3)
	seedReadListUser(t, db, "alice@example.com")
	token := signinToken(t, app, "alice@example.com")

	status, body := readListCall(t, app, http.MethodPost, "/api/v1/read-lists/", token, `{"name":"Civil War","description":"Reading order"}`)
	if status != http.StatusCreated {
		t.Fatalf("create returned %d: %s", status, body)
	}
	var created struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode create: %v (%s)", err, body)
	}
	listID := created.Data.ID
	if listID == "" || created.Data.Name != "Civil War" {
		t.Fatalf("create returned %+v", created.Data)
	}

	for i := 0; i < 3; i++ {
		status, body := readListCall(t, app, http.MethodPost, "/api/v1/read-lists/"+listID+"/books", token,
			fmt.Sprintf(`{"book_id":"bk-%d"}`, i))
		if status != http.StatusOK {
			t.Fatalf("add bk-%d returned %d: %s", i, status, body)
		}
	}
	if got := readListBookOrder(t, app, token, listID); fmt.Sprint(got) != fmt.Sprint([]string{"bk-0", "bk-1", "bk-2"}) {
		t.Fatalf("order after appends = %v", got)
	}

	status, body = readListCall(t, app, http.MethodPut, "/api/v1/read-lists/"+listID+"/order", token,
		`{"book_ids":["bk-2","bk-0","bk-1"]}`)
	if status != http.StatusOK {
		t.Fatalf("reorder returned %d: %s", status, body)
	}
	if got := readListBookOrder(t, app, token, listID); fmt.Sprint(got) != fmt.Sprint([]string{"bk-2", "bk-0", "bk-1"}) {
		t.Fatalf("order after reorder = %v", got)
	}

	walked := make([]string, 0, 3)
	after := ""
	for step := 0; step < 5; step++ {
		path := "/api/v1/read-lists/" + listID + "/next"
		if after != "" {
			path += "?after=" + after
		}
		status, body := readListCall(t, app, http.MethodGet, path, token, "")
		if status != http.StatusOK {
			t.Fatalf("next returned %d: %s", status, body)
		}
		var next struct {
			Data struct {
				HasNext bool `json:"has_next"`
				Book    struct {
					ID string `json:"id"`
				} `json:"book"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &next); err != nil {
			t.Fatalf("decode next: %v (%s)", err, body)
		}
		if !next.Data.HasNext {
			break
		}
		walked = append(walked, next.Data.Book.ID)
		after = next.Data.Book.ID
	}
	if want := []string{"bk-2", "bk-0", "bk-1"}; fmt.Sprint(walked) != fmt.Sprint(want) {
		t.Errorf("/next walked %v, want %v — the reorder did not reach the reader", walked, want)
	}

	status, body = readListCall(t, app, http.MethodDelete, "/api/v1/read-lists/"+listID+"/books/bk-0", token, "")
	if status != http.StatusOK {
		t.Fatalf("remove returned %d: %s", status, body)
	}
	if got := readListBookOrder(t, app, token, listID); fmt.Sprint(got) != fmt.Sprint([]string{"bk-2", "bk-1"}) {
		t.Errorf("order after remove = %v, want [bk-2 bk-1]", got)
	}

	status, body = readListCall(t, app, http.MethodGet, "/api/v1/read-lists/", token, "")
	if status != http.StatusOK {
		t.Fatalf("list returned %d: %s", status, body)
	}
	var page struct {
		Data []struct {
			ID        string `json:"id"`
			BookCount int64  `json:"book_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("decode page: %v (%s)", err, body)
	}
	if len(page.Data) != 1 || page.Data[0].ID != listID || page.Data[0].BookCount != 2 {
		t.Errorf("list page = %+v, want one entry holding 2 books", page.Data)
	}

	if status, body := readListCall(t, app, http.MethodDelete, "/api/v1/read-lists/"+listID, token, ""); status != http.StatusOK {
		t.Fatalf("delete returned %d: %s", status, body)
	}
	if status, _ := readListCall(t, app, http.MethodGet, "/api/v1/read-lists/"+listID, token, ""); status == http.StatusOK {
		t.Error("the deleted list is still readable")
	}
}

// Ownership is enforced in the service, so it has to survive the trip through the router: bob is a
// legitimate signed-in user holding book.collection, and that must still not open alice's list.
func TestReadListIsPrivateToItsOwnerOverHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	seedReadListBooks(t, db, 2)
	seedReadListUser(t, db, "alice@example.com")
	seedReadListUser(t, db, "bob@example.com")
	alice := signinToken(t, app, "alice@example.com")
	bob := signinToken(t, app, "bob@example.com")

	status, body := readListCall(t, app, http.MethodPost, "/api/v1/read-lists/", alice, `{"name":"Private"}`)
	if status != http.StatusCreated {
		t.Fatalf("create returned %d: %s", status, body)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatal(err)
	}
	listID := created.Data.ID
	if status, body := readListCall(t, app, http.MethodPost, "/api/v1/read-lists/"+listID+"/books", alice, `{"book_id":"bk-0"}`); status != http.StatusOK {
		t.Fatalf("alice could not fill her own list: %d %s", status, body)
	}

	for _, probe := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/read-lists/" + listID, ""},
		{http.MethodGet, "/api/v1/read-lists/" + listID + "/books", ""},
		{http.MethodGet, "/api/v1/read-lists/" + listID + "/next", ""},
		{http.MethodPut, "/api/v1/read-lists/" + listID, `{"name":"Stolen"}`},
		{http.MethodPost, "/api/v1/read-lists/" + listID + "/books", `{"book_id":"bk-1"}`},
		{http.MethodDelete, "/api/v1/read-lists/" + listID + "/books/bk-0", ""},
		{http.MethodPut, "/api/v1/read-lists/" + listID + "/order", `{"book_ids":["bk-0"]}`},
		{http.MethodDelete, "/api/v1/read-lists/" + listID, ""},
	} {
		status, body := readListCall(t, app, probe.method, probe.path, bob, probe.body)
		if status == http.StatusOK || status == http.StatusCreated {
			t.Errorf("%s %s succeeded for another user: %s", probe.method, probe.path, body)
		}
	}

	if got := readListBookOrder(t, app, alice, listID); fmt.Sprint(got) != fmt.Sprint([]string{"bk-0"}) {
		t.Errorf("alice's list came back as %v after bob's attempts", got)
	}
	if status, _ := readListCall(t, app, http.MethodGet, "/api/v1/read-lists/"+listID, "", ""); status != http.StatusUnauthorized {
		t.Errorf("an unauthenticated request got %d, want 401", status)
	}
}

// The import is the reason .cbl support exists, and it only counts if it works as an upload: a
// multipart file, matched against the seeded library, coming back with the gap reported.
func TestImportCBLOverHTTP(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret")

	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatal(err)
	}
	seedSeries(t, db, "lib", "s-cw", "Civil War", [][3]string{{"bk-cw1", "Civil War 1", "1"}, {"bk-cw7", "Civil War 7", "7"}})
	seedReadListUser(t, db, "alice@example.com")
	token := signinToken(t, app, "alice@example.com")

	var payload bytes.Buffer
	form := multipart.NewWriter(&payload)
	part, err := form.CreateFormFile("file", "MV2 - Civil War.cbl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(`<?xml version="1.0"?>
<ReadingList><Name>MV2 - Civil War</Name><Books>
  <Book Series="Civil War" Number="01" Volume="2006" />
  <Book Series="Wolverine" Number="42" />
  <Book Series="civil war" Number="7" />
</Books></ReadingList>`)); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/read-lists/import", &payload)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("import returned %d: %s", resp.StatusCode, raw)
	}
	var decoded struct {
		Data struct {
			ReadList struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				BookCount int64  `json:"book_count"`
			} `json:"read_list"`
			Total     int `json:"total"`
			Matched   int `json:"matched"`
			Unmatched []struct {
				Series string `json:"series"`
				Number string `json:"number"`
			} `json:"unmatched"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode import: %v (%s)", err, raw)
	}
	if decoded.Data.ReadList.Name != "MV2 - Civil War" {
		t.Errorf("name = %q, want the name inside the file", decoded.Data.ReadList.Name)
	}
	if decoded.Data.Total != 3 || decoded.Data.Matched != 2 || decoded.Data.ReadList.BookCount != 2 {
		t.Errorf("total/matched/count = %d/%d/%d, want 3/2/2", decoded.Data.Total, decoded.Data.Matched, decoded.Data.ReadList.BookCount)
	}
	if len(decoded.Data.Unmatched) != 1 || decoded.Data.Unmatched[0].Series != "Wolverine" {
		t.Errorf("unmatched = %+v, want just Wolverine", decoded.Data.Unmatched)
	}
	if got := readListBookOrder(t, app, token, decoded.Data.ReadList.ID); fmt.Sprint(got) != fmt.Sprint([]string{"bk-cw1", "bk-cw7"}) {
		t.Errorf("imported order = %v, want [bk-cw1 bk-cw7]", got)
	}

	status, body := readListCall(t, app, http.MethodPost, "/api/v1/read-lists/import", token, "")
	if status != http.StatusBadRequest {
		t.Errorf("an import with no file returned %d: %s", status, body)
	}
}
