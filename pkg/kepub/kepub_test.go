package kepub

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"

	"novelhub/pkg/constants"
)

func TestInjectKoboSpans(t *testing.T) {
	inputHTML := `<html><body><p>Hello world. Welcome to NovelHub!</p><div>Another paragraph here.</div></body></html>`
	output := InjectKoboSpans(inputHTML)

	if !strings.Contains(output, `class="koboSpan"`) {
		t.Fatalf("expected koboSpan in output, got: %s", output)
	}
	if !strings.Contains(output, `id="koboSpan-1"`) {
		t.Fatalf("expected koboSpan-1 in output, got: %s", output)
	}
}

func TestConvertEPUBToKePubConvertsHTMLAndStreamsAssets(t *testing.T) {
	input := makeZip(t, map[string][]byte{
		"chapter.xhtml": []byte(`<html><body><p>Hello.</p></body></html>`),
		"image.bin":     {0, 1, 2, 3, 4},
	})
	var output bytes.Buffer
	if err := ConvertEPUBToKePub(bytes.NewReader(input), int64(len(input)), &output); err != nil {
		t.Fatal(err)
	}

	converted, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatalf("output is not a valid zip: %v", err)
	}
	for _, file := range converted.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		switch file.Name {
		case "chapter.xhtml":
			if !bytes.Contains(data, []byte(`class="koboSpan"`)) {
				t.Fatalf("HTML was not converted: %s", data)
			}
		case "image.bin":
			if !bytes.Equal(data, []byte{0, 1, 2, 3, 4}) {
				t.Fatalf("asset changed: %v", data)
			}
		}
	}
}

func TestConvertEPUBToKePubRejectsArchiveLimitsBeforeWriting(t *testing.T) {
	t.Run("entries", func(t *testing.T) {
		entries := make(map[string][]byte, constants.MaxArchiveEntries+1)
		for i := 0; i <= constants.MaxArchiveEntries; i++ {
			entries[fmt.Sprintf("%d", i)] = nil
		}
		input := makeZip(t, entries)
		var output bytes.Buffer
		if err := ConvertEPUBToKePub(bytes.NewReader(input), int64(len(input)), &output); err == nil || output.Len() != 0 {
			t.Fatalf("expected entry limit error before output, err=%v output=%d", err, output.Len())
		}
	})

	t.Run("entry size", func(t *testing.T) {
		input := makeZip(t, map[string][]byte{"large": nil})
		patchCentralSizes(input, uint32(constants.MaxArchiveAssetSize+1))
		var output bytes.Buffer
		if err := ConvertEPUBToKePub(bytes.NewReader(input), int64(len(input)), &output); err == nil || output.Len() != 0 {
			t.Fatalf("expected asset limit error before output, err=%v output=%d", err, output.Len())
		}
	})

	t.Run("aggregate size", func(t *testing.T) {
		count := int(constants.MaxArchiveUncompressedBytes/constants.MaxArchiveAssetSize) + 1
		entries := make(map[string][]byte, count)
		for i := 0; i < count; i++ {
			entries[fmt.Sprintf("%d", i)] = nil
		}
		input := makeZip(t, entries)
		patchCentralSizes(input, uint32(constants.MaxArchiveAssetSize))
		var output bytes.Buffer
		if err := ConvertEPUBToKePub(bytes.NewReader(input), int64(len(input)), &output); err == nil || output.Len() != 0 {
			t.Fatalf("expected aggregate limit error before output, err=%v output=%d", err, output.Len())
		}
	})
}

func makeZip(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func patchCentralSizes(data []byte, size uint32) {
	for offset := 0; offset+46 <= len(data); {
		if binary.LittleEndian.Uint32(data[offset:]) == 0x02014b50 {
			binary.LittleEndian.PutUint32(data[offset+24:], size)
			offset += 46 + int(binary.LittleEndian.Uint16(data[offset+28:])) + int(binary.LittleEndian.Uint16(data[offset+30:])) + int(binary.LittleEndian.Uint16(data[offset+32:]))
			continue
		}
		offset++
	}
}
