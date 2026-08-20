package main

import (
	"strings"
	"testing"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello <b>World</b></p>", "Hello World"},
		{"<div><span>A</span><br/><span>B</span></div>", "A B"},
		{"No tags here", "No tags here"},
		{"<p>Multiple    spaces   and\nnewlines</p>", "Multiple spaces and newlines"},
	}

	for _, tt := range tests {
		got := stripHTMLTags(tt.input)
		if got != tt.expected {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestInjectMetaTags(t *testing.T) {
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>NovelHub</title>
    <meta name="description" content="A modern light novel library." />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="NovelHub" />
    <meta property="og:title" content="NovelHub" />
    <meta property="og:description" content="A modern light novel library." />
    <meta property="og:image" content="/pwa-512x512.png" />
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="NovelHub" />
    <meta name="twitter:description" content="A modern light novel library." />
    <meta name="twitter:image" content="/pwa-512x512.png" />
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>`

	tags := metaTags{
		Title:       "Test Book - Author Name | Custom Library",
		Description: "A fascinating story about adventures.",
		OGType:      "book",
		OGSiteName:  "Custom Library",
		OGTitle:     "Test Book",
		OGDesc:      "A fascinating story about adventures.",
		OGImage:     "https://calibre.example.com/covers/123.jpg",
		OGImageType: "image/jpeg",
		OGURL:       "https://calibre.example.com/books/123",
		TwitterCard: "summary",
		Author:      "Author Name",
		ReleaseDate: "2024-05-10",
		Tags:        []string{"Fantasy", "Adventure"},
	}

	result := injectMetaTags(htmlTemplate, tags)

	if !strings.Contains(result, "<title>Test Book - Author Name | Custom Library</title>") {
		t.Errorf("expected title to be updated, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta name="description" content="A fascinating story about adventures." />`) {
		t.Errorf("expected description to be updated, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:type" content="book" />`) {
		t.Errorf("expected og:type to be book, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:image" content="https://calibre.example.com/covers/123.jpg" />`) {
		t.Errorf("expected og:image to be updated, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:image:secure_url" content="https://calibre.example.com/covers/123.jpg" />`) {
		t.Errorf("expected og:image:secure_url to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:image:type" content="image/jpeg" />`) {
		t.Errorf("expected og:image:type to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:url" content="https://calibre.example.com/books/123" />`) {
		t.Errorf("expected og:url to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<link rel="canonical" href="https://calibre.example.com/books/123" />`) {
		t.Errorf("expected canonical link to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:book:author" content="Author Name" />`) {
		t.Errorf("expected og:book:author to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:book:release_date" content="2024-05-10" />`) {
		t.Errorf("expected og:book:release_date to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:book:tag" content="Fantasy" />`) {
		t.Errorf("expected og:book:tag Fantasy to be injected, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:book:tag" content="Adventure" />`) {
		t.Errorf("expected og:book:tag Adventure to be injected, got:\n%s", result)
	}
}
