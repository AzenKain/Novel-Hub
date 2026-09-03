package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	// Try from the routes package dir outward.
	for _, prefix := range []string{"../..", "../../..", "../../../.."} {
		p := filepath.Join(prefix, rel)
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
	}
	t.Fatalf("could not read %s", rel)
	return ""
}

// TestAuditScopeNewFeaturesHaveNoPermissionKeys proves task T3.1: none of the
// new features (age rating, kids mode, smart filters, magic code, OAuth) has a
// permission key seeded in the RBAC schema, so there is nothing to enforce
// them with.
func TestAuditScopeNewFeaturesHaveNoPermissionKeys(t *testing.T) {
	src := readRepoFile(t, "db/schema/65_permissions_settings.sql") +
		"\n" + readRepoFile(t, "db/schema/91_rbac_restructure.sql")
	// Only scan the permissions insertion block, not the settings seeding
	if parts := strings.Split(src, "INSERT INTO app_settings"); len(parts) > 0 {
		src = parts[0]
	}
	for _, key := range []string{"age.rating", "kids.mode", "smart.filter", "magic.code", "oauth"} {
		if strings.Contains(src, key) {
			t.Fatalf("unexpected: permission key %q now exists; audit claim no longer holds", key)
		}
	}
}

// TestAuditScopeGoFuncBypassesWorker verifies that background tasks in bookController
// no longer spawn unmanaged go func() and instead go through the worker queue.
func TestAuditScopeGoFuncBypassesWorker(t *testing.T) {
	src := readRepoFile(t, "internal/controllers/bookController.go")
	if strings.Contains(src, "go func()") {
		t.Fatalf("unexpected: internal/controllers/bookController.go still spawns unmanaged go func()")
	}
}

// TestAuditScopeGenTokenIsOneLineWrapper proves task T5.2: GenToken is a
// trivial 1-line wrapper around genToken, which AGENTS.md forbids.
func TestAuditScopeGenTokenIsOneLineWrapper(t *testing.T) {
	src := readRepoFile(t, "internal/services/authService.go")
	idx := strings.Index(src, "func (a *authService) GenToken(")
	if idx < 0 {
		t.Fatal("GenToken method not found")
	}
	body := src[idx:]
	end := strings.Index(body, "\n}\n")
	if end < 0 {
		t.Fatal("could not bound GenToken method body")
	}
	body = body[:end]
	lines := strings.Count(body, "\n")
	if lines > 1 {
		t.Fatalf("unexpected: GenToken body is %d lines; wrapper may have been removed", lines)
	}
	if !strings.Contains(body, "return a.genToken(user)") {
		t.Fatal("unexpected: GenToken body changed")
	}
}

// TestAuditScopeSmartFilterRawBind verifies that smart filter controller
// uses validator.ValidateBodyDto instead of raw c.Bind().Body.
func TestAuditScopeSmartFilterRawBind(t *testing.T) {
	src := readRepoFile(t, "internal/controllers/smartFilterController.go")
	rawBinds := strings.Count(src, "c.Bind().Body")
	if rawBinds > 0 {
		t.Fatalf("unexpected: found %d raw c.Bind().Body calls in smartFilterController.go", rawBinds)
	}
}

