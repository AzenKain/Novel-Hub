// Package kobo implements the parts of the Kobo store API that a device needs in order to sync against a self-hosted library.
package kobo

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"novelhub/pkg/jsonx"
)

// SyncTokenHeader carries the token both ways.
const SyncTokenHeader = "x-kobo-synctoken"

const (
	Version    = "1-1-0"
	MinVersion = "1-0-0"
)

// SyncToken is the device's sync cursor.
type SyncToken struct {
	RawKoboStoreToken string

	BooksLastModified        time.Time
	BooksLastCreated         time.Time
	ArchiveLastModified      time.Time
	ReadingStateLastModified time.Time
	TagsLastModified         time.Time
}

type wireToken struct {
	Version string   `json:"version"`
	Data    wireData `json:"data"`
}

type wireData struct {
	RawKoboStoreToken        string `json:"raw_kobo_store_token"`
	BooksLastModified        string `json:"books_last_modified"`
	BooksLastCreated         string `json:"books_last_created"`
	ArchiveLastModified      string `json:"archive_last_modified"`
	ReadingStateLastModified string `json:"reading_state_last_modified"`
	TagsLastModified         string `json:"tags_last_modified"`
}

// ParseSyncToken reads the header value.
func ParseSyncToken(header string) SyncToken {
	header = strings.TrimSpace(header)
	if header == "" {
		return SyncToken{}
	}
	if strings.Contains(header, ".") {
		return SyncToken{RawKoboStoreToken: header}
	}

	if pad := len(header) % 4; pad != 0 {
		header += strings.Repeat("=", 4-pad)
	}
	raw, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return SyncToken{}
	}
	var wire wireToken
	if err := jsonx.Unmarshal(raw, &wire); err != nil {
		return SyncToken{}
	}
	if compareVersion(wire.Version, MinVersion) < 0 {
		return SyncToken{}
	}
	return SyncToken{
		RawKoboStoreToken:        wire.Data.RawKoboStoreToken,
		BooksLastModified:        parseEpoch(wire.Data.BooksLastModified),
		BooksLastCreated:         parseEpoch(wire.Data.BooksLastCreated),
		ArchiveLastModified:      parseEpoch(wire.Data.ArchiveLastModified),
		ReadingStateLastModified: parseEpoch(wire.Data.ReadingStateLastModified),
		TagsLastModified:         parseEpoch(wire.Data.TagsLastModified),
	}
}

// Encode renders the token for the response header.
func (t SyncToken) Encode() (string, error) {
	payload, err := jsonx.Marshal(wireToken{
		Version: Version,
		Data: wireData{
			RawKoboStoreToken:        t.RawKoboStoreToken,
			BooksLastModified:        formatEpoch(t.BooksLastModified),
			BooksLastCreated:         formatEpoch(t.BooksLastCreated),
			ArchiveLastModified:      formatEpoch(t.ArchiveLastModified),
			ReadingStateLastModified: formatEpoch(t.ReadingStateLastModified),
			TagsLastModified:         formatEpoch(t.TagsLastModified),
		},
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(payload), nil
}

func formatEpoch(t time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return strconv.FormatInt(t.UTC().Unix(), 10)
}

func parseEpoch(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return time.Time{}
	}
	if dot := strings.IndexByte(raw, '.'); dot >= 0 {
		raw = raw[:dot]
	}
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || secs <= 0 {
		return time.Time{}
	}
	return time.Unix(secs, 0).UTC()
}

func compareVersion(a, b string) int {
	aParts := strings.Split(a, "-")
	bParts := strings.Split(b, "-")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		var av, bv int64
		if i < len(aParts) {
			av, _ = strconv.ParseInt(aParts[i], 10, 64)
		}
		if i < len(bParts) {
			bv, _ = strconv.ParseInt(bParts[i], 10, 64)
		}
		switch {
		case av < bv:
			return -1
		case av > bv:
			return 1
		}
	}
	return 0
}
