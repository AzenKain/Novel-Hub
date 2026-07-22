package services

import (
	"context"
	"html"
	"regexp"
	"strings"

	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func (s *bookService) SearchInBook(ctx context.Context, bookID, query string) ([]*models.BookSearchSnippet, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "search query cannot be empty")
	}

	chapters, err := s.bookRepo.ListChaptersByBook(ctx, bookID)
	if err != nil {
		return nil, err
	}

	var results []*models.BookSearchSnippet
	lowerQueryRunes := []rune(strings.ToLower(query))
	if len(lowerQueryRunes) == 0 {
		return results, nil
	}

	for _, ch := range chapters {
		if ch.ContentPath == nil || *ch.ContentPath == "" {
			continue
		}
		htmlContent, err := s.GetChapterHTML(ctx, bookID, ch.ID, "")
		if err != nil || htmlContent == "" {
			continue
		}

		plainText := htmlTagRegex.ReplaceAllString(htmlContent, " ")
		plainText = html.UnescapeString(plainText)
		plainText = strings.Join(strings.Fields(plainText), " ")

		plainRunes := []rune(plainText)
		lowerRunes := []rune(strings.ToLower(plainText))

		for i := 0; i <= len(lowerRunes)-len(lowerQueryRunes); i++ {
			match := true
			for j := 0; j < len(lowerQueryRunes); j++ {
				if lowerRunes[i+j] != lowerQueryRunes[j] {
					match = false
					break
				}
			}
			if match {
				start := i - 40
				if start < 0 {
					start = 0
				}
				end := i + len(lowerQueryRunes) + 40
				if end > len(plainRunes) {
					end = len(plainRunes)
				}

				prefix := html.EscapeString(string(plainRunes[start:i]))
				matched := html.EscapeString(string(plainRunes[i : i+len(lowerQueryRunes)]))
				suffix := html.EscapeString(string(plainRunes[i+len(lowerQueryRunes) : end]))

				snippetHTML := prefix + "<mark class=\"bg-warning/40 text-warning-content font-bold px-0.5 rounded\">" + matched + "</mark>" + suffix

				results = append(results, &models.BookSearchSnippet{
					ChapterID:    ch.ID,
					ChapterTitle: ch.Title,
					ChapterIndex: ch.ChapterIndex,
					Snippet:      snippetHTML,
					Offset:       i,
				})

				i += len(lowerQueryRunes) - 1
				if len(results) >= 50 {
					break
				}
			}
		}
		if len(results) >= 50 {
			break
		}
	}

	return results, nil
}
