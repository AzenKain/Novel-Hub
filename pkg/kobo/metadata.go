package kobo

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TimestampFormat is the only timestamp format the device accepts in entitlement and metadata payloads: calibre-web's convert_to_kobo_timestamp_string().
const TimestampFormat = "2006-01-02T15:04:05Z"

// SyncItemLimit caps one sync response.
const SyncItemLimit = 100

const (
	SyncContinueHeader = "x-kobo-sync"
	SyncContinueValue  = "continue"
)

const (
	APITokenHeader = "x-kobo-apitoken"
	APITokenValue  = "e30="
)

// KoboFormats maps a stored file format to the format names the device understands.
var KoboFormats = map[string][]string{
	"KEPUB": {"KEPUB"},
	"EPUB":  {"EPUB3", "EPUB"},
}

const defaultCategoryGUID = "00000000-0000-0000-0000-000000000001"

// FormatTimestamp renders a timestamp for the wire.
func FormatTimestamp(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format(TimestampFormat)
	}
	return t.UTC().Format(TimestampFormat)
}

// DownloadURL is one entry of BookMetadata.DownloadUrls.
type DownloadURL struct {
	Format   string `json:"Format"`
	Size     int64  `json:"Size"`
	URL      string `json:"Url"`
	Platform string `json:"Platform"`
}

// Series is BookMetadata.Series.
type Series struct {
	Name        string  `json:"Name"`
	Number      int     `json:"Number"`
	NumberFloat float64 `json:"NumberFloat"`
	ID          string  `json:"Id"`
}

// Publisher is BookMetadata.Publisher.
type Publisher struct {
	Imprint string `json:"Imprint"`
	Name    string `json:"Name"`
}

// Price covers CurrentDisplayPrice and CurrentLoveDisplayPrice.
type Price struct {
	CurrencyCode string  `json:"CurrencyCode,omitempty"`
	TotalAmount  float64 `json:"TotalAmount"`
}

// ContributorRole is one entry of BookMetadata.ContributorRoles.
type ContributorRole struct {
	Name string `json:"Name"`
}

// TagItem represents a book item inside a Kobo Collection/UserTag.
type TagItem struct {
	BookID    string `json:"BookId"`
	DateAdded string `json:"DateAdded"`
}

// Tag represents a Kobo UserTag (Collection / Shelf on the eReader).
type Tag struct {
	ID           string    `json:"Id"`
	Name         string    `json:"Name"`
	Type         string    `json:"Type"`
	Items        []TagItem `json:"Items"`
	Created      string    `json:"Created"`
	LastModified string    `json:"LastModified"`
}

// BookEntitlement is the device's record that a user owns a book.
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

// BookMetadata is what the device renders in the library.
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

// BookInfo is the server-side input to the wire types above — the subset of a book the Kobo payloads actually need, so the builders stay independent of NovelHub's entity types.
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
			index = 1
		}
		meta.Series = &Series{
			Name:        name,
			Number:      int(index),
			NumberFloat: index,
			ID:          uuid.NewMD5(uuid.NameSpaceDNS, []byte(name)).String(),
		}
	}

	return meta
}

// BookDownloadURL builds one DownloadUrls entry.
func BookDownloadURL(endpointURL, bookID, format, koboFormat string, size int64) DownloadURL {
	return DownloadURL{
		Format:   koboFormat,
		Size:     size,
		URL:      fmt.Sprintf("%s/download/%s/%s", strings.TrimRight(endpointURL, "/"), bookID, strings.ToLower(format)),
		Platform: "Generic",
	}
}
