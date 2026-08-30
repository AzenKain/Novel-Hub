package controllers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
)

type stubIntegrationsServiceForCtrl struct {
	exportedCount int
	markdown      string
	apkgBytes     []byte
	bookTitle     string
	csvStr        string
	err           error
}

func (s *stubIntegrationsServiceForCtrl) ExportHighlightsToReadwise(_ context.Context, _ string, _ string, _ *response.JWTClaims) (int, error) {
	return s.exportedCount, s.err
}

func (s *stubIntegrationsServiceForCtrl) ExportHighlightsMarkdown(_ context.Context, _ string, _ string, _ *response.JWTClaims) (string, error) {
	return s.markdown, s.err
}

func (s *stubIntegrationsServiceForCtrl) ExportHighlightsAnki(_ context.Context, _ string, _ string, _ *response.JWTClaims) ([]byte, string, error) {
	return s.apkgBytes, s.bookTitle, s.err
}

func (s *stubIntegrationsServiceForCtrl) ExportHighlightsCSV(_ context.Context, _ string, _ string, _ *response.JWTClaims) (string, error) {
	return s.csvStr, s.err
}

const validTestUserID = "0195540a-5b12-7000-8000-000000000001"

func TestIntegrationsController_ExportHighlightsAnki(t *testing.T) {
	stub := &stubIntegrationsServiceForCtrl{
		apkgBytes: []byte("PK\x03\x04fake-apkg-zip-content"),
		bookTitle: "Dune",
		err:       nil,
	}
	ctrl := NewIntegrationsController(stub)

	app := fiber.New()
	app.Get("/highlights/:book_id/export.apkg", func(c fiber.Ctx) error {
		c.Locals("uid", validTestUserID)
		return ctrl.ExportHighlightsAnki(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/highlights/book-1/export.apkg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/apkg" {
		t.Fatalf("expected Content-Type 'application/apkg', got '%s'", contentType)
	}

	disposition := resp.Header.Get("Content-Disposition")
	if disposition != `attachment; filename="Dune_highlights.apkg"` {
		t.Fatalf("expected disposition with Dune_highlights.apkg, got '%s'", disposition)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "PK\x03\x04fake-apkg-zip-content" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestIntegrationsController_ExportHighlightsCSV(t *testing.T) {
	stub := &stubIntegrationsServiceForCtrl{
		csvStr: "#separator:tab\nFront\tBack\tContext\tTags\n",
		err:    nil,
	}
	ctrl := NewIntegrationsController(stub)

	app := fiber.New()
	app.Get("/highlights/:book_id/export.csv", func(c fiber.Ctx) error {
		c.Locals("uid", validTestUserID)
		return ctrl.ExportHighlightsCSV(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/highlights/book-1/export.csv", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/csv; charset=utf-8" {
		t.Fatalf("expected Content-Type 'text/csv; charset=utf-8', got '%s'", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "#separator:tab\nFront\tBack\tContext\tTags\n" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestIntegrationsController_ExportHighlightsAnki_Unauthorized(t *testing.T) {
	stub := &stubIntegrationsServiceForCtrl{}
	ctrl := NewIntegrationsController(stub)

	app := fiber.New()
	app.Get("/highlights/:book_id/export.apkg", ctrl.ExportHighlightsAnki)

	req := httptest.NewRequest(http.MethodGet, "/highlights/book-1/export.apkg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestIntegrationsController_ExportHighlightsAnki_NotFound(t *testing.T) {
	stub := &stubIntegrationsServiceForCtrl{
		err: apperrors.New(apperrors.ErrNotFound, "No highlights to export"),
	}
	ctrl := NewIntegrationsController(stub)

	app := fiber.New()
	app.Get("/highlights/:book_id/export.apkg", func(c fiber.Ctx) error {
		c.Locals("uid", validTestUserID)
		return ctrl.ExportHighlightsAnki(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/highlights/book-1/export.apkg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}
