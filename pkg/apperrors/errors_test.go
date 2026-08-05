package apperrors

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

// Some repositories return sql.ErrNoRows raw, others wrap it; callers must accept both.
func TestIsNotFoundAcceptsBothSpellings(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"raw sql.ErrNoRows from featureRepository", sql.ErrNoRows, true},
		{"wrapped ErrNotFound from a service", New(ErrNotFound, "job not found"), true},
		{"sql.ErrNoRows behind fmt.Errorf", fmt.Errorf("load progress: %w", sql.ErrNoRows), true},
		{"ErrNotFound behind fmt.Errorf", fmt.Errorf("load job: %w", ErrNotFound), true},
		{"bare ErrNotFound sentinel", ErrNotFound, true},
		{"nil", nil, false},
		{"a real failure must not read as absent", errors.New("database is locked"), false},
		{"another AppError kind", New(ErrForbidden, "nope"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
