package opds

import (
	"encoding/xml"
	"time"
)

const (
	NamespaceAtom = "http://www.w3.org/2005/Atom"
	NamespaceOpds = "http://opds-spec.org/2010/catalog"
	NamespaceDc   = "http://purl.org/dc/terms/"
	NamespaceOs   = "http://a9.com/-/spec/opensearch/1.1/"
)

type Feed struct {
	XMLName      xml.Name  `xml:"feed"`
	Xmlns        string    `xml:"xmlns,attr"`
	XmlnsDc      string    `xml:"xmlns:dc,attr"`
	XmlnsOpds    string    `xml:"xmlns:opds,attr"`
	XmlnsOs      string    `xml:"xmlns:os,attr,omitempty"`
	ID           string    `xml:"id"`
	Title        string    `xml:"title"`
	Updated      time.Time `xml:"updated"`
	Icon         string    `xml:"icon,omitempty"`
	Author       *Author   `xml:"author,omitempty"`
	Links        []Link    `xml:"link"`
	Entries      []Entry   `xml:"entry"`
	ItemsPerPage int       `xml:"os:itemsPerPage,omitempty"`
	TotalResults int       `xml:"os:totalResults,omitempty"`
}

type Entry struct {
	ID        string    `xml:"id"`
	Title     string    `xml:"title"`
	Updated   time.Time `xml:"updated"`
	Published time.Time `xml:"published,omitempty"`
	Summary   string    `xml:"summary,omitempty"`
	Content   string    `xml:"content,omitempty"`
	Author    *Author   `xml:"author,omitempty"`
	Links     []Link    `xml:"link"`
	Category  *Category `xml:"category,omitempty"`
	Language  string    `xml:"dc:language,omitempty"`
	Publisher string    `xml:"dc:publisher,omitempty"`
	Issued    string    `xml:"dc:issued,omitempty"`
}

type Link struct {
	Href  string `xml:"href,attr"`
	Type  string `xml:"type,attr"`
	Rel   string `xml:"rel,attr"`
	Title string `xml:"title,attr,omitempty"`
}

type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

type Category struct {
	Term  string `xml:"term,attr"`
	Label string `xml:"label,attr,omitempty"`
}

type OpenSearchDescription struct {
	XMLName        xml.Name      `xml:"OpenSearchDescription"`
	Xmlns          string        `xml:"xmlns,attr"`
	ShortName      string        `xml:"ShortName"`
	Description    string        `xml:"Description"`
	InputEncoding  string        `xml:"InputEncoding"`
	OutputEncoding string        `xml:"OutputEncoding"`
	URL            OpenSearchURL `xml:"Url"`
}

type OpenSearchURL struct {
	Type     string `xml:"type,attr"`
	Template string `xml:"template,attr"`
}
