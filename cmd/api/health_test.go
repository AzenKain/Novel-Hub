package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// /health used to return a hardcoded true, staying green with the DB gone.
func TestHealthReportsDatabaseFailure(t *testing.T) {
	app, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("healthy instance returned %d: %s", resp.StatusCode, body)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("closed database still reported healthy: %d %s", resp.StatusCode, body)
	}
}
