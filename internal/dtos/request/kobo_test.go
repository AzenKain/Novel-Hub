package request

import (
	"testing"

	"novelhub/pkg/jsonx"
)

const devicePutStateBody = `{
  "ReadingStates": [{
    "CurrentBookmark": {
      "ProgressPercent": 63,
      "ContentSourceProgressPercent": 61,
      "Location": {"Value": "kobo.11.2", "Type": "KoboSpan", "Source": "kepub"}
    },
    "Statistics": {"SpentReadingMinutes": 42, "RemainingTimeMinutes": 18},
    "StatusInfo": {"Status": "Reading"}
  }]
}`

func TestPutKoboStateDtoParsesDeviceBody(t *testing.T) {
	var dto PutKoboStateDto
	if err := jsonx.Unmarshal([]byte(devicePutStateBody), &dto); err != nil {
		t.Fatalf("unmarshal device body: %v", err)
	}
	if len(dto.ReadingStates) != 1 {
		t.Fatalf("ReadingStates has %d entries, want 1", len(dto.ReadingStates))
	}

	state := dto.ReadingStates[0]
	if state.CurrentBookmark == nil || state.CurrentBookmark.ProgressPercent == nil || *state.CurrentBookmark.ProgressPercent != 63 {
		t.Errorf("ProgressPercent not parsed: %#v", state.CurrentBookmark)
	}
	if state.CurrentBookmark.ContentSourceProgressPercent == nil || *state.CurrentBookmark.ContentSourceProgressPercent != 61 {
		t.Errorf("ContentSourceProgressPercent not parsed: %#v", state.CurrentBookmark)
	}
	if state.CurrentBookmark.Location == nil || state.CurrentBookmark.Location.Value != "kobo.11.2" {
		t.Errorf("Location not parsed: %#v", state.CurrentBookmark.Location)
	}
	if state.StatusInfo == nil || state.StatusInfo.Status != "Reading" {
		t.Errorf("StatusInfo not parsed: %#v", state.StatusInfo)
	}
	if state.Statistics == nil || state.Statistics.SpentReadingMinutes == nil || *state.Statistics.SpentReadingMinutes != 42 {
		t.Errorf("Statistics not parsed: %#v", state.Statistics)
	}
}

// Devices omit blocks they have nothing for.
func TestPutKoboStateDtoTreatsNullBlocksAsAbsent(t *testing.T) {
	const body = `{"ReadingStates":[{"CurrentBookmark":null,"Statistics":null,"StatusInfo":null}]}`
	var dto PutKoboStateDto
	if err := jsonx.Unmarshal([]byte(body), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dto.ReadingStates) != 1 {
		t.Fatalf("ReadingStates has %d entries, want 1", len(dto.ReadingStates))
	}
	state := dto.ReadingStates[0]
	if state.CurrentBookmark != nil || state.Statistics != nil || state.StatusInfo != nil {
		t.Error("null blocks must decode to nil so they can be skipped")
	}
}

// A bookmark with no Location at all: the device knows the percentage but not the span.
func TestPutKoboStateDtoBookmarkWithoutLocation(t *testing.T) {
	const body = `{"ReadingStates":[{"CurrentBookmark":{"ProgressPercent":10}}]}`
	var dto PutKoboStateDto
	if err := jsonx.Unmarshal([]byte(body), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bookmark := dto.ReadingStates[0].CurrentBookmark
	if bookmark == nil || bookmark.ProgressPercent == nil || *bookmark.ProgressPercent != 10 {
		t.Fatalf("ProgressPercent not parsed: %#v", bookmark)
	}
	if bookmark.Location != nil {
		t.Errorf("Location = %#v, want nil when the device did not send one", bookmark.Location)
	}
	if bookmark.ContentSourceProgressPercent != nil {
		t.Errorf("ContentSourceProgressPercent = %v, want nil (absent), not 0", *bookmark.ContentSourceProgressPercent)
	}
}
