package controllers

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
)

type mockCalibreServerService struct {
	services.CalibreServerService
	filePath string
}

func (m *mockCalibreServerService) GetLibraryInfo(_ context.Context, _ *response.JWTClaims) (*response.CalibreLibraryInfoResponse, error) {
	return &response.CalibreLibraryInfoResponse{
		LibraryMap:     map[string]string{"1": "NovelHub Library"},
		DefaultLibrary: "1",
	}, nil
}

func (m *mockCalibreServerService) GetCategories(_ context.Context, _ string, _ *response.JWTClaims) (map[string]*response.CalibreCategorySummary, error) {
	return map[string]*response.CalibreCategorySummary{
		"allbooks": {Name: "All books", Count: 10},
		"authors":  {Name: "Authors", Count: 5},
	}, nil
}

func (m *mockCalibreServerService) GetCategory(_ context.Context, _, categoryName string, _, _ int64, _, _ string, _ *response.JWTClaims) (*response.CalibreCategoryDetailResponse, error) {
	return &response.CalibreCategoryDetailResponse{
		CategoryName: categoryName,
		TotalNum:     1,
		Items: []response.CalibreCategoryItem{
			{Name: "Arthur Conan Doyle", Count: 1},
		},
	}, nil
}

func (m *mockCalibreServerService) GetBooksInCategory(_ context.Context, _, _, _ string, _, _ int64, _, _ string, _ *response.JWTClaims) (*response.CalibreBooksInResponse, error) {
	return &response.CalibreBooksInResponse{
		TotalNum: 1,
		BookIDs:  []string{"book-1"},
	}, nil
}

func (m *mockCalibreServerService) SearchBooks(_ context.Context, _, query string, _, _ int64, _, _ string, _ *response.JWTClaims) (*response.CalibreSearchResponse, error) {
	return &response.CalibreSearchResponse{
		TotalNum: 1,
		Query:    query,
		BookIDs:  []string{"book-1"},
	}, nil
}

func (m *mockCalibreServerService) GetBooksMetadata(_ context.Context, _ string, bookIDs []string, _ *response.JWTClaims) (map[string]*response.CalibreBookMetadataResponse, error) {
	return map[string]*response.CalibreBookMetadataResponse{
		"book-1": {Title: "The Sign of Four"},
	}, nil
}

func (m *mockCalibreServerService) GetBookMetadata(_ context.Context, _, bookID string, _ *response.JWTClaims) (*response.CalibreBookMetadataResponse, error) {
	if bookID == "notfound" {
		return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	return &response.CalibreBookMetadataResponse{Title: "The Sign of Four"}, nil
}

func (m *mockCalibreServerService) GetBookCover(_ context.Context, bookID string, _ bool, _ *response.JWTClaims) (string, error) {
	if bookID == "notfound" {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}
	return m.filePath, nil
}

func (m *mockCalibreServerService) GetBookFile(_ context.Context, bookID string, format string, _ *response.JWTClaims) (string, string, error) {
	if bookID == "notfound" {
		return "", "", apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	return m.filePath, "book.epub", nil
}

func setupCalibreControllerApp(t *testing.T) (*fiber.App, string) {
	tmpDir := t.TempDir()
	sampleFile := filepath.Join(tmpDir, "sample.epub")
	if err := os.WriteFile(sampleFile, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	svc := &mockCalibreServerService{filePath: sampleFile}
	ctrl := NewCalibreServerController(svc)

	app := fiber.New()
	app.Get("/calibre/ajax/library-info", ctrl.GetLibraryInfo)
	app.Get("/calibre/ajax/categories", ctrl.GetCategories)
	app.Get("/calibre/ajax/category/:encoded_name", ctrl.GetCategory)
	app.Get("/calibre/ajax/books_in/:encoded_category/:encoded_item", ctrl.GetBooksInCategory)
	app.Get("/calibre/ajax/search", ctrl.Search)
	app.Get("/calibre/ajax/books", ctrl.GetBooks)
	app.Get("/calibre/ajax/book/:book_id", ctrl.GetBook)
	app.Get("/calibre/get/:what/:book_id", ctrl.GetContent)

	return app, sampleFile
}

func TestCalibreServerController_GetLibraryInfo(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)
	req := httptest.NewRequest("GET", "/calibre/ajax/library-info", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}
}

func TestCalibreServerController_GetCategories(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)
	req := httptest.NewRequest("GET", "/calibre/ajax/categories", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}
}

func TestCalibreServerController_GetCategory(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)
	req := httptest.NewRequest("GET", "/calibre/ajax/category/617574686f7273?num=10&offset=0", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}
}

func TestCalibreServerController_GetBooksInCategory(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)
	req := httptest.NewRequest("GET", "/calibre/ajax/books_in/617574686f7273/item?num=10&offset=0", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}
}

func TestCalibreServerController_Search(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)
	req := httptest.NewRequest("GET", "/calibre/ajax/search?query=test&num=10&offset=0", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}
}

func TestCalibreServerController_GetBooks(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)
	req := httptest.NewRequest("GET", "/calibre/ajax/books?ids=book-1", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}
}

func TestCalibreServerController_GetBook_SuccessAndNotFound(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)

	req := httptest.NewRequest("GET", "/calibre/ajax/book/book-1", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got status %d, err %v", resp.StatusCode, err)
	}

	reqNF := httptest.NewRequest("GET", "/calibre/ajax/book/notfound", nil)
	respNF, err := app.Test(reqNF)
	if err != nil || respNF.StatusCode != 404 {
		t.Fatalf("expected 404, got status %d, err %v", respNF.StatusCode, err)
	}
}

func TestCalibreServerController_GetContent(t *testing.T) {
	app, _ := setupCalibreControllerApp(t)

	reqCover := httptest.NewRequest("GET", "/calibre/get/cover/book-1", nil)
	respCover, err := app.Test(reqCover)
	if err != nil || respCover.StatusCode != 200 {
		t.Fatalf("expected 200 for cover, got status %d, err %v", respCover.StatusCode, err)
	}

	reqFile := httptest.NewRequest("GET", "/calibre/get/epub/book-1", nil)
	respFile, err := app.Test(reqFile)
	if err != nil || respFile.StatusCode != 200 {
		t.Fatalf("expected 200 for epub, got status %d, err %v", respFile.StatusCode, err)
	}
	if ct := respFile.Header.Get("Content-Type"); ct != "application/epub+zip" {
		t.Errorf("expected Content-Type application/epub+zip, got %q", ct)
	}
}
