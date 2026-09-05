package response

import (
	"testing"

	"novelhub/pkg/jsonx"
)

// The tracker search controller used to rebuild each result into a fiber.Map by hand, listing the four key names a second time.
func TestTrackerSearchResultKeepsItsWireKeys(t *testing.T) {
	raw, err := jsonx.Marshal(TrackerSearchResultResponse{
		ExternalSeriesID: "12345",
		TitleEnglish:     "Frieren",
		TitleRomaji:      "Sousou no Frieren",
		MediaType:        "MANGA",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := jsonx.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for key, want := range map[string]string{
		"external_series_id": "12345",
		"title_english":      "Frieren",
		"title_romaji":       "Sousou no Frieren",
		"media_type":         "MANGA",
	} {
		if got, ok := decoded[key]; !ok {
			t.Errorf("%s missing from the payload; the tracker picker cannot render it", key)
		} else if got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
	if len(decoded) != 4 {
		t.Errorf("payload carries %d keys, want exactly the 4 the client reads: %v", len(decoded), decoded)
	}
}
