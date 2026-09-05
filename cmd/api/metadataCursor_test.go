package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
)

// Author pagination dead-ended on any name containing a comma: EncodeCursor joined the sort value and the id with ",", DecodeCursor split on the first one, and a decode of anything other than two parts was read as "no cursor" — so page 2 returned page 1 and infinite scroll looped.
func TestAuthorPaginationWalksPastCommaNames(t *testing.T) {
	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib', 'Main')`); err != nil {
		t.Fatalf("seed library: %v", err)
	}

	names := []string{
		"Adams, Douglas", "Bradbury, Ray", "Clarke, Arthur C.",
		"Dick, Philip K.", "Ellison, Harlan", "Frank Herbert",
	}
	for i, name := range names {
		authorID := uuid.Must(uuid.NewV7()).String()
		if _, err := db.Exec(`INSERT INTO authors (id, name) VALUES (?, ?)`, authorID, name); err != nil {
			t.Fatalf("seed author %q: %v", name, err)
		}
		if _, err := db.Exec(`
			INSERT INTO books (id, library_id, title, author_id, status)
			VALUES (?, 'lib', ?, ?, 'published')`,
			uuid.Must(uuid.NewV7()).String(), fmt.Sprintf("Book %d", i), authorID); err != nil {
			t.Fatalf("seed book: %v", err)
		}
	}

	seen := map[string]bool{}
	cursor := ""
	for page := 0; page < len(names)+2; page++ {
		path := "/api/v1/metadata/authors?limit=2"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d = %d: %s", page, resp.StatusCode, body)
		}

		var payload struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
			Pagination struct {
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("page %d decode: %v (%s)", page, err, body)
		}
		for _, item := range payload.Data {
			seen[item.Name] = true
		}
		if payload.Pagination.NextCursor == "" {
			break
		}
		if payload.Pagination.NextCursor == cursor {
			t.Fatalf("page %d returned the same cursor, so the walk cannot advance: %s", page, cursor)
		}
		cursor = payload.Pagination.NextCursor
	}

	for _, name := range names {
		if !seen[name] {
			t.Errorf("author %q was never reachable by paging (saw %d of %d)", name, len(seen), len(names))
		}
	}
}
