package models

type BookSearchSnippet struct {
	ChapterID    string `json:"chapter_id"`
	ChapterTitle string `json:"chapter_title"`
	ChapterIndex int64  `json:"chapter_index"`
	Snippet      string `json:"snippet"`
	Offset       int    `json:"offset"`
}
