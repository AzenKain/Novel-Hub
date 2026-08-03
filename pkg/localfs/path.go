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


func ResolveBookFilePath(bookID string, rawPath string) string {
	if strings.TrimSpace(rawPath) == "" {
		return rawPath
	}

	normalizedRaw := rawPath
	if filepath.Separator == '/' {
		normalizedRaw = strings.ReplaceAll(rawPath, "\\", "/")
	} else {
		normalizedRaw = strings.ReplaceAll(rawPath, "/", "\\")
	}

	if _, err := os.Stat(normalizedRaw); err == nil {
		return normalizedRaw
	}

	slashNormalized := strings.ReplaceAll(rawPath, "\\", "/")
	parts := strings.Split(slashNormalized, "/")
	filename := parts[len(parts)-1]
	if filename == "" || filename == "." {
		return rawPath
	}

	booksDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "books")
	absBooksDir, err := filepath.Abs(booksDir)
	if err != nil {
		absBooksDir = booksDir
	}

	return filepath.Join(absBooksDir, bookID, filename)
}
