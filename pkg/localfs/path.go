package localfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)


func SafeJoin(base string, parts ...string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	joined := filepath.Join(append([]string{absBase}, parts...)...)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute joined path: %w", err)
	}

	rel, err := filepath.Rel(absBase, absJoined)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("path traversal attempt detected")
	}

	return absJoined, nil
}

func IsValidFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
