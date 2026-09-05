package repositories

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The author:name / tag:name / series:name / publisher:name / language:name cache entries are never invalidated.
func TestMetadataNameEntitiesHaveNoRenameOrDelete(t *testing.T) {
	entities := []string{"Author", "Tag", "Series", "Publisher", "Language"}
	pattern := regexp.MustCompile(`-- name: (Update|Delete|Rename)(` + strings.Join(entities, "|") + `)\b`)

	dir := filepath.Join("..", "..", "db", "query")
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range pattern.FindAllString(string(body), -1) {
			t.Errorf("db/query/%s declares %q; the *:name cache entries written in bookMetadataRepository.go are never invalidated, so this query needs a matching cache.Del before it ships", f.Name(), strings.TrimPrefix(m, "-- name: "))
		}
	}
}
