package comic

import (
	"archive/tar"
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
