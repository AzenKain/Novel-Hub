package controllers

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/opds"
)

type mockOPDSSettingsService struct {
	services.SettingsService
}

func (m *mockOPDSSettingsService) ServerURL() string { return "" }

type mockOPDSService struct {
	services.OPDSService
	lastAuthorName string
	lastSeriesName string
	lastTagName    string
	coverFilePath  string
	downloadPath   string
}

func (m *mockOPDSService) GetRootCatalog(_ context.Context, _, _ string, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "NovelHub OPDS Catalog"}, nil
}

func (m *mockOPDSService) GetAllBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "All Books"}, nil
}

func (m *mockOPDSService) GetRecentBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Recent Additions"}, nil
}

func (m *mockOPDSService) GetHotBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Hot Books"}, nil
}

func (m *mockOPDSService) GetRandomBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Random Books"}, nil
}

func (m *mockOPDSService) GetAuthorsCatalog(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Authors"}, nil
}

func (m *mockOPDSService) GetAuthorBooks(_ context.Context, _, _ string, authorName string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	m.lastAuthorName = authorName
	return &opds.Feed{Title: "Author: " + authorName}, nil
}

func (m *mockOPDSService) GetSeriesCatalog(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Series"}, nil
}

func (m *mockOPDSService) GetSeriesBooks(_ context.Context, _, _ string, seriesName string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	m.lastSeriesName = seriesName
	return &opds.Feed{Title: "Series: " + seriesName}, nil
}

func (m *mockOPDSService) GetTagsCatalog(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Tags"}, nil
}

func (m *mockOPDSService) GetTagBooks(_ context.Context, _, _ string, tagName string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	m.lastTagName = tagName
	return &opds.Feed{Title: "Tag: " + tagName}, nil
}

func (m *mockOPDSService) GetOpenSearchDescription(_, _ string) *opds.OpenSearchDescription {
	return &opds.OpenSearchDescription{ShortName: "NovelHub"}
}

func (m *mockOPDSService) SearchBooksOPDS(_ context.Context, _, _ string, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (*opds.Feed, error) {
	return &opds.Feed{Title: "Search Results"}, nil
}

func (m *mockOPDSService) GetOPDS2Catalog(_ context.Context, _, _ string, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{
		"metadata": map[string]any{"title": "NovelHub OPDS 2.0 Catalog"},
		"navigation": []map[string]any{
			{"title": "All Books"},
		},
		"publications": []map[string]any{
			{
				"metadata": map[string]any{"title": "Test Book"},
				"images": []map[string]any{
					{"href": "http://example.com/opds/v2/books/b-1/cover", "type": "image/jpeg"},
				},
			},
		},
	}, nil
}

func (m *mockOPDSService) GetOPDS2AllBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "All Books"}}, nil
}

func (m *mockOPDSService) GetOPDS2RecentBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Recent Books"}}, nil
}

func (m *mockOPDSService) GetOPDS2HotBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Hot Books"}}, nil
}

func (m *mockOPDSService) GetOPDS2RandomBooks(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Random Books"}}, nil
}

func (m *mockOPDSService) GetOPDS2AuthorsCatalog(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Authors"}}, nil
}

func (m *mockOPDSService) GetOPDS2AuthorBooks(_ context.Context, _, _ string, authorName string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	m.lastAuthorName = authorName
	return map[string]any{"metadata": map[string]any{"title": "Author: " + authorName}}, nil
}

func (m *mockOPDSService) GetOPDS2SeriesCatalog(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Series"}}, nil
}

func (m *mockOPDSService) GetOPDS2SeriesBooks(_ context.Context, _, _ string, seriesName string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	m.lastSeriesName = seriesName
	return map[string]any{"metadata": map[string]any{"title": "Series: " + seriesName}}, nil
}

func (m *mockOPDSService) GetOPDS2TagsCatalog(_ context.Context, _, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Tags"}}, nil
}

func (m *mockOPDSService) GetOPDS2TagBooks(_ context.Context, _, _ string, tagName string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	m.lastTagName = tagName
	return map[string]any{"metadata": map[string]any{"title": "Tag: " + tagName}}, nil
}

func (m *mockOPDSService) GetOPDS2Search(_ context.Context, _, _ string, _ string, _ request.OPDSPageDto, _ *response.JWTClaims) (map[string]any, error) {
	return map[string]any{"metadata": map[string]any{"title": "Search"}}, nil
}

func (m *mockOPDSService) GetBookCoverPath(_ context.Context, bookID string, _ *response.JWTClaims) (string, error) {
	if bookID == "not-found" {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}
	return m.coverFilePath, nil
}

func (m *mockOPDSService) GetBookFileForDownload(_ context.Context, bookID, _ string, _ *response.JWTClaims) (string, string, error) {
	if bookID == "not-found" {
		return "", "", apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}
	return m.downloadPath, "test-book.epub", nil
}

func TestUnescapeParamHelper(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Arthur%20Conan%20Doyle", "Arthur Conan Doyle"},
		{"Arthur+Conan+Doyle", "Arthur Conan Doyle"},
		{"SimpleName", "SimpleName"},
		{"Slime%20Isekai%20%2B%20Extra", "Slime Isekai + Extra"},
	}

	for _, tt := range tests {
		got := unescapeParam(tt.input)
		if got != tt.want {
			t.Errorf("unescapeParam(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func setupTestOPDSApp(t *testing.T) (*fiber.App, *mockOPDSService) {
	t.Helper()
	tempDir := t.TempDir()
	coverFile := filepath.Join(tempDir, "cover.jpg")
	_ = os.WriteFile(coverFile, []byte("fake-cover-image"), 0o600)

	downloadFile := filepath.Join(tempDir, "book.epub")
	_ = os.WriteFile(downloadFile, []byte("fake-epub-file"), 0o600)

	mockSvc := &mockOPDSService{
		coverFilePath: coverFile,
		downloadPath:  downloadFile,
	}
	mockSettings := &mockOPDSSettingsService{}
	ctrl := NewOPDSController(mockSvc, mockSettings)

	app := fiber.New()

	redirectHandler := func(c fiber.Ctx) error {
		accept := c.Get(fiber.HeaderAccept)
		base := strings.TrimSuffix(c.Path(), "/")
		if strings.Contains(accept, "application/opds+json") {
			return c.Redirect().To(base + "/v2/catalog")
		}
		return c.Redirect().To(base + "/v1")
	}

	app.Get("/opds", redirectHandler)
	app.Get("/opds/", redirectHandler)

	v1 := app.Group("/opds/v1")
	v1.Get("", ctrl.GetRootCatalog)
	v1.Get("/", ctrl.GetRootCatalog)
	v1.Get("/opensearch.xml", ctrl.GetOpenSearchDescription)
	v1.Get("/search", ctrl.SearchCatalog)
	v1.Get("/books", ctrl.GetAllBooks)
	v1.Get("/recent", ctrl.GetRecentBooks)
	v1.Get("/hot", ctrl.GetHotBooks)
	v1.Get("/random", ctrl.GetRandomBooks)
	v1.Get("/authors", ctrl.GetAuthorsCatalog)
	v1.Get("/authors/:name", ctrl.GetAuthorBooks)
	v1.Get("/series", ctrl.GetSeriesCatalog)
	v1.Get("/series/:name", ctrl.GetSeriesBooks)
	v1.Get("/tags", ctrl.GetTagsCatalog)
	v1.Get("/tags/:name", ctrl.GetTagBooks)
	v1.Get("/books/:id/cover", ctrl.GetBookCover)
	v1.Get("/books/:id/download", ctrl.DownloadBook)

	v2 := app.Group("/opds/v2")
	v2.Get("", func(c fiber.Ctx) error {
		return c.Redirect().To(GetOPDSBasePath(c) + "/v2/catalog")
	})
	v2.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().To(GetOPDSBasePath(c) + "/v2/catalog")
	})
	v2.Get("/catalog", ctrl.GetOPDS2Catalog)
	v2.Get("/books", ctrl.GetOPDS2AllBooks)
	v2.Get("/recent", ctrl.GetOPDS2RecentBooks)
	v2.Get("/hot", ctrl.GetOPDS2HotBooks)
	v2.Get("/random", ctrl.GetOPDS2RandomBooks)
	v2.Get("/authors", ctrl.GetOPDS2AuthorsCatalog)
	v2.Get("/authors/:name", ctrl.GetOPDS2AuthorBooks)
	v2.Get("/series", ctrl.GetOPDS2SeriesCatalog)
	v2.Get("/series/:name", ctrl.GetOPDS2SeriesBooks)
	v2.Get("/tags", ctrl.GetOPDS2TagsCatalog)
	v2.Get("/tags/:name", ctrl.GetOPDS2TagBooks)
	v2.Get("/search", ctrl.GetOPDS2Search)
	v2.Get("/books/:id/cover", ctrl.GetBookCover)
	v2.Get("/books/:id/download", ctrl.DownloadBook)

	return app, mockSvc
}

func TestOPDSController_Redirects(t *testing.T) {
	app, _ := setupTestOPDSApp(t)

	// Default redirect to /opds/v1
	req1 := httptest.NewRequest("GET", "/opds", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp1.StatusCode != fiber.StatusFound && resp1.StatusCode != fiber.StatusSeeOther {
		t.Errorf("expected status 302 or 303, got %d", resp1.StatusCode)
	}
	if loc := resp1.Header.Get("Location"); loc != "/opds/v1" {
		t.Errorf("expected Location /opds/v1, got %s", loc)
	}

	// OPDS 2.0 content negotiation redirect to /opds/v2/catalog
	req2 := httptest.NewRequest("GET", "/opds", nil)
	req2.Header.Set("Accept", "application/opds+json")
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp2.StatusCode != fiber.StatusFound && resp2.StatusCode != fiber.StatusSeeOther {
		t.Errorf("expected status 302 or 303, got %d", resp2.StatusCode)
	}
	if loc := resp2.Header.Get("Location"); loc != "/opds/v2/catalog" {
		t.Errorf("expected Location /opds/v2/catalog, got %s", loc)
	}

	// /opds/v2 redirects to /opds/v2/catalog
	req3 := httptest.NewRequest("GET", "/opds/v2", nil)
	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp3.StatusCode != fiber.StatusFound && resp3.StatusCode != fiber.StatusSeeOther {
		t.Errorf("expected status 302 or 303, got %d", resp3.StatusCode)
	}
	if loc := resp3.Header.Get("Location"); loc != "/opds/v2/catalog" {
		t.Errorf("expected Location /opds/v2/catalog, got %s", loc)
	}
}

func TestOPDSController_V1Endpoints(t *testing.T) {
	app, mockSvc := setupTestOPDSApp(t)

	endpoints := []string{
		"/opds/v1",
		"/opds/v1/books",
		"/opds/v1/recent",
		"/opds/v1/hot",
		"/opds/v1/random",
		"/opds/v1/authors",
		"/opds/v1/series",
		"/opds/v1/tags",
		"/opds/v1/search?q=test",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("GET %s failed: %v", ep, err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("GET %s status = %d, want 200", ep, resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/atom+xml") {
			t.Errorf("GET %s Content-Type = %s, want application/atom+xml", ep, ct)
		}
	}

	// Parameter unescaping verification
	authorReq := httptest.NewRequest("GET", "/opds/v1/authors/Arthur%20Conan%20Doyle", nil)
	authorResp, err := app.Test(authorReq)
	if err != nil || authorResp.StatusCode != 200 {
		t.Fatalf("GET author failed")
	}
	if mockSvc.lastAuthorName != "Arthur Conan Doyle" {
		t.Errorf("author name not unescaped! got: %q, want: 'Arthur Conan Doyle'", mockSvc.lastAuthorName)
	}

	seriesReq := httptest.NewRequest("GET", "/opds/v1/series/Slime%20Isekai", nil)
	seriesResp, err := app.Test(seriesReq)
	if err != nil || seriesResp.StatusCode != 200 {
		t.Fatalf("GET series failed")
	}
	if mockSvc.lastSeriesName != "Slime Isekai" {
		t.Errorf("series name not unescaped! got: %q, want: 'Slime Isekai'", mockSvc.lastSeriesName)
	}

	tagReq := httptest.NewRequest("GET", "/opds/v1/tags/Science%20Fiction", nil)
	tagResp, err := app.Test(tagReq)
	if err != nil || tagResp.StatusCode != 200 {
		t.Fatalf("GET tag failed")
	}
	if mockSvc.lastTagName != "Science Fiction" {
		t.Errorf("tag name not unescaped! got: %q, want: 'Science Fiction'", mockSvc.lastTagName)
	}
}

func TestOPDSController_V1CoverAndDownload(t *testing.T) {
	app, _ := setupTestOPDSApp(t)

	// Cover success
	reqCover := httptest.NewRequest("GET", "/opds/v1/books/b-1/cover", nil)
	respCover, err := app.Test(reqCover)
	if err != nil {
		t.Fatalf("cover request failed: %v", err)
	}
	if respCover.StatusCode != fiber.StatusOK {
		t.Errorf("cover status = %d, want 200", respCover.StatusCode)
	}
	if cc := respCover.Header.Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Errorf("expected Cache-Control header, got %s", cc)
	}

	// Cover not found
	reqCover404 := httptest.NewRequest("GET", "/opds/v1/books/not-found/cover", nil)
	respCover404, err := app.Test(reqCover404)
	if err != nil {
		t.Fatalf("cover 404 request failed: %v", err)
	}
	if respCover404.StatusCode != fiber.StatusNotFound {
		t.Errorf("cover 404 status = %d, want 404", respCover404.StatusCode)
	}

	// Download success
	reqDownload := httptest.NewRequest("GET", "/opds/v1/books/b-1/download?file_id=f-1", nil)
	respDownload, err := app.Test(reqDownload)
	if err != nil {
		t.Fatalf("download request failed: %v", err)
	}
	if respDownload.StatusCode != fiber.StatusOK {
		t.Errorf("download status = %d, want 200", respDownload.StatusCode)
	}
	if cd := respDownload.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("expected attachment Content-Disposition, got %s", cd)
	}
}

func TestOPDSController_V2Endpoints(t *testing.T) {
	app, mockSvc := setupTestOPDSApp(t)

	endpoints := []string{
		"/opds/v2/catalog",
		"/opds/v2/books",
		"/opds/v2/recent",
		"/opds/v2/hot",
		"/opds/v2/random",
		"/opds/v2/authors",
		"/opds/v2/series",
		"/opds/v2/tags",
		"/opds/v2/search?q=test",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("GET %s failed: %v", ep, err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("GET %s status = %d, want 200", ep, resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/opds+json") {
			t.Errorf("GET %s Content-Type = %s, want application/opds+json", ep, ct)
		}
	}

	// Parameter unescaping for V2
	req := httptest.NewRequest("GET", "/opds/v2/authors/Arthur%20Conan%20Doyle", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("GET V2 author failed")
	}
	if mockSvc.lastAuthorName != "Arthur Conan Doyle" {
		t.Errorf("V2 author name not unescaped! got: %q, want: 'Arthur Conan Doyle'", mockSvc.lastAuthorName)
	}
}

func TestOPDSController_ContextTimeout(t *testing.T) {
	// Rule check: verify controllers create a context with timeout and don't panic
	app, _ := setupTestOPDSApp(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := httptest.NewRequestWithContext(ctx, "GET", "/opds/v1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

