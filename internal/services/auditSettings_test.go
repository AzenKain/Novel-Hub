package services

import (
	"strings"
	"testing"
)

// The whole point of encrypting smtp.password is that it does not sit in plaintext somewhere every admin can read.
func TestSettingsAuditLabelNeverCarriesTheSMTPPassword(t *testing.T) {
	const password = "sup3r-secret-mail-pw"

	label := SettingsAuditLabel(map[string]any{
		"smtp.password": password,
		"smtp.host":     "mail.example.com",
	})

	if strings.Contains(label, password) {
		t.Fatalf("the SMTP password reached the audit label: %q", label)
	}
	if !strings.Contains(label, "smtp.password") {
		t.Errorf("the label does not record that the password was changed at all: %q", label)
	}
	if !strings.Contains(label, "mail.example.com") {
		t.Errorf("a non-secret value was withheld, which makes the trail useless: %q", label)
	}
}

func TestSettingsAuditLabelRecordsValuesInAStableOrder(t *testing.T) {
	values := map[string]any{
		"auth.registration_enabled": false,
		"limits.rate_limit_auth":    20,
		"site.title":                "My Library",
	}

	first := SettingsAuditLabel(values)
	for range 5 {
		if again := SettingsAuditLabel(values); again != first {
			t.Fatalf("map iteration order leaked into the label:\n%q\n%q", first, again)
		}
	}

	for _, want := range []string{
		"auth.registration_enabled = false",
		"limits.rate_limit_auth = 20",
		"site.title = My Library",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("label is missing %q: %s", want, first)
		}
	}
}

// A long value must not be able to bloat every row of the table.
func TestSettingsAuditLabelTruncatesLongValues(t *testing.T) {
	label := SettingsAuditLabel(map[string]any{
		"site.description": strings.Repeat("x", 500),
	})
	if len(label) > auditValueMaxLen+80 {
		t.Fatalf("a 500-character setting produced a %d-character label", len(label))
	}
	if !strings.HasSuffix(label, "…") {
		t.Errorf("truncation is not marked, so the reader cannot tell: %q", label)
	}
}

func TestSecretSettingKeyCoversOnlyWhatIsEncrypted(t *testing.T) {
	if !secretSettingKey("smtp.password") {
		t.Error("smtp.password is encrypted at rest but not treated as secret")
	}
	for _, key := range []string{"smtp.host", "smtp.username", "site.title", "auth.login_required"} {
		if secretSettingKey(key) {
			t.Errorf("%s is readable via GET /settings, so hiding it from the audit trail gains nothing", key)
		}
	}
}
