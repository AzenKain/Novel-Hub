package services

import (
	"context"
	"encoding/base64"
	"net/url"
	"testing"

	"novelhub/pkg/jsonx"
)

// TestAuditOAuthOpenRedirect verifies that external redirect URLs are sanitized to prevent open redirect.
func TestAuditOAuthOpenRedirect(t *testing.T) {
	svc, settings, _, _ := newOAuthTestAuthService(t)
	ctx := context.Background()

	_, err := settings.UpdateSettings(ctx, map[string]any{
		"auth.registration_enabled":  true,
		"oauth.google.enabled":       true,
		"oauth.google.client_id":     "audit-client-id",
		"oauth.google.client_secret": "audit-client-secret",
		"oauth.google.redirect_uri":  "http://localhost:8080/api/v1/auth/oauth2/google/callback",
	})
	if err != nil {
		t.Fatal(err)
	}

	evil := "https://evil.example.com/steal"
	authURL, stateUUID, err := svc.BuildOAuthURL(ctx, "google", evil)
	if err != nil {
		t.Fatalf("BuildOAuthURL failed: %v", err)
	}
	if authURL == "" || stateUUID == "" {
		t.Fatal("expected authURL and stateUUID")
	}

	// Decode the state back out and check what redirect the callback will use.
	decoded, err := base64.URLEncoding.DecodeString(mustExtractState(t, authURL))
	if err != nil {
		t.Fatal(err)
	}
	var state OAuthState
	if err := jsonx.Unmarshal(decoded, &state); err != nil {
		t.Fatal(err)
	}
	// SECURITY CHECK: external redirect host must be sanitized to "/"
	if state.RedirectURL != "/" {
		t.Fatalf("expected RedirectURL to be sanitized to '/', got %q", state.RedirectURL)
	}
}

func mustExtractState(t *testing.T, authURL string) string {
	t.Helper()
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	st := parsed.Query().Get("state")
	if st == "" {
		t.Fatal("no state= param found in authURL")
	}
	return st
}

// TestAuditOAuthEmailMatchTakeover verifies that an account registered with one provider
// cannot be taken over by signing in with a different provider using the same email address.
func TestAuditOAuthEmailMatchTakeover(t *testing.T) {
	svc, settings, userRepo, _ := newOAuthTestAuthService(t)
	ctx := context.Background()

	_, err := settings.UpdateSettings(ctx, map[string]any{
		"auth.registration_enabled": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Victim registers via Google.
	victimTokens, err := svc.SigninOrRegisterOAuth(ctx, "GOOGLE", "victim@example.com", "Victim", "", "google-sub-victim")
	if err != nil {
		t.Fatalf("victim Google registration failed: %v", err)
	}
	if victimTokens.AccessToken == "" {
		t.Fatal("expected victim tokens")
	}

	// Attacker attempts to log in via GitHub with the same email.
	attackerTokens, err := svc.SigninOrRegisterOAuth(ctx, "GITHUB", "victim@example.com", "Attacker", "", "github-sub-attacker")
	// SECURITY CHECK: Must fail with error to prevent account takeover.
	if err == nil || attackerTokens != nil {
		t.Fatalf("expected GitHub login with victim email to be rejected; takeover was possible")
	}

	victim, err := userRepo.GetAuthByEmail(ctx, "victim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if victim.AuthProvider != "GOOGLE" {
		t.Fatalf("unexpected: account provider changed to %q", victim.AuthProvider)
	}
}
