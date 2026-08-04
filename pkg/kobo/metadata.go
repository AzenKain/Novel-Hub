package kobo

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TimestampFormat is the only timestamp format the device accepts in entitlement and
// metadata payloads: calibre-web's convert_to_kobo_timestamp_string().
const TimestampFormat = "2006-01-02T15:04:05Z"

// SyncItemLimit caps one sync response. When more remain, the response carries
// "x-kobo-sync: continue" and the device immediately asks again.
const SyncItemLimit = 100

// SyncContinueHeader / SyncContinueValue tell the device there is another page.
const (
	SyncContinueHeader = "x-kobo-sync"
	SyncContinueValue  = "continue"
)

// APITokenHeader is returned with /v1/initialization. calibre-web sends the constant "e30="
// (base64 of "{}"), i.e. an empty token — the device only checks that it is present.
const (
	APITokenHeader = "x-kobo-apitoken"
	APITokenValue  = "e30="
)

// KoboFormats maps a stored file format to the format names the device understands. Anything
// not in here is not offered for download. Mirrors calibre-web's KOBO_FORMATS.
var KoboFormats = map[string][]string{
	"KEPUB": {"KEPUB"},
	"EPUB":  {"EPUB3", "EPUB"},
}

// defaultCategoryGUID is a placeholder GUID calibre-web uses for both Categories and Genre.
// The device requires the fields to be well-formed GUIDs but does not act on the values.
const defaultCategoryGUID = "00000000-0000-0000-0000-000000000001"

// FormatTimestamp renders a timestamp for the wire. A zero time becomes "now" rather than
// year 1: calibre-web falls back to now on AttributeError, and a year-1 date has been observed
// to make devices reject an entitlement outright.
func FormatTimestamp(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format(TimestampFormat)
	}
	return t.UTC().Format(TimestampFormat)
}

// DownloadURL is one entry of BookMetadata.DownloadUrls. Without at least one of these the
// device shows the book but has no way to fetch it.
type DownloadURL struct {
	Format   string `json:"Format"`
	Size     int64  `json:"Size"`
	URL      string `json:"Url"`
	Platform string `json:"Platform"`
}

// Series is BookMetadata.Series. Id must be stable across syncs or the device treats each sync
// as a different series, so it is derived from the name (calibre-web: uuid3 of the name).
type Series struct {
	Name        string  `json:"Name"`
	Number      int     `json:"Number"`
	NumberFloat float64 `json:"NumberFloat"`
	ID          string  `json:"Id"`
}

// Publisher is BookMetadata.Publisher. Imprint is always sent, empty if unknown — the device
// expects the key to exist.
type Publisher struct {
	Imprint string `json:"Imprint"`
	Name    string `json:"Name"`
}

// Price covers CurrentDisplayPrice and CurrentLoveDisplayPrice. Self-hosted books are free,
// but omitting the fields makes the device treat the entitlement as unpurchased.
type Price struct {
	CurrencyCode string  `json:"CurrencyCode,omitempty"`
	TotalAmount  float64 `json:"TotalAmount"`
}

// ContributorRole is one entry of BookMetadata.ContributorRoles.
type ContributorRole struct {
	Name string `json:"Name"`
}

// BookEntitlement is the device's record that a user owns a book. Field set and values are
// calibre-web's create_book_entitlement().
type BookEntitlement struct {
	Accessibility       string            `json:"Accessibility"`
	ActivePeriod        map[string]string `json:"ActivePeriod"`
	Created             string            `json:"Created"`
	CrossRevisionID     string            `json:"CrossRevisionId"`
	ID                  string            `json:"Id"`
	IsRemoved           bool              `json:"IsRemoved"`
	IsHiddenFromArchive bool              `json:"IsHiddenFromArchive"`
	IsLocked            bool              `json:"IsLocked"`
	LastModified        string            `json:"LastModified"`
	OriginCategory      string            `json:"OriginCategory"`
	RevisionID          string            `json:"RevisionId"`
	Status              string            `json:"Status"`
}

// BookMetadata is what the device renders in the library. Mirrors calibre-web's get_metadata().
//
// The same book UUID is deliberately reused for CoverImageId, CrossRevisionId, EntitlementId,
// RevisionId and WorkId — the store distinguishes these, a self-hosted library has nothing to
// distinguish, and the device is happy as long as they are consistent between syncs.
type BookMetadata struct {
	Categories              []string          `json:"Categories"`
	CoverImageID            string            `json:"CoverImageId"`
	CrossRevisionID         string            `json:"CrossRevisionId"`
	CurrentDisplayPrice     Price             `json:"CurrentDisplayPrice"`
	CurrentLoveDisplayPrice Price             `json:"CurrentLoveDisplayPrice"`
	Description             *string           `json:"Description"`
	DownloadUrls            []DownloadURL     `json:"DownloadUrls"`
	EntitlementID           string            `json:"EntitlementId"`
	ExternalIDs             []string          `json:"ExternalIds"`
	Genre                   string            `json:"Genre"`
	IsEligibleForKoboLove   bool              `json:"IsEligibleForKoboLove"`
	IsInternetArchive       bool              `json:"IsInternetArchive"`
	IsPreOrder              bool              `json:"IsPreOrder"`
	IsSocialEnabled         bool              `json:"IsSocialEnabled"`
	Language                string            `json:"Language"`
	PhoneticPronunciations  map[string]string `json:"PhoneticPronunciations"`
	PublicationDate         string            `json:"PublicationDate"`
	Publisher               Publisher         `json:"Publisher"`
	RevisionID              string            `json:"RevisionId"`
	Title                   string            `json:"Title"`
	WorkID                  string            `json:"WorkId"`
	ContributorRoles        []ContributorRole `json:"ContributorRoles,omitempty"`
	Contributors            []string          `json:"Contributors,omitempty"`
	Series                  *Series           `json:"Series,omitempty"`
}

// BookInfo is the server-side input to the wire types above — the subset of a book the Kobo
// payloads actually need, so the builders stay independent of NovelHub's entity types.
type BookInfo struct {
	UUID         string
	Title        string
	Description  *string
	Authors      []string
	Publisher    string
	Language     string
	SeriesName   string
	SeriesIndex  float64
	Created      time.Time
	LastModified time.Time
	PublishedAt  time.Time
	Archived     bool
	Downloads    []DownloadURL
}

// NewBookEntitlement builds the entitlement half of a sync item.
func NewBookEntitlement(b BookInfo) BookEntitlement {
	return BookEntitlement{
		Accessibility:       "Full",
		ActivePeriod:        map[string]string{"From": FormatTimestamp(time.Now().UTC())},
		Created:             FormatTimestamp(b.Created),
		CrossRevisionID:     b.UUID,
		ID:                  b.UUID,
		IsRemoved:           b.Archived,
		IsHiddenFromArchive: false,
		IsLocked:            false,
		LastModified:        FormatTimestamp(b.LastModified),
		OriginCategory:      "Imported",
		RevisionID:          b.UUID,
		Status:              "Active",
	}
}

// NewBookMetadata builds the metadata half of a sync item.
func NewBookMetadata(b BookInfo) BookMetadata {
	lang := strings.TrimSpace(b.Language)
	if lang == "" {
		// calibre-web defaults to "en" when a book has no language; the device needs a value.
		lang = "en"
	}

	downloads := b.Downloads
	if downloads == nil {
		downloads = []DownloadURL{}
	}

	meta := BookMetadata{
		Categories:              []string{defaultCategoryGUID},
		CoverImageID:            b.UUID,
		CrossRevisionID:         b.UUID,
		CurrentDisplayPrice:     Price{CurrencyCode: "USD", TotalAmount: 0},
		CurrentLoveDisplayPrice: Price{TotalAmount: 0},
		Description:             b.Description,
		DownloadUrls:            downloads,
		EntitlementID:           b.UUID,
		ExternalIDs:             []string{},
		Genre:                   defaultCategoryGUID,
		IsEligibleForKoboLove:   false,
		IsInternetArchive:       false,
		IsPreOrder:              false,
		IsSocialEnabled:         true,
		Language:                lang,
		PhoneticPronunciations:  map[string]string{},
		PublicationDate:         FormatTimestamp(b.PublishedAt),
		Publisher:               Publisher{Imprint: "", Name: b.Publisher},
		RevisionID:              b.UUID,
		Title:                   b.Title,
		WorkID:                  b.UUID,
	}

	for _, author := range b.Authors {
		author = strings.TrimSpace(author)
		if author == "" {
			continue
		}
		meta.ContributorRoles = append(meta.ContributorRoles, ContributorRole{Name: author})
		meta.Contributors = append(meta.Contributors, author)
	}

	if name := strings.TrimSpace(b.SeriesName); name != "" {
		index := b.SeriesIndex
		if index == 0 {
			// calibre-web's get_seriesindex() falls back to 1, not 0: a device sorts "book 0"
			// oddly and some firmware hides it.
			index = 1
		}
		meta.Series = &Series{
			Name:        name,
			Number:      int(index),
			NumberFloat: index,
			// Derived from the name so it is identical on every sync and every server — the
			// device groups books by this id. calibre-web uses uuid3(NAMESPACE_DNS, name),
			// which is the MD5-based variant. Determinism is the only requirement here; this
			// is not a security use.
			ID: uuid.NewMD5(uuid.NameSpaceDNS, []byte(name)).String(),
		}
	}

	return meta
}

// BookDownloadURL builds one DownloadUrls entry. format is the stored format ("EPUB"),
// koboFormat is what the device is told ("EPUB3").
func BookDownloadURL(endpointURL, bookID, format, koboFormat string, size int64) DownloadURL {
	return DownloadURL{
		Format:   koboFormat,
		Size:     size,
		URL:      fmt.Sprintf("%s/download/%s/%s", strings.TrimRight(endpointURL, "/"), bookID, strings.ToLower(format)),
		Platform: "Generic",
	}
}
