package comic

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"novelhub/pkg/constants"
)

var ErrEmptyReadingList = errors.New("reading list has no book entries")

type ReadingList struct {
	XMLName xml.Name          `xml:"ReadingList"`
	Name    string            `xml:"Name"`
	Books   []ReadingListBook `xml:"Books>Book"`
}

// ComicRack publishes no XSD, so every attribute is optional and unknown ones are ignored.
type ReadingListBook struct {
	Series string `xml:"Series,attr"`
	Number string `xml:"Number,attr"`
	Volume string `xml:"Volume,attr"`
	Year   string `xml:"Year,attr"`
}

// The slice keeps document order, which for a .cbl IS the reading order — nothing sorts it.
func ParseCBL(r io.Reader) (*ReadingList, error) {
	var list ReadingList
	if err := xml.NewDecoder(io.LimitReader(r, constants.MaxCBLBytes)).Decode(&list); err != nil {
		return nil, err
	}
	list.Name = strings.TrimSpace(list.Name)
	for i := range list.Books {
		list.Books[i].Series = strings.TrimSpace(list.Books[i].Series)
		list.Books[i].Number = strings.TrimSpace(list.Books[i].Number)
		list.Books[i].Volume = strings.TrimSpace(list.Books[i].Volume)
		list.Books[i].Year = strings.TrimSpace(list.Books[i].Year)
	}
	if len(list.Books) == 0 {
		return nil, ErrEmptyReadingList
	}
	return &list, nil
}
