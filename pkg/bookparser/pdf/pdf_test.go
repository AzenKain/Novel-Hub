package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPDFEmbeddedImageAssets(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0xff, 0xd9}
	data := append([]byte("%PDF-1.7\nstream\n"), jpeg...)
	data = append(data, []byte("\nendstream\n%%EOF")...)

	path := filepath.Join(t.TempDir(), "fixture.pdf")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	parser := NewParser()
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages returned error: %v", err)
	}
	if len(images) != 1 || images[0] != "images/embedded-0001.jpg" {
		t.Fatalf("images = %#v, want embedded jpeg", images)
	}

	asset, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset returned error: %v", err)
	}
	if !bytes.Equal(asset, jpeg) {
		t.Fatalf("asset = %x, want %x", asset, jpeg)
	}
}
