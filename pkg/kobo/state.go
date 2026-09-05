package kobo

import (
	"strings"
	"time"
)

const (
	StatusReadyToRead = "ReadyToRead"
	StatusReading     = "Reading"
	StatusFinished    = "Finished"
)

type Location struct {
	Value  string `json:"Value"`
	Type   string `json:"Type"`
	Source string `json:"Source"`
}

type CurrentBookmark struct {
	LastModified                 string    `json:"LastModified"`
	ProgressPercent              *float64  `json:"ProgressPercent,omitempty"`
	ContentSourceProgressPercent *float64  `json:"ContentSourceProgressPercent,omitempty"`
	Location                     *Location `json:"Location,omitempty"`
}

// Statistics is the reading-time block.
type Statistics struct {
	LastModified         string `json:"LastModified"`
	SpentReadingMinutes  *int64 `json:"SpentReadingMinutes,omitempty"`
	RemainingTimeMinutes *int64 `json:"RemainingTimeMinutes,omitempty"`
}

type StatusInfo struct {
	LastModified           string `json:"LastModified"`
	Status                 string `json:"Status"`
	TimesStartedReading    int64  `json:"TimesStartedReading"`
	LastTimeStartedReading string `json:"LastTimeStartedReading,omitempty"`
}

type ReadingState struct {
	EntitlementID     string          `json:"EntitlementId"`
	Created           string          `json:"Created"`
	LastModified      string          `json:"LastModified"`
	PriorityTimestamp string          `json:"PriorityTimestamp"`
	StatusInfo        StatusInfo      `json:"StatusInfo"`
	Statistics        Statistics      `json:"Statistics"`
	CurrentBookmark   CurrentBookmark `json:"CurrentBookmark"`
}

type ReadingStateInput struct {
	BookUUID        string
	BookCreated     time.Time
	LastModified    time.Time
	ProgressPercent float64
	LocationValue   string
	LocationType    string
	LocationSource  string
	OpenedCount     int64
	LastOpenedAt    time.Time
}

func StatusFor(progressPercent float64, opened int64) string {
	switch {
	case progressPercent >= 100:
		return StatusFinished
	case progressPercent > 0 || opened > 0:
		return StatusReading
	default:
		return StatusReadyToRead
	}
}

func NewReadingState(in ReadingStateInput) ReadingState {
	lastModified := FormatTimestamp(in.LastModified)

	bookmark := CurrentBookmark{LastModified: lastModified}
	progress := in.ProgressPercent
	bookmark.ProgressPercent = &progress
	bookmark.ContentSourceProgressPercent = &progress
	if value := strings.TrimSpace(in.LocationValue); value != "" {
		locType := in.LocationType
		if locType == "" {
			locType = "KoboSpan"
		}
		source := in.LocationSource
		bookmark.Location = &Location{Value: value, Type: locType, Source: source}
	}

	status := StatusInfo{
		LastModified:        lastModified,
		Status:              StatusFor(in.ProgressPercent, in.OpenedCount),
		TimesStartedReading: in.OpenedCount,
	}
	if !in.LastOpenedAt.IsZero() {
		status.LastTimeStartedReading = FormatTimestamp(in.LastOpenedAt)
	}

	return ReadingState{
		EntitlementID:     in.BookUUID,
		Created:           FormatTimestamp(in.BookCreated),
		LastModified:      lastModified,
		PriorityTimestamp: lastModified,
		StatusInfo:        status,
		Statistics:        Statistics{LastModified: lastModified},
		CurrentBookmark:   bookmark,
	}
}

// PutStateSubResult is the per-block acknowledgement the device looks for.
type PutStateSubResult struct {
	Result string `json:"Result"`
}

// PutStateResult is one entry of the PUT response's UpdateResults.
type PutStateResult struct {
	EntitlementID         string             `json:"EntitlementId"`
	CurrentBookmarkResult *PutStateSubResult `json:"CurrentBookmarkResult,omitempty"`
	StatisticsResult      *PutStateSubResult `json:"StatisticsResult,omitempty"`
	StatusInfoResult      *PutStateSubResult `json:"StatusInfoResult,omitempty"`
	LastModified          string             `json:"LastModified"`
	PriorityTimestamp     string             `json:"PriorityTimestamp"`
}

// PutStateResponse is the full PUT response body.
type PutStateResponse struct {
	RequestResult string           `json:"RequestResult"`
	UpdateResults []PutStateResult `json:"UpdateResults"`
}
