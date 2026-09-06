package localfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"novelhub/pkg/config"
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

func isSubpath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func ResolveBookFilePath(bookID string, rawPath string) string {
	if strings.TrimSpace(rawPath) == "" {
		return rawPath
	}

	dataDir := config.GetConfigWithDefault("DATA_DIR", "./data")
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		absDataDir = dataDir
	}
	booksDir := filepath.Join(dataDir, "books")
	absBooksDir, err := filepath.Abs(booksDir)
	if err != nil {
		absBooksDir = booksDir
	}
	absTempDir, err := filepath.Abs(os.TempDir())
	if err != nil {
		absTempDir = os.TempDir()
	}

	normalizedRaw := rawPath
	if filepath.Separator == '/' {
		normalizedRaw = strings.ReplaceAll(rawPath, "\\", "/")
	} else {
		normalizedRaw = strings.ReplaceAll(rawPath, "/", "\\")
	}

	if absNormalized, err := filepath.Abs(normalizedRaw); err == nil {
		if isSubpath(absDataDir, absNormalized) || isSubpath(absTempDir, absNormalized) {
			if _, err := os.Stat(absNormalized); err == nil {
				return absNormalized
			}
		}
	}

	slashNormalized := strings.ReplaceAll(rawPath, "\\", "/")
	filename := filepath.Base(slashNormalized)
	if filename == "" || filename == "." || filename == ".." || filename == "/" || filename == "\\" {
		return ""
	}

	safePath, err := SafeJoin(absBooksDir, bookID, filename)
	if err != nil {
		return ""
	}

	return safePath
}
