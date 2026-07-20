package bookparser

import (
	"bytes"
	"testing"
)

func TestExtractEmbeddedImageAssets(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0xff, 0xd9}
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82}
	data := append([]byte("prefix"), jpeg...)
	data = append(data, []byte("middle")...)
	data = append(data, png...)

	assets := ExtractEmbeddedImageAssets(data)
	if len(assets) != 2 {
		t.Fatalf("assets = %d, want 2", len(assets))
	}
	if assets[0].Name != "images/embedded-0001.jpg" || !bytes.Equal(assets[0].Data, jpeg) {
		t.Fatalf("unexpected first asset: %#v", assets[0])
	}
	if assets[1].Name != "images/embedded-0002.png" || !bytes.Equal(assets[1].Data, png) {
		t.Fatalf("unexpected second asset: %#v", assets[1])
	}

	found, err := FindEmbeddedImageAsset(assets, "embedded-0002.png?cache=1")
	if err != nil {
		t.Fatalf("FindEmbeddedImageAsset returned error: %v", err)
	}
	if !bytes.Equal(found, png) {
		t.Fatalf("found asset = %x, want %x", found, png)
	}
}
