package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSystemLogServiceTailFiltersAndRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	content := "{\"level\":\"info\",\"message\":\"started\"}\n{\"level\":\"error\",\"message\":\"failed book\"}\n"
	if err := os.WriteFile(filepath.Join(dir, "novelhub.log"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	service := NewSystemLogService(dir)
	result, err := service.Tail(context.Background(), "novelhub.log", 10, "error", "book")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 1 || result.Lines[0] != "{\"level\":\"error\",\"message\":\"failed book\"}" {
		t.Fatalf("unexpected lines: %#v", result.Lines)
	}
	if _, err := service.Path("../novelhub.log"); err == nil {
		t.Fatal("expected traversal path rejection")
	}
}
