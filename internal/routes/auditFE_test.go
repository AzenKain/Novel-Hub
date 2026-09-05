package routes

import (
	"strings"
	"testing"
)

// TestAuditFE_DiscordFieldMutatesGoogleState proves task T2.3: in OAuthSettings.tsx the Discord Client ID input reads discordClientId but its onChange calls setGoogleClientId — a copy-paste bug that makes the two provider configs interfere.
func TestAuditFE_DiscordFieldMutatesGoogleState(t *testing.T) {
	src := readRepoFile(t, "web/src/pages/admin/OAuthSettings.tsx")
	if !strings.Contains(src, "setGoogleClientId(e.target.value)") {
		t.Fatalf("unexpected: Discord field no longer calls setGoogleClientId; bug may be fixed")
	}
	if !strings.Contains(src, "value={discordClientId}") {
		t.Fatal("setup broken: discordClientId input not found")
	}
}

// TestAuditFE_BangumiNeverImplemented proves task T2.4: the roadmap promised bangumi_id (EPIC 1.2) but no reference exists anywhere in code or schema.
func TestAuditFE_BangumiNeverImplemented(t *testing.T) {
	for _, rel := range []string{
		"db/schema/20_books.sql",
		"db/query/books.sql",
		"internal/models/book.go",
		"internal/services/bookService_enrich.go",
	} {
		src := readRepoFile(t, rel)
		if strings.Contains(src, "bangumi") {
			t.Fatalf("unexpected: %s now references bangumi; audit claim may be fixed", rel)
		}
	}
}

// TestAuditFE_OfflineCacheBuffersWholeFile proves task T4.1: useOfflineBook fetches the entire stream into a Blob before caching — no chunking, no quota.
func TestAuditFE_OfflineCacheBuffersWholeFile(t *testing.T) {
	src := readRepoFile(t, "web/src/hooks/useOfflineBook.ts")
	if !strings.Contains(src, "res.blob()") {
		t.Fatalf("unexpected: useOfflineBook no longer buffers with res.blob(); may be chunked now")
	}
	if strings.Contains(src, "offlineStore.usage") || strings.Contains(src, "getQuota") {
		t.Fatal("unexpected: quota check present; audit claim may be fixed")
	}
}
