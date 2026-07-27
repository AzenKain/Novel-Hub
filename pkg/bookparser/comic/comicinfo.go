package comic

import (
	"encoding/xml"
	"io"
	"strings"

	"novelhub/pkg/bookparser"
)

type ComicInfo struct {
	XMLName     xml.Name `xml:"ComicInfo"`
	Title       string   `xml:"Title"`
	Series      string   `xml:"Series"`
	Number      string   `xml:"Number"`
	Volume      string   `xml:"Volume"`
	Summary     string   `xml:"Summary"`
	Writer      string   `xml:"Writer"`
	Publisher   string   `xml:"Publisher"`
	Genre       string   `xml:"Genre"`
	PageCount   int      `xml:"PageCount"`
	LanguageISO string   `xml:"LanguageISO"`
	Manga       string   `xml:"Manga"`
}

func ParseComicInfoXML(r io.Reader) (*bookparser.BookMetadata, error) {
	var info ComicInfo
	if err := xml.NewDecoder(r).Decode(&info); err != nil {
		return nil, err
	}

	meta := &bookparser.BookMetadata{
		Title:       info.Title,
		Series:      info.Series,
		SeriesIndex: info.Number,
		Description: info.Summary,
		Author:      info.Writer,
		Publisher:   info.Publisher,
		Language:    info.LanguageISO,
	}
	if info.Genre != "" {
		meta.Subjects = []string{info.Genre}
	}
	if info.Volume != "" && meta.SeriesIndex == "" {
		meta.SeriesIndex = info.Volume
	}
	if strings.EqualFold(info.Manga, "YesAndRightToLeft") {
		meta.ReadingDirection = "rtl"
	}

	return meta, nil
}
