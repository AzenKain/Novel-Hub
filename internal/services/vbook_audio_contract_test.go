package services

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"novelhub/pkg/cache"
)

func TestVBookAudioZipDetailContract(t *testing.T) {
	for _, root := range []string{"../../cmd/api/dist/vbook", "../../web/public/vbook"} {
		fsys := os.DirFS(filepath.Clean(root))
		svc := NewVBookService(nil, nil, nil, nil, fsys, cache.NewRamCache())
		zipBytes, err := svc.GetPluginZipAudio(context.Background(), "http://localhost:3434")
		if err != nil {
			t.Fatalf("%s: %v", root, err)
		}
		zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
		if err != nil {
			t.Fatal(err)
		}
		var detailSrc string
		for _, f := range zr.File {
			if f.Name == "src/detail.js" {
				rc, _ := f.Open()
				buf := new(bytes.Buffer)
				buf.ReadFrom(rc)
				rc.Close()
				detailSrc = buf.String()
				break
			}
		}
		if detailSrc == "" {
			t.Fatalf("%s: src/detail.js missing from zip", root)
		}
		if !bytes.Contains([]byte(detailSrc), []byte(`detail.format = "album"`)) {
			t.Fatalf("%s: audio detail lacks format:\"album\":\n%s", root, detailSrc)
		}
	}
}
