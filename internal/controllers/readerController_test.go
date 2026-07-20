package controllers

import (
	"net/url"
	"testing"
)

func TestEncodedChapterIDRoundTrips(t *testing.T) {
	chapterID := "019f302a-7732-73a2-8b9a-151ba649b1cf:0"
	encoded := url.PathEscape(chapterID)
	decoded := decodeRouteParam(encoded)
	if decoded != chapterID {
		t.Fatalf("expected %q, got %q", chapterID, decoded)
	}
}
