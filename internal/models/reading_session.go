package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type ReadingSessionEntity struct {
	ID              string     `json:"id"`
	UserID          int64      `json:"userId"`
	BookID          string     `json:"bookId"`
	DurationSeconds int64      `json:"durationSeconds"`
	WordsRead       int64      `json:"wordsRead"`
	Date            time.Time  `json:"date"`
	CreatedAt       *time.Time `json:"createdAt"`
	UpdatedAt       *time.Time `json:"updatedAt"`
}

func (e *ReadingSessionEntity) FromSqlc(res sqlc.ReadingSession) *ReadingSessionEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.DurationSeconds = res.DurationSeconds
	e.WordsRead = res.WordsRead
	e.Date = res.SessionDate
	e.CreatedAt = convert.NullTimeToTimePtr(res.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

func (e *ReadingSessionEntity) ToResponse() *response.ReadingSessionResponse {
	if e == nil {
		return nil
	}
	var createdAt, updatedAt time.Time
	if e.CreatedAt != nil {
		createdAt = *e.CreatedAt
	}
	if e.UpdatedAt != nil {
		updatedAt = *e.UpdatedAt
	}
	return &response.ReadingSessionResponse{
		ID:              e.ID,
		UserID:          e.UserID,
		BookID:          e.BookID,
		DurationSeconds: e.DurationSeconds,
		WordsRead:       e.WordsRead,
		Date:            e.Date,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

type ReadingHeatmapEntity struct {
	Date            time.Time `json:"date"`
	DurationSeconds int64     `json:"durationSeconds"`
	WordsRead       int64     `json:"wordsRead"`
}

func (e *ReadingHeatmapEntity) FromSqlc(res sqlc.GetReadingHeatmapRow) *ReadingHeatmapEntity {
	e.Date = res.SessionDate
	e.DurationSeconds = int64(res.TotalDuration.Float64)
	e.WordsRead = int64(res.TotalWords.Float64)
	return e
}

type ReadingHeatmapEntities []*ReadingHeatmapEntity

func (e *ReadingHeatmapEntities) FromSqlc(rows []sqlc.GetReadingHeatmapRow) []*ReadingHeatmapEntity {
	slice := make([]*ReadingHeatmapEntity, len(rows))
	flat := make([]ReadingHeatmapEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *ReadingHeatmapEntity) ToResponse() *response.ReadingHeatmapResponse {
	if e == nil {
		return nil
	}
	return &response.ReadingHeatmapResponse{
		Date:            e.Date,
		DurationSeconds: e.DurationSeconds,
		WordsRead:       e.WordsRead,
	}
}

func ReadingHeatmapEntitiesToResponse(entities []*ReadingHeatmapEntity) []*response.ReadingHeatmapResponse {
	out := make([]*response.ReadingHeatmapResponse, 0, len(entities))
	for _, h := range entities {
		if h == nil {
			continue
		}
		out = append(out, h.ToResponse())
	}
	return out
}
