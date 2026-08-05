package models

import (
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookSeriesEntity struct {
	SeriesID    string  `json:"series_id"`
	SeriesName  string  `json:"series_name"`
	SeriesIndex *string `json:"series_index,omitempty"`
}

type NextInSeriesEntity struct {
	SeriesID    string  `json:"series_id"`
	SeriesName  string  `json:"series_name"`
	BookID      string  `json:"book_id"`
	LibraryID   string  `json:"library_id"`
	Title       string  `json:"title"`
	CoverURL    *string `json:"cover_url,omitempty"`
	SeriesIndex *string `json:"series_index,omitempty"`
}

func BookSeriesFromSqlc(rows []sqlc.GetBookSeriesRow) []*BookSeriesEntity {
	out := make([]*BookSeriesEntity, 0, len(rows))
	for _, row := range rows {
		out = append(out, &BookSeriesEntity{
			SeriesID:    row.ID,
			SeriesName:  row.Name,
			SeriesIndex: convert.NullStringToStrPtr(row.SeriesIndex),
		})
	}
	return out
}

func (e *BookSeriesEntity) ToResponse() *response.BookSeriesResponse {
	if e == nil {
		return nil
	}
	return &response.BookSeriesResponse{
		SeriesID:    e.SeriesID,
		SeriesName:  e.SeriesName,
		SeriesIndex: e.SeriesIndex,
	}
}

func BookSeriesToResponse(entities []*BookSeriesEntity) []*response.BookSeriesResponse {
	out := make([]*response.BookSeriesResponse, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		out = append(out, entity.ToResponse())
	}
	return out
}

func (e *NextInSeriesEntity) ToResponse() *response.NextInSeriesResponse {
	if e == nil {
		return nil
	}
	return &response.NextInSeriesResponse{
		SeriesID:    e.SeriesID,
		SeriesName:  e.SeriesName,
		BookID:      e.BookID,
		Title:       e.Title,
		CoverURL:    e.CoverURL,
		SeriesIndex: e.SeriesIndex,
	}
}
