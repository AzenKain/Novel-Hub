package models

import (
	"strconv"

	"novelhub/internal/gen/sqlc"
)

// sqlc types SUM/COALESCE/strftime columns as interface{} or sql.NullFloat64;
// these helpers normalise them to int64/string for the layer above.
func normalizeInt64(raw any) int64 {
	switch v := raw.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
}

func normalizeString(raw any) string {
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}

func nameCountEntities[T any](rows []T, name func(T) string, count func(T) int64) []*NameCountEntity {
	out := make([]*NameCountEntity, len(rows))
	for i, row := range rows {
		out[i] = nameCountEntity(row, name, count)
	}
	return out
}

type ReadingStatsByBookEntity struct {
	BookID        string `json:"book_id"`
	TotalDuration int64  `json:"total_duration"`
	TotalWords    int64  `json:"total_words"`
}

func (e *ReadingStatsByBookEntity) FromSqlc(r sqlc.GetReadingStatsByBookRow) *ReadingStatsByBookEntity {
	e.BookID = r.BookID
	e.TotalDuration = normalizeInt64(r.TotalDuration)
	e.TotalWords = normalizeInt64(r.TotalWords)
	return e
}

type ReadingStatsSinceEntity struct {
	TotalDuration int64 `json:"total_duration"`
	TotalWords    int64 `json:"total_words"`
}

func (e *ReadingStatsSinceEntity) FromSqlc(r sqlc.GetReadingStatsSinceRow) *ReadingStatsSinceEntity {
	e.TotalDuration = normalizeInt64(r.TotalDuration)
	e.TotalWords = normalizeInt64(r.TotalWords)
	return e
}

type ListeningHistoryEntity struct {
	Month        string `json:"month"`
	TotalSeconds int64  `json:"total_seconds"`
}

func (e *ListeningHistoryEntity) FromSqlc(r sqlc.GetListeningHistoryRow) *ListeningHistoryEntity {
	e.Month = normalizeString(r.Month)
	e.TotalSeconds = normalizeInt64(r.TotalDuration)
	return e
}

type ListeningHistoryEntities []*ListeningHistoryEntity

func (e *ListeningHistoryEntities) FromSqlc(rows []sqlc.GetListeningHistoryRow) []*ListeningHistoryEntity {
	out := make([]*ListeningHistoryEntity, len(rows))
	for i, row := range rows {
		out[i] = (&ListeningHistoryEntity{}).FromSqlc(row)
	}
	return out
}

type ListeningStatsEntity struct {
	TotalDuration int64 `json:"total_duration"`
	TotalWords    int64 `json:"total_words"`
}

func (e *ListeningStatsEntity) FromSqlc(r sqlc.GetListeningStatsRow) *ListeningStatsEntity {
	e.TotalDuration = normalizeInt64(r.TotalDuration)
	e.TotalWords = normalizeInt64(r.TotalWords)
	return e
}

type NameCountEntity struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type LibraryBreakdownEntity struct {
	Formats    []*NameCountEntity `json:"formats"`
	Tags       []*NameCountEntity `json:"tags"`
	Authors    []*NameCountEntity `json:"authors"`
	Publishers []*NameCountEntity `json:"publishers"`
	Languages  []*NameCountEntity `json:"languages"`
}

func nameCountEntity[T any](row T, name func(T) string, count func(T) int64) *NameCountEntity {
	return &NameCountEntity{Name: name(row), Count: count(row)}
}

func (e *LibraryBreakdownEntity) AddFormat(rows []sqlc.StatsByFormatRow) {
	e.Formats = nameCountEntities(rows, func(r sqlc.StatsByFormatRow) string { return r.Name }, func(r sqlc.StatsByFormatRow) int64 { return r.BookCount })
}

func (e *LibraryBreakdownEntity) AddTags(rows []sqlc.StatsByTagRow) {
	e.Tags = nameCountEntities(rows, func(r sqlc.StatsByTagRow) string { return r.Name }, func(r sqlc.StatsByTagRow) int64 { return r.BookCount })
}

func (e *LibraryBreakdownEntity) AddAuthors(rows []sqlc.StatsByAuthorRow) {
	e.Authors = nameCountEntities(rows, func(r sqlc.StatsByAuthorRow) string { return r.Name }, func(r sqlc.StatsByAuthorRow) int64 { return r.BookCount })
}

func (e *LibraryBreakdownEntity) AddPublishers(rows []sqlc.StatsByPublisherRow) {
	e.Publishers = nameCountEntities(rows, func(r sqlc.StatsByPublisherRow) string { return r.Name }, func(r sqlc.StatsByPublisherRow) int64 { return r.BookCount })
}

func (e *LibraryBreakdownEntity) AddLanguages(rows []sqlc.StatsByLanguageRow) {
	e.Languages = nameCountEntities(rows, func(r sqlc.StatsByLanguageRow) string { return r.Name }, func(r sqlc.StatsByLanguageRow) int64 { return r.BookCount })
}