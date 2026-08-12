package services

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"novelhub/pkg/jsonx"
)

// TestAuditOAuthOpenRedirect proves task T1.3: the redirect parameter passed to
// BuildOAuthURL is embedded in the OAuth state verbatim, with no host
// validation. The callback later 302-redirects the user's browser to that URL,
// making the login flow an open redirect.
//
// PASSING = bug confirmed: an attacker can plant
// /auth/oauth2/google/login?redirect=https://evil.example/ and the victim's
// browser lands there right after a successful login.
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
		t.Fatalf("BuildOAuthURL rejected an external redirect target: %v (no open redirect to prove)", err)
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
	// BUG PROOF: the external host survived end-to-end — the callback will
	// c.Redirect().To(state.RedirectURL) straight to it.
	if state.RedirectURL != evil {
		t.Fatalf("unexpected: RedirectURL was sanitized to %q; open redirect may have been fixed", state.RedirectURL)
	}
}

func mustExtractState(t *testing.T, authURL string) string {
	t.Helper()
	q := authURL
	if i := strings.IndexByte(q, '?'); i >= 0 {
		q = q[i+1:]
	}
	for _, pair := range strings.Split(q, "&") {
		if strings.HasPrefix(pair, "state=") {
			return strings.TrimPrefix(pair, "state=")
		}
	}
	t.Fatal("no state= param found in authURL")
	return ""
}

// TestAuditOAuthEmailMatchTakeover proves task T1.4: SigninOrRegisterOAuth
// matches accounts by email only — it never checks that the provider and
// oauth2 sub match the existing account. Logging in with a *different* provider
// that happens to share the victim's email returns a session for the victim's
// account.
//
// PASSING = bug confirmed: attacker's GitHub login minted tokens for the
// victim's Google-created account.
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

	// Attacker logs in via GitHub with the same email but a different sub.
	attackerTokens, err := svc.SigninOrRegisterOAuth(ctx, "GITHUB", "victim@example.com", "Attacker", "", "github-sub-attacker")
	// BUG PROOF: no error — the attacker was handed the victim's account.
	if err != nil {
		t.Fatalf("unexpected: GitHub login with victim email was rejected; provider/email_verified check may exist now")
	}

	victim, err := userRepo.GetAuthByEmail(ctx, "victim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if victim.AuthProvider != "GOOGLE" {
		t.Fatalf("unexpected: account provider changed to %q", victim.AuthProvider)
	}

	// Prove the attacker's token IS the victim's account (same subject, same row).
	parsed, err := jwt.Parse(attackerTokens.AccessToken, func(token *jwt.Token) (any, error) {
		return []byte("oauth-test-access-secret"), nil
	})
	if err != nil {
		t.Fatalf("failed to parse attacker token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}
	sub, _ := claims["sub"].(string)
	if sub != victim.ID {
		t.Fatalf("attacker token sub %q != victim account %q — no takeover to prove", sub, victim.ID)
	}
	exp, _ := claims["exp"].(float64)
	if int64(exp) < time.Now().Unix() {
		t.Fatal("attacker token already expired")
	}
	// BUG PROOF: attacker holds a live session as the victim.
	_ = sub
}
