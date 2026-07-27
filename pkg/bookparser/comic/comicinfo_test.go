package comic

import (
	"strings"
	"testing"
)

func TestParseComicInfoXMLReadingDirection(t *testing.T) {
	cases := []struct {
		name  string
		manga string
		want  string
	}{
		{"right to left", "<Manga>YesAndRightToLeft</Manga>", "rtl"},
		{"manga but ltr", "<Manga>Yes</Manga>", ""},
		{"no tag", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xmlDoc := `<?xml version="1.0"?><ComicInfo><Title>Vol 1</Title>` + tc.manga + `</ComicInfo>`
			meta, err := ParseComicInfoXML(strings.NewReader(xmlDoc))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if meta.ReadingDirection != tc.want {
				t.Fatalf("ReadingDirection = %q, want %q", meta.ReadingDirection, tc.want)
			}
			if meta.Title != "Vol 1" {
				t.Fatalf("Title = %q, want %q", meta.Title, "Vol 1")
			}
		})
	}
}
