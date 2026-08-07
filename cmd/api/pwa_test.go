package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// A cached sw.js serves the old build forever; dist has one MaxAge, so this pins the exception.
func TestServiceWorkerAndManifestAreNotCached(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	for _, path := range []string{"/sw.js", "/manifest.webmanifest"} {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s returned %d; run `cd web && bun run build` first", path, resp.StatusCode)
		}
		cacheControl := resp.Header.Get("Cache-Control")
		if !strings.Contains(cacheControl, "no-store") {
			t.Errorf("%s Cache-Control = %q, want no-store; a cached worker freezes the old build", path, cacheControl)
		}
	}
}

// public/ files have no content hash, so a year-long max-age hides translation fixes forever.
func TestRevisionlessAssetsUseAShortMaxAge(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	for _, path := range []string{
		"/locales/en.json",
		"/locales/vi.json",
		"/locales/ja.json",
		"/locales/ko.json",
		"/locales/zh-CN.json",
		"/locales/zh-TW.json",
		"/locales/es.json",
		"/locales/fr.json",
		"/locales/de.json",
		"/locales/pt.json",
		"/locales/ru.json",
		"/locales/ar.json",
		"/locales/hi.json",
		"/locales/id.json",
		"/locales/th.json",
		"/locales/it.json",
		"/favicon.ico",
		"/logo.svg",
		"/pwa-192x192.png",
	} {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s returned %d", path, resp.StatusCode)
		}
		if got := resp.Header.Get("Cache-Control"); strings.Contains(got, "31536000") {
			t.Errorf("%s is served with a one-year max-age (%q) but carries no content hash", path, got)
		}
	}
}

func TestHashedAssetsKeepTheLongMaxAge(t *testing.T) {
	app, _, err := setupTestAppWithDB(t)
	if err != nil {
		t.Fatalf("failed to setup app: %v", err)
	}

	index, err := os.ReadFile("dist/index.html")
	if err != nil {
		t.Skipf("no built frontend: %v", err)
	}
	match := regexp.MustCompile(`/assets/[^"']+\.js`).Find(index)
	if match == nil {
		t.Fatal("index.html references no hashed asset")
	}

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, string(match), nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s returned %d", match, resp.StatusCode)
	}
	if got := resp.Header.Get("Cache-Control"); !strings.Contains(got, "31536000") {
		t.Errorf("hashed asset %s lost its long max-age: %q", match, got)
	}
}

// Cookie auth with no per-user cache key: a cached /api response would leak across users.
func TestServiceWorkerNeverCachesTheAPI(t *testing.T) {
	worker, err := os.ReadFile("dist/sw.js")
	if err != nil {
		t.Skipf("no built service worker: %v", err)
	}

	for _, route := range regexp.MustCompile(`registerRoute\([^,]+`).FindAllString(string(worker), -1) {
		if strings.Contains(route, "NavigationRoute") {
			continue
		}
		if strings.Contains(route, "/api") {
			t.Errorf("a runtime cache route matches the API: %s", route)
		}
	}
}
