package kobo

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"novelhub/pkg/jsonx"
)

// The device stores whatever token we send and returns it verbatim next time, so a token that does not survive a round trip breaks incremental sync silently: every sync looks like a first sync and the device re-downloads the whole library.
func TestSyncTokenRoundTrip(t *testing.T) {
	want := SyncToken{
		BooksLastModified:        time.Unix(1_700_000_000, 0).UTC(),
		BooksLastCreated:         time.Unix(1_700_000_111, 0).UTC(),
		ArchiveLastModified:      time.Unix(1_700_000_222, 0).UTC(),
		ReadingStateLastModified: time.Unix(1_700_000_333, 0).UTC(),
		TagsLastModified:         time.Unix(1_700_000_444, 0).UTC(),
	}

	encoded, err := want.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got := ParseSyncToken(encoded)

	for _, tc := range []struct {
		label     string
		want, got time.Time
	}{
		{"BooksLastModified", want.BooksLastModified, got.BooksLastModified},
		{"BooksLastCreated", want.BooksLastCreated, got.BooksLastCreated},
		{"ArchiveLastModified", want.ArchiveLastModified, got.ArchiveLastModified},
		{"ReadingStateLastModified", want.ReadingStateLastModified, got.ReadingStateLastModified},
		{"TagsLastModified", want.TagsLastModified, got.TagsLastModified},
	} {
		if !tc.got.Equal(tc.want) {
			t.Errorf("%s = %v, want %v", tc.label, tc.got, tc.want)
		}
	}
}

// Wire format is fixed by calibre-web compatibility, not preference: base64 of {"version":"1-1-0","data":{...}} with timestamps as epoch-second STRINGS.
func TestSyncTokenWireFormatMatchesCalibreWeb(t *testing.T) {
	encoded, err := SyncToken{BooksLastModified: time.Unix(1_700_000_000, 0).UTC()}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("token is not base64: %v", err)
	}

	var probe struct {
		Version string            `json:"version"`
		Data    map[string]string `json:"data"`
	}
	if err := jsonx.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("token is not the expected JSON shape: %v (%s)", err, raw)
	}
	if probe.Version != "1-1-0" {
		t.Errorf("version = %q, want 1-1-0", probe.Version)
	}
	for _, key := range []string{
		"raw_kobo_store_token", "books_last_modified", "books_last_created",
		"archive_last_modified", "reading_state_last_modified", "tags_last_modified",
	} {
		if _, ok := probe.Data[key]; !ok {
			t.Errorf("data is missing %q; calibre-web's schema requires all six", key)
		}
	}
	if probe.Data["books_last_modified"] != "1700000000" {
		t.Errorf("books_last_modified = %q, want epoch seconds 1700000000", probe.Data["books_last_modified"])
	}
}

// A device that also talks to the real Kobo store sends that store's token, which has the form "<b64>.<b64>".
func TestSyncTokenPassesThroughKoboStoreToken(t *testing.T) {
	const storeToken = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJrb2JvIn0"
	got := ParseSyncToken(storeToken)
	if got.RawKoboStoreToken != storeToken {
		t.Fatalf("store token = %q, want it kept verbatim", got.RawKoboStoreToken)
	}
	if !got.BooksLastModified.IsZero() {
		t.Error("a store token must not be parsed as one of ours")
	}
}

// Anything unreadable must degrade to a full sync rather than an error: a device stuck on a 500 can never recover, whereas a full sync is merely slow.
func TestSyncTokenBadInputFallsBackToFullSync(t *testing.T) {
	for _, tc := range []struct{ label, header string }{
		{"empty", ""},
		{"whitespace", "   "},
		{"not base64", "!!!not-base64!!!"},
		{"base64 of garbage", base64.StdEncoding.EncodeToString([]byte("not json"))},
		{"version below minimum", mustEncodeVersion(t, "0-9-0")},
	} {
		t.Run(tc.label, func(t *testing.T) {
			got := ParseSyncToken(tc.header)
			if !got.BooksLastModified.IsZero() || !got.BooksLastCreated.IsZero() {
				t.Fatalf("%s should yield a zero token (full sync), got %+v", tc.label, got)
			}
		})
	}
}

// calibre-web pads on read because the padding can be stripped in transit.
func TestSyncTokenAcceptsUnpaddedBase64(t *testing.T) {
	payload := []byte(`{"version":"1-1-0","data":{"books_last_modified":"1700000000" }}`)
	encoded := base64.StdEncoding.EncodeToString(payload)
	if !strings.HasSuffix(encoded, "=") {
		t.Fatalf("fixture must exercise padding; %q has none", encoded)
	}
	stripped := strings.TrimRight(encoded, "=")

	got := ParseSyncToken(stripped)
	if want := time.Unix(1_700_000_000, 0).UTC(); !got.BooksLastModified.Equal(want) {
		t.Fatalf("unpadded token parsed as %v, want %v — devices may strip padding", got.BooksLastModified, want)
	}
}

// calibre-web writes total_seconds(), a float, so "1700000000.0" must parse.
func TestSyncTokenAcceptsFractionalEpoch(t *testing.T) {
	payload := `{"version":"1-1-0","data":{"books_last_modified":"1700000000.0"}}`
	got := ParseSyncToken(base64.StdEncoding.EncodeToString([]byte(payload)))
	if want := time.Unix(1_700_000_000, 0).UTC(); !got.BooksLastModified.Equal(want) {
		t.Fatalf("fractional epoch = %v, want %v", got.BooksLastModified, want)
	}
}

func TestCompareVersion(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"1-1-0", "1-0-0", 1},
		{"1-0-0", "1-1-0", -1},
		{"1-1-0", "1-1-0", 0},
		{"0-9-0", "1-0-0", -1},
		{"2", "1-9-9", 1},
	} {
		if got := compareVersion(tc.a, tc.b); got != tc.want {
			t.Errorf("compareVersion(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func mustEncodeVersion(t *testing.T, version string) string {
	t.Helper()
	payload, err := jsonx.Marshal(map[string]any{
		"version": version,
		"data":    map[string]string{"books_last_modified": "1700000000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(payload)
}
