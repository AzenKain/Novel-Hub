package defaultcover

import (
	"strings"
	"testing"
)

func TestGenerateSVG_WordWrap(t *testing.T) {
	title := "0: Introducing Artsy Engineering Radio"
	author := "Artsy Engineering"

	svgBytes := GenerateSVG(title, author)
	svgStr := string(svgBytes)

	if !strings.Contains(svgStr, "<?xml") {
		t.Error("expected XML header in SVG output")
	}

	if !strings.Contains(svgStr, "0: Introducing Artsy") {
		t.Error("expected first wrapped title line in SVG output")
	}
	if !strings.Contains(svgStr, "Engineering Radio") {
		t.Error("expected second wrapped title line in SVG output")
	}

	if !strings.Contains(svgStr, "Artsy Engineering") {
		t.Error("expected author name in SVG output")
	}
}

func TestGenerateSVG_ShortTitle(t *testing.T) {
	title := "Short"
	author := "Author"

	svgBytes := GenerateSVG(title, author)
	svgStr := string(svgBytes)

	if !strings.Contains(svgStr, "Short") {
		t.Error("expected short title in SVG output")
	}
}
