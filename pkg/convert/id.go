package convert

import (
	"fmt"

	"github.com/google/uuid"
)

// ParseID validates a UUID identifier coming from an untrusted source (path
// param, JWT subject) so malformed values are rejected before reaching SQL.
func ParseID(value string) (string, error) {
	if err := uuid.Validate(value); err != nil {
		return "", fmt.Errorf("invalid ID: %s", value)
	}
	return value, nil
}
