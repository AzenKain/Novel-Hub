package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRemoveAllWithRetry(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "busy", err: syscall.EBUSY},
		{name: "not empty", err: syscall.ENOTEMPTY},
	} {
		t.Run(tc.name, func(t *testing.T) {
			booksDir := t.TempDir()
			bookID := "019f92cb-c0c2-7bc8-8f3b-5f91d796bbcd"
			bookDir := filepath.Join(booksDir, bookID)
			if err := os.Mkdir(bookDir, 0750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bookDir, "book.epub"), []byte("book"), 0600); err != nil {
				t.Fatal(err)
			}

			root, err := os.OpenRoot(booksDir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			attempts := 0
			err = removeAllWithRetry(context.Background(), func() error {
				attempts++
				if attempts == 1 {
					return fmt.Errorf("transient remove error: %w", tc.err)
				}
				return root.RemoveAll(bookID)
			})
			if err != nil {
				t.Fatal(err)
			}
			if attempts != 2 {
				t.Fatalf("got %d removal attempts, want 2", attempts)
			}
			if _, err := os.Stat(bookDir); !os.IsNotExist(err) {
				t.Fatalf("book directory still exists: %v", err)
			}
		})
	}
}
