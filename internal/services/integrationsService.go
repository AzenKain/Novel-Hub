package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/anki"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

var readwiseAPIEndpoint = "https://readwise.io/api/v2/highlights/"

type IntegrationsService interface {
	ExportHighlightsToReadwise(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (int, error)
	ExportHighlightsMarkdown(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (string, error)
	ExportHighlightsAnki(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) ([]byte, string, error)
	ExportHighlightsCSV(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (string, error)
}

type integrationsService struct {
	highlightRepo repositories.HighlightRepository
	trackerRepo   repositories.TrackerRepository
	bookRepo      repositories.BookDBRepository
	permissions   PermissionCache
	httpClient    *http.Client
}

func NewIntegrationsService(highlightRepo repositories.HighlightRepository, trackerRepo repositories.TrackerRepository, bookRepo repositories.BookDBRepository, permissions PermissionCache) IntegrationsService {
	return &integrationsService{
		highlightRepo: highlightRepo,
		trackerRepo:   trackerRepo,
		bookRepo:      bookRepo,
		permissions:   permissions,
		httpClient:    netx.NewSafeHTTPClient(30 * time.Second),
	}
}

func (s *integrationsService) canExportBook(ctx context.Context, bookID string, claims *response.JWTClaims) bool {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return false
	}
	resolved := resolveClaims(claims)
	attrs := map[string]any{"library_id": book.LibraryID}
	return s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermBookRead, attrs) &&
		s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermBookHighlight, attrs)
}

func (s *integrationsService) ExportHighlightsToReadwise(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (int, error) {
	if !s.canExportBook(ctx, bookID, claims) {
		return 0, apperrors.New(apperrors.ErrForbidden, "Highlights are not allowed for this book")
	}

	tracker, err := s.trackerRepo.GetUserTracker(ctx, userID, "readwise")
	if err != nil || tracker == nil || tracker.AccessToken == "" {
		return 0, apperrors.New(apperrors.ErrBadRequest, "Readwise integration not connected")
	}

	highlights, err := s.highlightRepo.GetHighlightsByBook(ctx, userID, bookID)
	if err != nil {
		return 0, apperrors.New(apperrors.ErrInternalError, "Failed to load highlights")
	}
	if len(highlights) == 0 {
		return 0, apperrors.New(apperrors.ErrNotFound, "No highlights to export")
	}

	exported := 0
	for start := 0; start < len(highlights); start += 100 {
		end := min(start+100, len(highlights))
		if err := s.postReadwiseBatch(ctx, tracker.AccessToken, highlights[start:end]); err != nil {
			return exported, apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Readwise export failed: %v", err))
		}
		exported += end - start
	}
	return exported, nil
}

func (s *integrationsService) postReadwiseBatch(ctx context.Context, token string, highlights []*models.HighlightBookEntity) error {
	items := make([]map[string]any, 0, len(highlights))
	for _, h := range highlights {
		item := map[string]any{
			"text":           h.TextContent,
			"title":          h.BookTitle,
			"author":         h.AuthorName,
			"source_type":    "novelhub",
			"category":       "books",
			"location_type":  "cfi",
			"highlighted_at": h.CreatedAt.Format(time.RFC3339),
		}
		if h.CfiRange != nil {
			item["location"] = *h.CfiRange
		}
		if h.Note != nil {
			item["note"] = *h.Note
		}
		items = append(items, item)
	}
	payload := map[string]any{"highlights": items}

	bodyBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", readwiseAPIEndpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Readwise API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (s *integrationsService) ExportHighlightsMarkdown(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (string, error) {
	if !s.canExportBook(ctx, bookID, claims) {
		return "", apperrors.New(apperrors.ErrForbidden, "Highlights are not allowed for this book")
	}

	highlights, err := s.highlightRepo.GetHighlightsByBook(ctx, userID, bookID)
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to load highlights")
	}
	if len(highlights) == 0 {
		return "", apperrors.New(apperrors.ErrNotFound, "No highlights to export")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", highlights[0].BookTitle)
	if author := highlights[0].AuthorName; author != "" {
		fmt.Fprintf(&b, "*%s*\n\n", author)
	}
	for _, h := range highlights {
		text := strings.TrimSpace(h.TextContent)
		if text == "" {
			continue
		}
		for _, line := range strings.Split(text, "\n") {
			fmt.Fprintf(&b, "> %s\n", line)
		}
		if h.Note != nil && strings.TrimSpace(*h.Note) != "" {
			fmt.Fprintf(&b, "> Note: %s\n", strings.TrimSpace(*h.Note))
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (s *integrationsService) ExportHighlightsAnki(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) ([]byte, string, error) {
	if !s.canExportBook(ctx, bookID, claims) {
		return nil, "", apperrors.New(apperrors.ErrForbidden, "Highlights are not allowed for this book")
	}

	highlights, err := s.highlightRepo.GetHighlightsByBook(ctx, userID, bookID)
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrInternalError, "Failed to load highlights")
	}
	if len(highlights) == 0 {
		return nil, "", apperrors.New(apperrors.ErrNotFound, "No highlights to export")
	}

	bookTitle := highlights[0].BookTitle
	authorName := highlights[0].AuthorName

	cards := make([]anki.Flashcard, 0, len(highlights))
	for _, h := range highlights {
		front := strings.TrimSpace(h.TextContent)
		if front == "" {
			continue
		}
		back := ""
		if h.Note != nil {
			back = strings.TrimSpace(*h.Note)
		}
		contextStr := bookTitle
		if authorName != "" {
			contextStr += " - " + authorName
		}
		tags := []string{"NovelHub", sanitizeAnkiTag(bookTitle)}

		cards = append(cards, anki.Flashcard{
			Front:   front,
			Back:    back,
			Context: contextStr,
			Tags:    tags,
		})
	}

	deckName := fmt.Sprintf("NovelHub::%s", bookTitle)
	apkgBytes, err := anki.GenerateApkg(cards, anki.DeckOptions{
		DeckName:    deckName,
		Description: fmt.Sprintf("Exported from NovelHub: %s by %s", bookTitle, authorName),
	})
	if err != nil {
		return nil, "", apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to generate Anki deck: %v", err))
	}
	return apkgBytes, bookTitle, nil
}

func (s *integrationsService) ExportHighlightsCSV(ctx context.Context, userID string, bookID string, claims *response.JWTClaims) (string, error) {
	if !s.canExportBook(ctx, bookID, claims) {
		return "", apperrors.New(apperrors.ErrForbidden, "Highlights are not allowed for this book")
	}

	highlights, err := s.highlightRepo.GetHighlightsByBook(ctx, userID, bookID)
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to load highlights")
	}
	if len(highlights) == 0 {
		return "", apperrors.New(apperrors.ErrNotFound, "No highlights to export")
	}

	bookTitle := highlights[0].BookTitle
	authorName := highlights[0].AuthorName

	cards := make([]anki.Flashcard, 0, len(highlights))
	for _, h := range highlights {
		front := strings.TrimSpace(h.TextContent)
		if front == "" {
			continue
		}
		back := ""
		if h.Note != nil {
			back = strings.TrimSpace(*h.Note)
		}
		contextStr := bookTitle
		if authorName != "" {
			contextStr += " - " + authorName
		}
		tags := []string{"NovelHub", sanitizeAnkiTag(bookTitle)}

		cards = append(cards, anki.Flashcard{
			Front:   front,
			Back:    back,
			Context: contextStr,
			Tags:    tags,
		})
	}

	csvStr, err := anki.GenerateCSV(cards)
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("Failed to generate CSV: %v", err))
	}
	return csvStr, nil
}

func sanitizeAnkiTag(tag string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == ' ' || r == ':' || r == ';' || r == ',' {
			return '_'
		}
		return r
	}, tag)
	if cleaned == "" {
		return "Book"
	}
	return cleaned
}
