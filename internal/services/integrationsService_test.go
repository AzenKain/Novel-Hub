package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/netx"
)

type stubIntegrationsHighlightRepo struct {
	repositories.HighlightRepository
	highlights []*models.HighlightBookEntity
	err        error
}

func (s *stubIntegrationsHighlightRepo) GetHighlightsByBook(ctx context.Context, _ string, _ string) ([]*models.HighlightBookEntity, error) {
	return s.highlights, s.err
}

type stubIntegrationsTrackerRepo struct {
	repositories.TrackerRepository
	tracker *models.UserTrackerEntity
	err     error
}

func (s *stubIntegrationsTrackerRepo) GetUserTracker(ctx context.Context, _ string, _ string) (*models.UserTrackerEntity, error) {
	return s.tracker, s.err
}

type stubIntegrationsBookRepo struct {
	repositories.BookDBRepository
	book *models.BookEntity
}

func (s *stubIntegrationsBookRepo) GetBook(ctx context.Context, _ string) (*models.BookEntity, error) {
	return s.book, nil
}

type stubPermissions struct {
	PermissionCache
}

func (s *stubPermissions) CanRoles(roleIDs []string, roles []constants.RoleType, permission string, attrs map[string]any) bool {
	return true
}

func newTestIntegrationsService(highlightRepo repositories.HighlightRepository, trackerRepo repositories.TrackerRepository) IntegrationsService {
	book := &models.BookEntity{ID: "book-1", LibraryID: "lib-1"}
	return &integrationsService{
		highlightRepo: highlightRepo,
		trackerRepo:   trackerRepo,
		bookRepo:      &stubIntegrationsBookRepo{book: book},
		permissions:   &stubPermissions{},
		httpClient:    http.DefaultClient,
	}
}

func TestExportHighlightsMarkdown(t *testing.T) {
	created := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	note := "Keep this"
	cfi := "epubcfi(/6/4!/4/2)"
	highlights := []*models.HighlightBookEntity{
		{
			HighlightEntity: models.HighlightEntity{
				TextContent: "First passage",
				Note:        &note,
				CfiRange:    &cfi,
				CreatedAt:   &created,
			},
			BookTitle:  "Test Book",
			AuthorName: "Jane Doe",
		},
		{
			HighlightEntity: models.HighlightEntity{
				TextContent: "Second line\nwith a break",
			},
			BookTitle:  "Test Book",
			AuthorName: "Jane Doe",
		},
	}

	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{highlights: highlights}, &stubIntegrationsTrackerRepo{})
	out, err := s.ExportHighlightsMarkdown(context.Background(), "user-1", "book-1", nil)
	if err != nil {
		t.Fatalf("ExportHighlightsMarkdown: %v", err)
	}

	if !strings.HasPrefix(out, "# Test Book\n\n*Jane Doe*\n\n") {
		t.Fatalf("markdown missing title/author header:\n%s", out)
	}
	if !strings.Contains(out, "> First passage\n> Note: Keep this\n") {
		t.Fatalf("markdown missing highlight with note:\n%s", out)
	}
	if !strings.Contains(out, "> Second line\n> with a break\n") {
		t.Fatalf("markdown missing multi-line blockquote:\n%s", out)
	}
}

func TestExportHighlightsMarkdownEmpty(t *testing.T) {
	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{}, &stubIntegrationsTrackerRepo{})
	_, err := s.ExportHighlightsMarkdown(context.Background(), "user-1", "book-1", nil)
	if err == nil || !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestExportHighlightsToReadwisePayload(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	var gotBody map[string]any
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	old := readwiseAPIEndpoint
	readwiseAPIEndpoint = server.URL
	defer func() { readwiseAPIEndpoint = old }()

	created := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	note := "wow"
	cfi := "epubcfi(/6/4)"
	highlights := []*models.HighlightBookEntity{
		{
			HighlightEntity: models.HighlightEntity{
				TextContent: "A great line",
				Note:        &note,
				CfiRange:    &cfi,
				CreatedAt:   &created,
			},
			BookTitle:  "Test Book",
			AuthorName: "Jane Doe",
		},
	}
	tracker := &models.UserTrackerEntity{AccessToken: "rw-token-123"}

	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{highlights: highlights}, &stubIntegrationsTrackerRepo{tracker: tracker})
	count, err := s.ExportHighlightsToReadwise(context.Background(), "user-1", "book-1", nil)
	if err != nil {
		t.Fatalf("ExportHighlightsToReadwise: %v", err)
	}
	if count != 1 {
		t.Fatalf("exported = %d, want 1", count)
	}
	if gotAuth != "Token rw-token-123" {
		t.Fatalf("Authorization = %q, want 'Token rw-token-123'", gotAuth)
	}

	list, ok := gotBody["highlights"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("highlights payload = %#v", gotBody["highlights"])
	}
	item, _ := list[0].(map[string]any)
	if item["text"] != "A great line" || item["title"] != "Test Book" || item["author"] != "Jane Doe" {
		t.Fatalf("highlight item = %#v", item)
	}
	if item["location"] != "epubcfi(/6/4)" || item["location_type"] != "cfi" || item["note"] != "wow" {
		t.Fatalf("highlight location/note = %#v", item)
	}
	if item["source_type"] != "novelhub" || item["category"] != "books" {
		t.Fatalf("highlight source = %#v", item)
	}
}

func TestExportHighlightsToReadwiseNoToken(t *testing.T) {
	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{}, &stubIntegrationsTrackerRepo{})
	_, err := s.ExportHighlightsToReadwise(context.Background(), "user-1", "book-1", nil)
	if err == nil || !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestExportHighlightsToReadwiseAPIError(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
	}))
	defer server.Close()

	old := readwiseAPIEndpoint
	readwiseAPIEndpoint = server.URL
	defer func() { readwiseAPIEndpoint = old }()

	created := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	highlights := []*models.HighlightBookEntity{
		{
			HighlightEntity: models.HighlightEntity{TextContent: "x", CreatedAt: &created},
			BookTitle:       "B",
		},
	}
	tracker := &models.UserTrackerEntity{AccessToken: "bad-token"}

	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{highlights: highlights}, &stubIntegrationsTrackerRepo{tracker: tracker})
	if _, err := s.ExportHighlightsToReadwise(context.Background(), "user-1", "book-1", nil); err == nil || !errors.Is(err, apperrors.ErrInternalError) {
		t.Fatalf("expected ErrInternalError, got %v", err)
	}
}

func TestExportHighlightsAnki(t *testing.T) {
	created := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	note := "Important concept"
	highlights := []*models.HighlightBookEntity{
		{
			HighlightEntity: models.HighlightEntity{
				TextContent: "To be or not to be, that is the question.",
				Note:        &note,
				CreatedAt:   &created,
			},
			BookTitle:  "Hamlet",
			AuthorName: "William Shakespeare",
		},
	}

	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{highlights: highlights}, &stubIntegrationsTrackerRepo{})
	apkgBytes, bookTitle, err := s.ExportHighlightsAnki(context.Background(), "user-1", "book-1", nil)
	if err != nil {
		t.Fatalf("ExportHighlightsAnki failed: %v", err)
	}

	if len(apkgBytes) == 0 {
		t.Fatal("expected non-empty apkg bytes")
	}

	if bookTitle != "Hamlet" {
		t.Fatalf("expected bookTitle 'Hamlet', got '%s'", bookTitle)
	}
}

func TestExportHighlightsCSV(t *testing.T) {
	created := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	note := "Key vocabulary"
	highlights := []*models.HighlightBookEntity{
		{
			HighlightEntity: models.HighlightEntity{
				TextContent: "Serendipity means a fortunate stroke of luck.",
				Note:        &note,
				CreatedAt:   &created,
			},
			BookTitle:  "Vocabulary Builder",
			AuthorName: "John Doe",
		},
	}

	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{highlights: highlights}, &stubIntegrationsTrackerRepo{})
	csvStr, err := s.ExportHighlightsCSV(context.Background(), "user-1", "book-1", nil)
	if err != nil {
		t.Fatalf("ExportHighlightsCSV failed: %v", err)
	}

	if !strings.Contains(csvStr, "Serendipity") || !strings.Contains(csvStr, "Key vocabulary") {
		t.Fatalf("CSV missing card content:\n%s", csvStr)
	}
}

func TestExportHighlightsAnki_NotFound(t *testing.T) {
	s := newTestIntegrationsService(&stubIntegrationsHighlightRepo{highlights: nil}, &stubIntegrationsTrackerRepo{})
	_, _, err := s.ExportHighlightsAnki(context.Background(), "user-1", "book-1", nil)
	if err == nil || !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty highlights, got %v", err)
	}
}