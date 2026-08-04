package kobo

import (
	"strings"
	"testing"
	"time"

	"novelhub/pkg/jsonx"
)

// The assertions here are contract assertions, not behaviour assertions: there is no Kobo
// device in CI, so the only thing that can be checked is that the bytes on the wire match the
// shape calibre-web sends. Every expected value below was read out of calibre-web's source
// (cps/kobo.py), which is the implementation validated against real hardware.

// resourceString reads one entry as a URL string. Lives in the test rather than the package:
// production code passes the whole map straight to the response, so nothing outside these
// assertions ever needs one key as a string.
func resourceString(t *testing.T, res map[string]any, key string) string {
	t.Helper()
	value, _ := res[key].(string)
	return value
}

func TestResourcesHasFullNativeMap(t *testing.T) {
	res := Resources("https://books.example.com/kobo/deadbeef")
	// calibre-web's NATIVE_KOBO_RESOURCES() has 147 entries. A device derives every URL it
	// calls from this map, so a short map silently disables device features.
	if len(res) != 147 {
		t.Fatalf("resource map has %d keys, want 147", len(res))
	}
	// A key that must still point at the real store — rewriting everything would break the
	// device's shop and account pages.
	if got := resourceString(t, res, "account_page"); got != "https://www.kobo.com/account/settings" {
		t.Errorf("account_page = %q, want the upstream Kobo URL", got)
	}
}

// Two entries of the native map are nested objects, not URL strings. Decoding the file into
// map[string]string silently dropped all 147 keys, so this pins the type.
func TestResourcesKeepsNestedObjectEntries(t *testing.T) {
	res := Resources("https://books.example.com/kobo/deadbeef")
	for _, key := range []string{"blackstone_header", "free_books_page"} {
		if _, ok := res[key].(map[string]any); !ok {
			t.Errorf("%s = %#v, want a nested object", key, res[key])
		}
	}
}

func TestResourcesRewritesOnlyTheSelfHostedKeys(t *testing.T) {
	endpoint := "https://books.example.com/kobo/deadbeef"
	res := Resources(endpoint)

	if got, want := resourceString(t, res, "library_sync"), endpoint+"/v1/library/sync"; got != want {
		t.Errorf("library_sync = %q, want %q", got, want)
	}
	// image_host is the bare origin: the device joins it with the templates itself.
	if got, want := resourceString(t, res, "image_host"), "https://books.example.com"; got != want {
		t.Errorf("image_host = %q, want %q", got, want)
	}
	// Placeholders must survive verbatim — the device substitutes them, we must not.
	for _, key := range []string{"image_url_template", "image_url_quality_template"} {
		tmpl := resourceString(t, res, key)
		if !strings.HasPrefix(tmpl, endpoint+"/") {
			t.Errorf("%s = %q, want it rooted at the token endpoint", key, tmpl)
		}
		for _, ph := range []string{"{ImageId}", "{width}", "{height}"} {
			if !strings.Contains(tmpl, ph) {
				t.Errorf("%s = %q, missing placeholder %s", key, tmpl, ph)
			}
		}
	}
	if !strings.Contains(resourceString(t, res, "image_url_quality_template"), "{Quality}") {
		t.Error("image_url_quality_template must keep the {Quality} placeholder")
	}
	if strings.Contains(resourceString(t, res, "image_url_template"), "{Quality}") {
		t.Error("image_url_template is the no-Quality variant; it must not contain {Quality}")
	}
}

func TestResourcesTrailingSlashDoesNotDoubleUp(t *testing.T) {
	res := Resources("https://books.example.com/kobo/deadbeef/")
	if got := resourceString(t, res, "library_sync"); strings.Contains(got, "//v1") {
		t.Errorf("library_sync = %q, trailing slash leaked into the path", got)
	}
}

func TestFormatTimestampMatchesKoboFormat(t *testing.T) {
	ts := time.Date(2026, 3, 14, 15, 9, 26, 500, time.UTC)
	if got, want := FormatTimestamp(ts), "2026-03-14T15:09:26Z"; got != want {
		t.Errorf("FormatTimestamp = %q, want %q", got, want)
	}
	// A zero time must not serialise as year 1: calibre-web substitutes now for exactly this.
	if got := FormatTimestamp(time.Time{}); strings.HasPrefix(got, "0001-") {
		t.Errorf("FormatTimestamp(zero) = %q, want a current timestamp", got)
	}
}

func TestFormatTimestampConvertsToUTC(t *testing.T) {
	zone := time.FixedZone("UTC+7", 7*3600)
	ts := time.Date(2026, 3, 14, 22, 0, 0, 0, zone)
	if got, want := FormatTimestamp(ts), "2026-03-14T15:00:00Z"; got != want {
		t.Errorf("FormatTimestamp = %q, want %q — the Z suffix promises UTC", got, want)
	}
}

func sampleBook() BookInfo {
	desc := "A book."
	return BookInfo{
		UUID:         "0195f2a1-0000-7000-8000-000000000001",
		Title:        "Test Book",
		Description:  &desc,
		Authors:      []string{"Ada Lovelace", "Alan Turing"},
		Publisher:    "NovelHub Press",
		Language:     "vi",
		SeriesName:   "Test Series",
		SeriesIndex:  2,
		Created:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LastModified: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		PublishedAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestBookEntitlementFieldSet(t *testing.T) {
	book := sampleBook()
	raw, err := jsonx.Marshal(NewBookEntitlement(book))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := jsonx.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// The exact 12 keys calibre-web's create_book_entitlement() emits.
	want := []string{
		"Accessibility", "ActivePeriod", "Created", "CrossRevisionId", "Id", "IsRemoved",
		"IsHiddenFromArchive", "IsLocked", "LastModified", "OriginCategory", "RevisionId",
		"Status",
	}
	for _, key := range want {
		if _, ok := got[key]; !ok {
			t.Errorf("BookEntitlement missing %s", key)
		}
	}
	if len(got) != len(want) {
		t.Errorf("BookEntitlement has %d keys, want exactly %d", len(got), len(want))
	}

	if got["Accessibility"] != "Full" {
		t.Errorf("Accessibility = %v, want Full", got["Accessibility"])
	}
	if got["OriginCategory"] != "Imported" {
		t.Errorf("OriginCategory = %v, want Imported", got["OriginCategory"])
	}
	if got["Status"] != "Active" {
		t.Errorf("Status = %v, want Active", got["Status"])
	}
	// Id, CrossRevisionId and RevisionId are all the book UUID — the device correlates them.
	for _, key := range []string{"Id", "CrossRevisionId", "RevisionId"} {
		if got[key] != book.UUID {
			t.Errorf("%s = %v, want the book UUID %s", key, got[key], book.UUID)
		}
	}
	if got["Created"] != "2026-01-01T00:00:00Z" {
		t.Errorf("Created = %v, want the book's created timestamp", got["Created"])
	}
}

func TestBookEntitlementArchivedSetsIsRemoved(t *testing.T) {
	book := sampleBook()
	book.Archived = true
	if !NewBookEntitlement(book).IsRemoved {
		t.Error("IsRemoved must be true for an archived book, otherwise the device keeps it")
	}
}

func TestBookMetadataFieldSet(t *testing.T) {
	book := sampleBook()
	raw, err := jsonx.Marshal(NewBookMetadata(book))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := jsonx.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Keys calibre-web's get_metadata() always emits.
	for _, key := range []string{
		"Categories", "CoverImageId", "CrossRevisionId", "CurrentDisplayPrice",
		"CurrentLoveDisplayPrice", "Description", "DownloadUrls", "EntitlementId",
		"ExternalIds", "Genre", "IsEligibleForKoboLove", "IsInternetArchive", "IsPreOrder",
		"IsSocialEnabled", "Language", "PhoneticPronunciations", "PublicationDate",
		"Publisher", "RevisionId", "Title", "WorkId",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("BookMetadata missing %s", key)
		}
	}

	// The UUID reuse the device relies on to tie metadata to its entitlement.
	for _, key := range []string{"CoverImageId", "CrossRevisionId", "EntitlementId", "RevisionId", "WorkId"} {
		if got[key] != book.UUID {
			t.Errorf("%s = %v, want the book UUID", key, got[key])
		}
	}

	if got["Language"] != "vi" {
		t.Errorf("Language = %v, want vi", got["Language"])
	}
	if pub, ok := got["Publisher"].(map[string]any); !ok {
		t.Error("Publisher must be an object with Imprint and Name")
	} else {
		if pub["Name"] != "NovelHub Press" {
			t.Errorf("Publisher.Name = %v", pub["Name"])
		}
		if _, has := pub["Imprint"]; !has {
			t.Error("Publisher.Imprint must be present even when empty")
		}
	}
	// Empty slices must serialise as [] not null: a null here has been seen to abort parsing.
	if _, ok := got["ExternalIds"].([]any); !ok {
		t.Errorf("ExternalIds = %#v, want an empty array", got["ExternalIds"])
	}
	if _, ok := got["DownloadUrls"].([]any); !ok {
		t.Errorf("DownloadUrls = %#v, want an array", got["DownloadUrls"])
	}
	if _, ok := got["PhoneticPronunciations"].(map[string]any); !ok {
		t.Errorf("PhoneticPronunciations = %#v, want an object", got["PhoneticPronunciations"])
	}
}

func TestBookMetadataContributors(t *testing.T) {
	meta := NewBookMetadata(sampleBook())
	if len(meta.Contributors) != 2 || meta.Contributors[0] != "Ada Lovelace" {
		t.Errorf("Contributors = %v, want both author names in order", meta.Contributors)
	}
	if len(meta.ContributorRoles) != 2 || meta.ContributorRoles[1].Name != "Alan Turing" {
		t.Errorf("ContributorRoles = %v, want one {Name} object per author", meta.ContributorRoles)
	}
}

func TestBookMetadataOmitsSeriesWhenAbsent(t *testing.T) {
	book := sampleBook()
	book.SeriesName = ""
	if NewBookMetadata(book).Series != nil {
		t.Error("Series must be omitted entirely for a standalone book")
	}
}

func TestBookMetadataSeriesIDIsStableAndMatchesUUID3(t *testing.T) {
	meta := NewBookMetadata(sampleBook())
	if meta.Series == nil {
		t.Fatal("Series missing")
	}
	// uuid3(NAMESPACE_DNS, "Test Series") computed with Python's uuid module — the value
	// calibre-web would produce for the same series name.
	const want = "6f6b0366-32e3-310e-81a5-73826405caa4"
	if meta.Series.ID != want {
		t.Errorf("Series.Id = %q, want %q (uuid3 of the series name)", meta.Series.ID, want)
	}
	if meta.Series.Number != 2 || meta.Series.NumberFloat != 2 {
		t.Errorf("Series number = %d/%v, want 2/2", meta.Series.Number, meta.Series.NumberFloat)
	}
}

func TestBookMetadataSeriesIndexZeroFallsBackToOne(t *testing.T) {
	book := sampleBook()
	book.SeriesIndex = 0
	meta := NewBookMetadata(book)
	if meta.Series.Number != 1 || meta.Series.NumberFloat != 1 {
		t.Errorf("series index 0 became %d/%v, want 1/1 like calibre-web", meta.Series.Number, meta.Series.NumberFloat)
	}
}

func TestBookMetadataDefaultsLanguage(t *testing.T) {
	book := sampleBook()
	book.Language = "  "
	if got := NewBookMetadata(book).Language; got != "en" {
		t.Errorf("Language = %q, want the en fallback", got)
	}
}

func TestBookDownloadURL(t *testing.T) {
	got := BookDownloadURL("https://books.example.com/kobo/deadbeef", "book-1", "EPUB", "EPUB3", 4096)
	if got.URL != "https://books.example.com/kobo/deadbeef/download/book-1/epub" {
		t.Errorf("URL = %q", got.URL)
	}
	// Format is what the device is told; the URL carries the stored format, lowercased.
	if got.Format != "EPUB3" {
		t.Errorf("Format = %q, want the Kobo format name", got.Format)
	}
	if got.Platform != "Generic" {
		t.Errorf("Platform = %q, want Generic", got.Platform)
	}
	if got.Size != 4096 {
		t.Errorf("Size = %d", got.Size)
	}
}

func TestKoboFormatsMapping(t *testing.T) {
	// Straight from calibre-web's KOBO_FORMATS. EPUB advertises two names; the device picks.
	if got := KoboFormats["EPUB"]; len(got) != 2 || got[0] != "EPUB3" || got[1] != "EPUB" {
		t.Errorf("KoboFormats[EPUB] = %v, want [EPUB3 EPUB]", got)
	}
	if got := KoboFormats["KEPUB"]; len(got) != 1 || got[0] != "KEPUB" {
		t.Errorf("KoboFormats[KEPUB] = %v, want [KEPUB]", got)
	}
	if _, ok := KoboFormats["PDF"]; ok {
		t.Error("PDF must not be offered: the device cannot open it through the sync API")
	}
}

func TestStatusFor(t *testing.T) {
	cases := []struct {
		progress float64
		opened   int64
		want     string
	}{
		{0, 0, StatusReadyToRead},
		{0, 1, StatusReading}, // opened but no progress recorded yet
		{42, 3, StatusReading},
		{100, 5, StatusFinished},
		{100, 0, StatusFinished},
	}
	for _, tc := range cases {
		if got := StatusFor(tc.progress, tc.opened); got != tc.want {
			t.Errorf("StatusFor(%v, %d) = %q, want %q", tc.progress, tc.opened, got, tc.want)
		}
	}
}

func TestNewReadingStateShape(t *testing.T) {
	in := ReadingStateInput{
		BookUUID:        "0195f2a1-0000-7000-8000-000000000001",
		BookCreated:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LastModified:    time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
		ProgressPercent: 37,
		LocationValue:   "kobo.7.1",
		LocationType:    "KoboSpan",
		OpenedCount:     4,
		LastOpenedAt:    time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC),
	}
	state := NewReadingState(in)

	if state.EntitlementID != in.BookUUID {
		t.Errorf("EntitlementId = %q", state.EntitlementID)
	}
	// PriorityTimestamp tracks LastModified — calibre-web notes they are always equal.
	if state.PriorityTimestamp != state.LastModified {
		t.Errorf("PriorityTimestamp %q != LastModified %q", state.PriorityTimestamp, state.LastModified)
	}
	if state.LastModified != "2026-02-01T12:00:00Z" {
		t.Errorf("LastModified = %q", state.LastModified)
	}
	if state.StatusInfo.Status != StatusReading {
		t.Errorf("Status = %q, want Reading at 37%%", state.StatusInfo.Status)
	}
	if state.StatusInfo.TimesStartedReading != 4 {
		t.Errorf("TimesStartedReading = %d, want the opened count", state.StatusInfo.TimesStartedReading)
	}
	if state.CurrentBookmark.ProgressPercent == nil || *state.CurrentBookmark.ProgressPercent != 37 {
		t.Errorf("ProgressPercent = %v, want 37", state.CurrentBookmark.ProgressPercent)
	}
	if state.CurrentBookmark.Location == nil || state.CurrentBookmark.Location.Value != "kobo.7.1" {
		t.Errorf("Location = %#v, want the stored CFI", state.CurrentBookmark.Location)
	}
}

func TestNewReadingStateOmitsLocationWhenUnset(t *testing.T) {
	state := NewReadingState(ReadingStateInput{BookUUID: "b", ProgressPercent: 10})
	if state.CurrentBookmark.Location != nil {
		t.Error("Location must be omitted when there is no stored position")
	}
	raw, err := jsonx.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), `"Location"`) {
		t.Errorf("Location key leaked into JSON: %s", raw)
	}
	// Statistics has no data either, so only LastModified may be present.
	if strings.Contains(string(raw), "SpentReadingMinutes") {
		t.Error("SpentReadingMinutes must be omitted when unknown")
	}
}

func TestNewReadingStateDefaultsLocationType(t *testing.T) {
	state := NewReadingState(ReadingStateInput{BookUUID: "b", LocationValue: "pos-1"})
	if state.CurrentBookmark.Location.Type != "KoboSpan" {
		t.Errorf("Location.Type = %q, want the KoboSpan default", state.CurrentBookmark.Location.Type)
	}
}

// The request body these responses answer is request.PutKoboStateDto; its parsing is covered by
// internal/dtos/request/kobo_test.go.

func TestPutStateResponseOmitsUnrequestedResults(t *testing.T) {
	resp := PutStateResponse{
		RequestResult: "Success",
		UpdateResults: []PutStateResult{{
			EntitlementID:         "book-1",
			CurrentBookmarkResult: &PutStateSubResult{Result: "Success"},
			LastModified:          "2026-02-01T12:00:00Z",
			PriorityTimestamp:     "2026-02-01T12:00:00Z",
		}},
	}
	raw, err := jsonx.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, `"CurrentBookmarkResult"`) {
		t.Errorf("bookmark result missing: %s", body)
	}
	// calibre-web only adds a sub-result for a block the request contained.
	if strings.Contains(body, "StatisticsResult") || strings.Contains(body, "StatusInfoResult") {
		t.Errorf("unrequested sub-results leaked: %s", body)
	}
	if !strings.Contains(body, `"RequestResult":"Success"`) {
		t.Errorf("RequestResult missing or wrong: %s", body)
	}
}
