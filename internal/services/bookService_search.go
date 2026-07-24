package services

import (
	"context"
	"strings"
	"unicode"

	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
)

func normalizeFTSQuery(query string) string {
	query = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, query)
	query = strings.Join(strings.Fields(query), " ")
	if query == "" {
		return ""
	}
	return `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
}

func (s *bookService) SearchInBook(ctx context.Context, bookID, query string) ([]*models.BookSearchSnippet, error) {
	query = normalizeFTSQuery(query)
	if query == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "search query cannot be empty")
	}
	return s.bookRepo.SearchFTSInBook(ctx, bookID, query)
}
