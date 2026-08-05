package convert

import (
	"testing"
	"time"
)

// DecodeCursor split on the first comma, so any sort value containing one produced fewer or more
// than two parts and the caller treated the cursor as absent — serving page 1 again. Author names
// are the live case: "Surname, Given" is the standard dc:creator form, and the EPUB parser joins
// multiple creators with ", ".
func TestCursorSurvivesCommasInTheSortValue(t *testing.T) {
	const id = "01920000-0000-7000-8000-000000000001"

	for _, name := range []string{
		"Tolkien",
		"Herbert, Frank",
		"Ballantine, Del Rey",
		"Clarke, Arthur C., Jr.",
		",leading",
		"trailing,",
	} {
		t.Run(name, func(t *testing.T) {
			parts := DecodeCursor(EncodeCursor(name, id))
			if len(parts) != 2 {
				t.Fatalf("decoded %d parts, want 2 — the caller reads this as 'no cursor' and repeats page 1", len(parts))
			}
			if parts[0] != name {
				t.Errorf("sort value = %q, want %q", parts[0], name)
			}
			if parts[1] != id {
				t.Errorf("id = %q, want %q", parts[1], id)
			}
		})
	}
}

func TestCursorRoundTripsTimeAndRejectsGarbage(t *testing.T) {
	const id = "01920000-0000-7000-8000-000000000002"
	stamp := time.Date(2026, 8, 5, 12, 34, 56, 789, time.UTC)

	parts := DecodeCursor(EncodeCursor(stamp, id))
	if len(parts) != 2 {
		t.Fatalf("time cursor decoded %d parts, want 2", len(parts))
	}
	if parts[0] != stamp.Format(time.RFC3339Nano) {
		t.Errorf("time = %q, want %q", parts[0], stamp.Format(time.RFC3339Nano))
	}
	if parts[1] != id {
		t.Errorf("id = %q, want %q", parts[1], id)
	}

	for _, bad := range []string{"", "not-base64!!", "bm8tY29tbWEtaGVyZQ=="} {
		if got := DecodeCursor(bad); got != nil {
			t.Errorf("DecodeCursor(%q) = %v, want nil", bad, got)
		}
	}
}
