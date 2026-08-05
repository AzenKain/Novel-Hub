package main

import (
	"net/http/httptest"
	"testing"
)

// book.offline is a download that lives in the browser, so it must land on exactly the roles
// that already hold book.download — and never on GUEST, whose offline copy would outlive the
// anonymous session that made it and stay readable on a shared device.
func TestOfflinePermissionMatchesDownloadGrants(t *testing.T) {
	_, db, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	var described string
	if err := db.QueryRow(`SELECT description FROM permissions WHERE key = 'book.offline'`).Scan(&described); err != nil {
		t.Fatalf("book.offline is not in the permissions table: %v", err)
	}
	if described == "" {
		t.Error("book.offline has no description, so the admin role editor shows a blank row")
	}

	rows, err := db.Query(`
		SELECT r.name,
		       MAX(CASE WHEN rp.permission_key = 'book.download' THEN 1 ELSE 0 END) AS can_download,
		       MAX(CASE WHEN rp.permission_key = 'book.offline'  THEN 1 ELSE 0 END) AS can_offline
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		GROUP BY r.id, r.name
	`)
	if err != nil {
		t.Fatalf("query grants: %v", err)
	}
	defer func() { _ = rows.Close() }()

	seen := 0
	for rows.Next() {
		var name string
		var canDownload, canOffline int
		if err := rows.Scan(&name, &canDownload, &canOffline); err != nil {
			t.Fatalf("scan: %v", err)
		}
		seen++
		if canDownload != canOffline {
			t.Errorf("%s: book.download=%d but book.offline=%d", name, canDownload, canOffline)
		}
		if name == "GUEST" && canOffline == 1 {
			t.Error("GUEST was granted book.offline")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if seen == 0 {
		t.Fatal("no roles were seeded, so this test proves nothing")
	}
}

// The reader routes carry no book.offline gate: an offline copy is assembled from the same
// bootstrap/chapter/asset responses the reader already serves, so the permission is a client
// affordance layered on book.read, and the routes stay reachable for normal reading.
func TestOfflinePermissionDoesNotBlockReaderRoutes(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/reader/does-not-exist/bootstrap", nil))
	if err != nil {
		t.Fatalf("bootstrap request: %v", err)
	}
	if resp.StatusCode == 403 {
		t.Error("the reader bootstrap route is permission-gated in a way that would break plain reading")
	}
}
