package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readAuditRouteFile(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "routes", name))
	if err != nil {
		src, err = os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
	}
	return string(src)
}

// TestAuditMagicCodeActivateNoRateLimit proves task T1.5: /auth/magic-code/activate
// is mounted without the auth limiter (only /request and /poll get one), so an
// attacker can brute-force the 6-digit magic code with no throttling.
//
// PASSING = bug confirmed: the mount line for /activate carries no limiter.
// A fix must fail this assertion.
func TestAuditMagicCodeActivateNoRateLimit(t *testing.T) {
	src := readAuditRouteFile(t, "authRoute.go")

	found := false
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `"/activate"`) {
			found = true
			if strings.Contains(trimmed, "authLimiter") || strings.Contains(trimmed, "RateLimit") {
				t.Fatalf("unexpected: /activate now has a rate limiter (%q); the audit claim no longer holds", trimmed)
			}
		}
	}
	if !found {
		t.Fatal("could not locate the /activate route mount")
	}
}

// TestAuditAgeRatingRouteMissingLibraryScope proves task T3.2: the write route
// PUT /books/:id/age-rating checks the plain book.edit permission but never
// scopes the book to a library (missing BookLibraryAttr, unlike bookRoutes.go),
// so anyone with book.edit anywhere can set ratings across every library.
//
// PASSING = bug confirmed: the route has RequirePermission but no BookLibraryAttr.
// A fix must fail this assertion.
func TestAuditAgeRatingRouteMissingLibraryScope(t *testing.T) {
	src := readAuditRouteFile(t, "ageRatingRoutes.go")

	var block []string
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, `"/books/:id/age-rating"`) {
			block = append(block, line)
			continue
		}
		if len(block) > 0 {
			block = append(block, line)
			if strings.Contains(line, "controller") {
				break
			}
		}
	}
	if len(block) == 0 {
		t.Fatal("could not locate the /books/:id/age-rating route block")
	}
	joined := strings.Join(block, "\n")
	if !strings.Contains(joined, "RequirePermission") {
		t.Fatal("setup broken: route no longer uses RequirePermission")
	}
	// BUG PROOF: no library scoping middleware on this route.
	if strings.Contains(joined, "BookLibraryAttr") {
		t.Fatalf("unexpected: route now has BookLibraryAttr; the audit claim no longer holds")
	}
}

// TestAuditSmartFilterServiceDoubleV1Prefix proves task T0.2: the frontend
// service calls baseURL("/api/v1") + "/v1/smart-filters" = /api/v1/v1/smart-filters,
// while the backend mounts the group at /smart-filters under /api/v1. Every
// smart-filter request 404s, which is why the whole EPIC-5 feature is dead on
// the frontend.
//
// PASSING = bug confirmed. Fixing the service paths must fail this assertion.
func TestAuditSmartFilterServiceDoubleV1Prefix(t *testing.T) {
	feSrc, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "services", "smartFilterService.ts"))
	if err != nil {
		feSrc, err = os.ReadFile(filepath.Join("..", "..", "..", "web", "src", "services", "smartFilterService.ts"))
		if err != nil {
			t.Fatal(err)
		}
	}
	beSrc := readAuditRouteFile(t, "smartFilterRoutes.go")

	// Backend mount: "/smart-filters" (under the /api/v1 group).
	if !strings.Contains(beSrc, `"/smart-filters"`) {
		t.Fatal("setup broken: backend smart-filter group not found")
	}

	// Every api.get/post/put/delete path in the FE service must start with
	// "/smart-filters"; the shipped code uses "/v1/smart-filters".
	bad := 0
	for _, line := range strings.Split(string(feSrc), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, `api.`) || !strings.Contains(trimmed, `"/`) {
			continue
		}
		idx := strings.Index(trimmed, `"/`)
		if idx < 0 {
			continue
		}
		path := trimmed[idx+1:]
		if end := strings.IndexByte(path, '"'); end >= 0 {
			path = path[:end]
		}
		if strings.HasPrefix(path, "/v1/") {
			bad++
		}
	}
	// BUG FIXED: no request path should double-prefix /v1.
	if bad > 0 {
		t.Fatalf("found %d request paths with double-prefixed /v1 in smartFilterService.ts", bad)
	}
}
