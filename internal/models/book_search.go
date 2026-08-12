package models

import (
	"html"
	"strings"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

const (
	ftsSnippetStart = "[NH_MARK_START]"
	ftsSnippetEnd   = "[NH_MARK_END]"
	searchMarkStart = `<mark class="bg-warning/40 text-warning-content font-bold px-0.5 rounded">`
)

type BookSearchSnippet struct {
	ChapterID    string `json:"chapter_id"`
	ChapterTitle string `json:"chapter_title"`
	ChapterIndex int64  `json:"chapter_index"`
	Snippet      string `json:"snippet"`
	Offset       int    `json:"offset"`
}

func (e *BookSearchSnippet) FromSqlc(row sqlc.SearchFTSInBookRow) *BookSearchSnippet {
	e.ChapterID = row.ChapterID
	e.ChapterTitle = row.ChapterTitle
	e.ChapterIndex = row.ChapterIndex
	e.Snippet = html.EscapeString(row.Snippet)
	e.Snippet = strings.ReplaceAll(e.Snippet, ftsSnippetStart, searchMarkStart)
	e.Snippet = strings.ReplaceAll(e.Snippet, ftsSnippetEnd, "</mark>")
	return e
}

func BookSearchSnippetsFromSqlc(rows []sqlc.SearchFTSInBookRow) []*BookSearchSnippet {
	results := make([]*BookSearchSnippet, len(rows))
	flat := make([]BookSearchSnippet, len(rows))
	for i, row := range rows {
		results[i] = flat[i].FromSqlc(row)
	}
	return results
}

func (e *BookSearchSnippet) ToResponse() *response.BookSearchSnippetResponse {
	if e == nil {
		return nil
	}
	return &response.BookSearchSnippetResponse{
		ChapterID:    e.ChapterID,
		ChapterTitle: e.ChapterTitle,
		ChapterIndex: e.ChapterIndex,
		Snippet:      e.Snippet,
		Offset:       e.Offset,
	}
}

func BookSearchSnippetsToResponse(entities []*BookSearchSnippet) []*response.BookSearchSnippetResponse {
	out := make([]*response.BookSearchSnippetResponse, 0, len(entities))
	for _, e := range entities {
		if e == nil {
			continue
		}
		out = append(out, e.ToResponse())
	}
	return out
}
