package presentation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresentationParserWithRealFiles(t *testing.T) {
	parser := NewParser()

	pptxPath := filepath.Join("..", "..", "..", "sample_file", "sample-5.pptx")
	if _, err := os.Stat(pptxPath); err == nil {
		meta, err := parser.ParseMetadata(pptxPath)
		if err != nil {
			t.Fatalf("PPTX ParseMetadata: %v", err)
		}
		if meta.Title == "" {
			t.Errorf("expected PPTX title")
		}

		spine, err := parser.ParseSpine(pptxPath)
		if err != nil || len(spine) == 0 {
			t.Fatalf("PPTX ParseSpine: %v (len %d)", err, len(spine))
		}

		slide1, err := parser.GetChapterContent(pptxPath, spine[0].ContentPath)
		if err != nil {
			t.Fatalf("PPTX GetChapterContent: %v", err)
		}
		if !strings.Contains(slide1, "<article") {
			t.Errorf("expected article in slide content, got: %s", slide1)
		}

		images, err := parser.ListImages(pptxPath)
		if err != nil {
			t.Fatalf("PPTX ListImages: %v", err)
		}
		if len(images) == 0 {
			t.Errorf("expected images in sample-5.pptx")
		}
	}

	odpPath := filepath.Join("..", "..", "..", "sample_file", "file_example_ODP_1MB.odp")
	if _, err := os.Stat(odpPath); err == nil {
		meta, err := parser.ParseMetadata(odpPath)
		if err != nil {
			t.Fatalf("ODP ParseMetadata: %v", err)
		}
		if meta.Title == "" {
			t.Errorf("expected ODP title")
		}

		spine, err := parser.ParseSpine(odpPath)
		if err != nil || len(spine) == 0 {
			t.Fatalf("ODP ParseSpine: %v (len %d)", err, len(spine))
		}

		content, err := parser.GetChapterContent(odpPath, spine[0].ContentPath)
		if err != nil {
			t.Fatalf("ODP GetChapterContent: %v", err)
		}
		if !strings.Contains(content, "<article") {
			t.Errorf("expected article in odp content, got: %s", content)
		}
	}

	pptPath := filepath.Join("..", "..", "..", "sample_file", "file_example_PPT_1MB.ppt")
	if _, err := os.Stat(pptPath); err == nil {
		meta, err := parser.ParseMetadata(pptPath)
		if err != nil {
			t.Fatalf("PPT ParseMetadata: %v", err)
		}
		if meta.Title == "" {
			t.Errorf("expected PPT title")
		}

		spine, err := parser.ParseSpine(pptPath)
		if err != nil || len(spine) == 0 {
			t.Fatalf("PPT ParseSpine: %v (len %d)", err, len(spine))
		}

		content, err := parser.GetChapterContent(pptPath, spine[0].ContentPath)
		if err != nil {
			t.Fatalf("PPT GetChapterContent: %v", err)
		}
		if !strings.Contains(content, "<article") {
			t.Errorf("expected article in ppt content, got: %s", content)
		}
	}
}

func TestPresentationMediaRendering(t *testing.T) {
	var out strings.Builder
	renderMediaElement(&out, "ppt/media/soundtrack.mp3")
	if !strings.Contains(out.String(), "<audio controls") || !strings.Contains(out.String(), "soundtrack.mp3") {
		t.Errorf("expected audio element, got: %s", out.String())
	}

	out.Reset()
	renderMediaElement(&out, "ppt/media/clip.mp4")
	if !strings.Contains(out.String(), "<video controls") || !strings.Contains(out.String(), "clip.mp4") {
		t.Errorf("expected video element, got: %s", out.String())
	}

	out.Reset()
	renderMediaElement(&out, "ppt/media/chart.png")
	if !strings.Contains(out.String(), "<img src=") || !strings.Contains(out.String(), "chart.png") {
		t.Errorf("expected image element, got: %s", out.String())
	}
}
