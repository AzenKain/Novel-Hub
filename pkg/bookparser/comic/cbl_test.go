package comic

import (
	"strings"
	"testing"

	"novelhub/pkg/constants"
)

const civilWarCBL = `<?xml version="1.0"?>
<ReadingList xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Name>  MV2 - Civil War  </Name>
  <Books>
    <Book Series="Amazing Spider-Man" Number="529" Volume="1963" Year="2006">
      <Id>b1a2c3d4-0000-0000-0000-000000000001</Id>
      <FileName>Amazing Spider-Man 529.cbz</FileName>
    </Book>
    <Book Series="Civil War" Number=" 1 " Volume="2006" Year="2006" Format="Comic" />
    <Book Series="Captain America" Number="25" />
  </Books>
</ReadingList>`

// Document order IS the reading order for a .cbl — Civil War #1 has to stay between the two tie-in issues.
func TestParseCBLKeepsDocumentOrder(t *testing.T) {
	list, err := ParseCBL(strings.NewReader(civilWarCBL))
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "MV2 - Civil War" {
		t.Errorf("Name = %q, want the trimmed name", list.Name)
	}
	want := []struct{ series, number string }{
		{"Amazing Spider-Man", "529"},
		{"Civil War", "1"},
		{"Captain America", "25"},
	}
	if len(list.Books) != len(want) {
		t.Fatalf("got %d books, want %d", len(list.Books), len(want))
	}
	for i, w := range want {
		if list.Books[i].Series != w.series || list.Books[i].Number != w.number {
			t.Errorf("book %d = %q #%q, want %q #%q", i, list.Books[i].Series, list.Books[i].Number, w.series, w.number)
		}
	}
}

// Volume is the series start year in real .cbl files, not a sequence number.
func TestParseCBLDoesNotFallBackVolumeToNumber(t *testing.T) {
	list, err := ParseCBL(strings.NewReader(`<ReadingList><Name>x</Name><Books>
		<Book Series="Civil War" Volume="2006" /></Books></ReadingList>`))
	if err != nil {
		t.Fatal(err)
	}
	if list.Books[0].Number != "" {
		t.Errorf("Number = %q, want empty — Volume must not leak into Number", list.Books[0].Number)
	}
}

func TestParseCBLRejectsBadInput(t *testing.T) {
	for name, body := range map[string]string{
		"broken xml":  `<ReadingList><Books><Book Series="a"`,
		"no books":    `<ReadingList><Name>empty</Name><Books></Books></ReadingList>`,
		"wrong root":  `<ComicInfo><Series>a</Series></ComicInfo>`,
		"empty input": ``,
	} {
		if _, err := ParseCBL(strings.NewReader(body)); err == nil {
			t.Errorf("%s: parsed without error, want a failure", name)
		}
	}
}

// A 40MB <Name> would otherwise be buffered whole.
func TestParseCBLStopsAtSizeLimit(t *testing.T) {
	oversized := `<ReadingList><Name>` + strings.Repeat("A", constants.MaxCBLBytes+1024) +
		`</Name><Books><Book Series="a" Number="1" /></Books></ReadingList>`
	if _, err := ParseCBL(strings.NewReader(oversized)); err == nil {
		t.Fatalf("a %d-byte document parsed fine, want the %d-byte limit to bite", len(oversized), constants.MaxCBLBytes)
	}

	sane := `<ReadingList><Name>ok</Name><Books><Book Series="a" Number="1" /></Books></ReadingList>`
	if _, err := ParseCBL(strings.NewReader(sane)); err != nil {
		t.Fatalf("the limit rejected a normal document too: %v", err)
	}
}
