// Package kobo implements the parts of the Kobo store API that a device needs in order to
// sync against a self-hosted library.
//
// The protocol is not published by Kobo. Everything here mirrors calibre-web
// (cps/kobo.py, cps/services/SyncToken.py, GPL-3.0), which is the reference implementation
// that has been validated against real hardware by its users. Where a choice looked
// arbitrary it was kept identical to calibre-web on purpose: without a device to test
// against, matching a known-working implementation is the only real evidence available.
package kobo

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"novelhub/pkg/jsonx"
)

// SyncTokenHeader carries the token both ways. Devices echo back whatever we last sent.
const SyncTokenHeader = "x-kobo-synctoken"

// Version numbering follows calibre-web so a token written by either implementation can be
// read by the other. MinVersion is the oldest token we will still parse; anything older is
// treated as absent rather than an error, so an old device just re-syncs from scratch.
const (
	Version    = "1-1-0"
	MinVersion = "1-0-0"
)

// SyncToken is the device's sync cursor. It is opaque to the device — it stores whatever we
// send and returns it on the next request — so the fields exist purely to let us answer
// "what changed since last time" without keeping per-device state on the server.
type SyncToken struct {
	// RawKoboStoreToken passes through untouched. A token minted by the real Kobo store has
	// the form "<b64>.<b64>"; we must not reinterpret it, only hand it back.
	RawKoboStoreToken string

	BooksLastModified        time.Time
	BooksLastCreated         time.Time
	ArchiveLastModified      time.Time
	ReadingStateLastModified time.Time
	TagsLastModified         time.Time
}

// wireToken is the JSON that gets base64-encoded into the header. Field names and the
// epoch-seconds encoding are fixed by calibre-web compatibility, not by preference.
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

// ParseSyncToken reads the header value. A malformed or too-old token yields a zero token and
// no error: the device then receives the whole library, which is correct-but-slow rather
// than a failed sync. That mirrors calibre-web, which logs and continues.
func ParseSyncToken(header string) SyncToken {
	header = strings.TrimSpace(header)
	if header == "" {
		return SyncToken{}
	}
	// "<b64>.<b64>" is the real store's format. Keep it verbatim so a device that also talks
	// to the Kobo store does not lose its upstream cursor.
	if strings.Contains(header, ".") {
		return SyncToken{RawKoboStoreToken: header}
	}

	// Restore stripped "=" padding before decoding: calibre-web pads on read because the device
	// (and calibre-web's own encoder in some versions) can drop it.
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

// Timestamps travel as epoch seconds, written as a JSON string. A zero time encodes as "0"
// rather than a negative or huge number: calibre-web uses datetime.min, and anything the
// device cannot parse would make it fall back to a full sync anyway.
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
	// Accept a fractional part: calibre-web writes total_seconds(), which is a float.
	if dot := strings.IndexByte(raw, '.'); dot >= 0 {
		raw = raw[:dot]
	}
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || secs <= 0 {
		return time.Time{}
	}
	return time.Unix(secs, 0).UTC()
}

// compareVersion orders the dash-separated version strings calibre-web uses ("1-1-0").
// Returns -1, 0 or 1.
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
