package routes

import (
	"strings"
	"testing"
)

// TestAuditMagicCodeActivationAutoFillsFromURL proves task T1.2: the magic
// code activation page (web/src/pages/auth/ActivateMagicCodePage.tsx) reads
// the 6-digit code straight from the URL query param (?code=) and auto-fills
// it into the activation form. Anyone who obtains an activation link — via
// referrer leak, browser history or a shared screenshot URL — gets the code
// pre-filled, turning the one-time code into a bearer credential.
//
// PASSING = bug confirmed: the code from the query param flows into state
// without any user confirmation keystroke.
func TestAuditMagicCodeActivationAutoFillsFromURL(t *testing.T) {
	src := readRepoFile(t, "web/src/pages/auth/ActivateMagicCodePage.tsx")

	if !strings.Contains(src, `searchParams.get("code")`) {
		t.Fatal("setup broken: page no longer reads code from the URL")
	}
	// The fetched code seeds both the initial state and a useEffect re-seed.
	if !strings.Contains(src, "useState(codeFromUrl)") {
		t.Fatalf("unexpected: initial state no longer comes from the URL code; bug may be fixed")
	}
	if !strings.Contains(src, "setCode(codeFromUrl)") {
		t.Fatalf("unexpected: useEffect no longer re-seeds from the URL code; bug may be fixed")
	}
}
