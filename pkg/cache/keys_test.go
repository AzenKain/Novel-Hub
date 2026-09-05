package cache

import (
	"testing"
	"time"
)

// A nil *time.Time used to panic here: it matched the fmt.Stringer case, and time.Time.String has a value receiver, so calling it through a nil pointer dereferences nothing.
func TestBuildKeyHandlesNilPointers(t *testing.T) {
	var (
		nilTime   *time.Time
		nilString *string
		untyped   any
	)

	for _, tc := range []struct {
		name string
		part any
	}{
		{"nil *time.Time", nilTime},
		{"nil *string", nilString},
		{"untyped nil", untyped},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("BuildKey panicked on %s: %v", tc.name, r)
				}
			}()
			if got := BuildKey("book_ids", tc.part, "id", int64(24)); got == "" {
				t.Error("BuildKey returned an empty key")
			}
		})
	}
}

// Cursors are keyed by value, not by address: two equal timestamps must hit the same cache entry and two different ones must not collide.
func TestBuildKeyIsValueStableForCursors(t *testing.T) {
	first := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	same := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	later := time.Date(2026, 8, 5, 11, 0, 0, 0, time.UTC)

	if BuildKey("k", &first) != BuildKey("k", &same) {
		t.Error("equal timestamps produced different keys, so every page misses the cache")
	}
	if BuildKey("k", &first) == BuildKey("k", &later) {
		t.Error("different timestamps collided, so one page serves another page's rows")
	}
	if BuildKey("k", &first) == BuildKey("k", (*time.Time)(nil)) {
		t.Error("a real cursor collided with the nil cursor, so page 2 could serve page 1")
	}
}
