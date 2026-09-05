package comic

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"novelhub/pkg/constants"
)

func TestCBZParserOrdersPagesAndReadsAssets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comic.cbz")
	if err := createZip(path, map[string]string{
		"page10.jpg": "ten",
		"page2.jpg":  "two",
		"notes.txt":  "skip",
	}); err != nil {
		t.Fatal(err)
	}

	parser := NewParser("cbz")
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) != 2 || images[0] != "page2.jpg" || images[1] != "page10.jpg" {
		t.Fatalf("unexpected image order: %+v", images)
	}
	html, err := parser.GetChapterContent(path, "comic")
	if err != nil {
		t.Fatalf("GetChapterContent failed: %v", err)
	}
	if !strings.Contains(html, `src="page2.jpg"`) || !strings.Contains(html, `src="page10.jpg"`) {
		t.Fatalf("unexpected html: %s", html)
	}
	data, err := parser.GetAsset(path, "page2.jpg")
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if string(data) != "two" {
		t.Fatalf("asset = %q, want two", string(data))
	}
}

func TestCBTParserOrdersPagesAndReadsAssets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comic.cbt")
	if err := createTar(path, map[string]string{
		"page10.jpg": "ten",
		"page2.jpg":  "two",
		"notes.txt":  "skip",
	}); err != nil {
		t.Fatal(err)
	}

	parser := NewParser("cbt")
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) != 2 || images[0] != "page2.jpg" || images[1] != "page10.jpg" {
		t.Fatalf("unexpected image order: %+v", images)
	}
	data, err := parser.GetAsset(path, "page10.jpg")
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if string(data) != "ten" {
		t.Fatalf("asset = %q, want ten", string(data))
	}
}

func createZip(path string, files map[string]string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	defer writer.Close()
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func createTar(path string, files map[string]string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := tar.NewWriter(out)
	defer writer.Close()
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func TestArchiveBudgetRejectsBombs(t *testing.T) {
	dir := t.TempDir()

	cbz := filepath.Join(dir, "bomb.cbz")
	if err := createZipWithDeclaredSize(cbz, "page1.jpg", constants.MaxArchiveUncompressedBytes+1, 3); err != nil {
		t.Fatal(err)
	}
	cb7 := filepath.Join(dir, "bomb.cb7")
	if err := createSevenZipEntries(t, cb7, constants.MaxArchiveEntries+1); err != nil {
		t.Skipf("cannot build .cb7 fixture: %v", err)
	}

	for _, tc := range []struct{ format, path, asset string }{
		{"cbz", cbz, "page1.jpg"},
		{"cb7", cb7, "page1.jpg"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			parser := NewParser(tc.format)
			if _, err := parser.ListImages(tc.path); err == nil {
				t.Errorf("ListImages accepted an archive outside the budget")
			}
			if _, err := parser.GetAsset(tc.path, tc.asset); err == nil {
				t.Errorf("GetAsset accepted an archive outside the budget")
			}
		})
	}
}

func TestArchiveBudgetAccumulatesEntrySizes(t *testing.T) {
	entrySize := int64(constants.MaxArchiveAssetSize)
	overBudget := int(constants.MaxArchiveUncompressedBytes/entrySize) + 1

	entries := 0
	var total int64
	var err error
	for i := 0; i < overBudget && err == nil; i++ {
		err = archiveBudgetAdd(&entries, &total, entrySize)
	}
	if err == nil {
		t.Fatalf("%d entries of %d bytes stayed inside the %d-byte budget",
			overBudget, entrySize, constants.MaxArchiveUncompressedBytes)
	}

	entries, total = 0, 0
	for range overBudget {
		if err := archiveBudgetAdd(&entries, &total, 0); err != nil {
			t.Fatalf("size 0 should only count entries, got %v", err)
		}
	}
	if total != 0 {
		t.Fatalf("size 0 accumulated %d bytes; a format passing 0 has no size budget", total)
	}
}

func createZipWithDeclaredSize(path, name string, declaredSize int64, rawBytes int) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	w, err := writer.CreateRaw(&zip.FileHeader{
		Name:               name,
		Method:             zip.Store,
		CompressedSize64:   uint64(rawBytes),
		UncompressedSize64: uint64(declaredSize),
	})
	if err != nil {
		return err
	}
	if _, err := w.Write(make([]byte, rawBytes)); err != nil {
		return err
	}
	return writer.Close()
}

func createSevenZipEntries(t *testing.T, path string, count int) error {
	t.Helper()
	if _, err := exec.LookPath("7z"); err != nil {
		return err
	}
	staging := t.TempDir()
	for i := 1; i <= count; i++ {
		member := filepath.Join(staging, fmt.Sprintf("page%d.jpg", i))
		if err := os.WriteFile(member, []byte("x"), 0600); err != nil {
			return err
		}
	}
	out, err := exec.Command("7z", "a", "-bso0", "-bsp0", path, staging+string(os.PathSeparator)+".").CombinedOutput()
	if err != nil {
		return fmt.Errorf("7z: %v: %s", err, out)
	}
	return nil
}

func TestRARParserOrdersPagesAndReadsAssets(t *testing.T) {
	realCBR := realCBRPath(t)
	parser := NewParser("rar")
	images, err := parser.ListImages(realCBR)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) == 0 {
		t.Fatal("expected at least one page in RAR archive")
	}
	data, err := parser.GetAsset(realCBR, images[0])
	if err != nil {
		t.Fatalf("GetAsset failed for first page: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("extracted page asset data is empty")
	}
}

func TestSevenZipParserOrdersPagesAndReadsAssets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comic.7z")
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "page2.jpg"), []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "page10.jpg"), []byte("ten"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "notes.txt"), []byte("skip"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := exec.LookPath("7z"); err != nil {
		t.Skip("7z command line not found, skipping 7z parser test")
	}
	out, err := exec.Command("7z", "a", "-bso0", "-bsp0", path, staging+string(os.PathSeparator)+".").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create 7z fixture: %v: %s", err, out)
	}

	parser := NewParser("7z")
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) != 2 || images[0] != "page2.jpg" || images[1] != "page10.jpg" {
		t.Fatalf("unexpected image order: %+v", images)
	}
	data, err := parser.GetAsset(path, "page10.jpg")
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if string(data) != "ten" {
		t.Fatalf("asset = %q, want ten", string(data))
	}
}
