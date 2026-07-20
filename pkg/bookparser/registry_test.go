package bookparser

import "testing"

type fakeParser struct{}

func (fakeParser) ParseMetadata(filePath string) (*BookMetadata, error) { return &BookMetadata{}, nil }
func (fakeParser) ParseSpine(filePath string) ([]ChapterData, error)    { return nil, nil }
func (fakeParser) ParseBook(filePath string) (*BookData, error)         { return &BookData{}, nil }
func (fakeParser) GetChapterContent(filePath, contentPath string) (string, error) {
	return "", nil
}
func (fakeParser) GetAsset(filePath, assetPath string) ([]byte, error) { return nil, nil }
func (fakeParser) ListImages(filePath string) ([]string, error)        { return nil, nil }
func (fakeParser) SaveOriginalMetadataAndFix(filePath string, meta *BookMetadata) error {
	return nil
}

func TestRegistryFormatDetection(t *testing.T) {
	registry := NewRegistry()
	registry.Register(fakeParser{}, "epub", "md")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "epub", path: "Book.epub", want: "epub"},
		{name: "kepub", path: "Book.kepub.epub", want: "kepub.epub"},
		{name: "markdown", path: "notes.markdown", want: "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatFromPath(tt.path); got != wantNormalized(tt.want) {
				t.Fatalf("FormatFromPath(%q) = %q, want %q", tt.path, got, wantNormalized(tt.want))
			}
		})
	}

	if _, ok := registry.ParserForFormat(".md"); !ok {
		t.Fatal("expected md parser")
	}
	if _, ok := registry.ParserForPath("notes.markdown"); !ok {
		t.Fatal("expected markdown path to resolve to md parser")
	}
}

func wantNormalized(format string) string {
	if format == "markdown" {
		return "md"
	}
	return format
}
