package webdav

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMultiStatusXML(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

	nodes := []WebDAVNode{
		{
			Href:        "/webdav/",
			DisplayName: "NovelHub",
			IsDir:       true,
			ModTime:     now,
		},
		{
			Href:        "/webdav/Default Library/",
			DisplayName: "Default Library",
			IsDir:       true,
			ModTime:     now,
		},
		{
			Href:        "/webdav/Default Library/Overlord Vol 1.epub",
			DisplayName: "Overlord Vol 1.epub",
			IsDir:       false,
			Size:        1548291,
			ContentType: "application/epub+zip",
			ModTime:     now,
			ETag:        "abc123etag",
		},
	}

	data, err := BuildMultiStatusXML(nodes)
	if err != nil {
		t.Fatalf("BuildMultiStatusXML failed: %v", err)
	}

	xmlStr := string(data)

	// Check DAV namespace
	if !strings.Contains(xmlStr, `<D:multistatus xmlns:D="DAV:">`) {
		t.Fatalf("missing D:multistatus element, got:\n%s", xmlStr)
	}

	// Check collection resource type
	if !strings.Contains(xmlStr, "<D:collection/>") && !strings.Contains(xmlStr, "<D:collection></D:collection>") {
		t.Fatalf("missing collection resourcetype in folder node, got:\n%s", xmlStr)
	}

	// Check content length and type
	if !strings.Contains(xmlStr, "<D:getcontentlength>1548291</D:getcontentlength>") {
		t.Fatalf("missing getcontentlength in file node, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<D:getcontenttype>application/epub+zip</D:getcontenttype>") {
		t.Fatalf("missing getcontenttype in file node, got:\n%s", xmlStr)
	}

	// Check ETag
	if !strings.Contains(xmlStr, "<D:getetag>abc123etag</D:getetag>") {
		t.Fatalf("missing formatted getetag, got:\n%s", xmlStr)
	}
}

func TestParseDepth(t *testing.T) {
	if ParseDepth("0") != 0 {
		t.Fatal("expected depth 0")
	}
	if ParseDepth("1") != 1 {
		t.Fatal("expected depth 1")
	}
	if ParseDepth("infinity") != 1 {
		t.Fatal("expected depth 1 for infinity (constrained)")
	}
	if ParseDepth("") != 1 {
		t.Fatal("expected default depth 1 for empty header")
	}
}

func TestSanitizeWebDAVPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "/"},
		{"/", "/"},
		{"/webdav", "/"},
		{"/webdav/", "/"},
		{"/webdav/Default", "/Default"},
		{"/webdav/Default/Book.epub", "/Default/Book.epub"},
		{"/webdav/Default/Sub/Book.epub", "/Default/Sub/Book.epub"},
		{"webdav/Default", "/Default"},
	}

	for _, tt := range tests {
		got := SanitizeWebDAVPath(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeWebDAVPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
