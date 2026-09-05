package controllers

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/webdav"
)

type stubWebDAVService struct {
	nodes []webdav.WebDAVNode
	err   error
}

func (s *stubWebDAVService) ResolvePath(_ context.Context, _ string, _ *response.JWTClaims, _ int) ([]webdav.WebDAVNode, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.nodes, nil
}

func (s *stubWebDAVService) GetBookFile(_ context.Context, _ string, _ *response.JWTClaims) (filePath string, mimeType string, downloadName string, err error) {
	return "/tmp/sample.epub", "application/epub+zip", "sample.epub", nil
}

func TestWebDAVController_OptionsAndPropfind(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	svc := &stubWebDAVService{
		nodes: []webdav.WebDAVNode{
			{
				Href:        "/webdav/",
				DisplayName: "NovelHub",
				IsDir:       true,
				ModTime:     now,
			},
			{
				Href:        "/webdav/Default/",
				DisplayName: "Default",
				IsDir:       true,
				ModTime:     now,
			},
		},
	}

	ctrl := NewWebDAVController(svc)
	app := fiber.New(fiber.Config{
		RequestMethods: []string{fiber.MethodOptions, fiber.MethodGet, fiber.MethodHead, "PROPFIND"},
	})

	app.Add([]string{"OPTIONS"}, "/webdav*", ctrl.HandleOptions)
	app.Add([]string{"PROPFIND"}, "/webdav*", ctrl.HandlePropfind)

	optReq := httptest.NewRequest("OPTIONS", "/webdav", nil)
	optResp, err := app.Test(optReq)
	if err != nil {
		t.Fatalf("OPTIONS test failed: %v", err)
	}
	if optResp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", optResp.StatusCode)
	}
	if optResp.Header.Get("DAV") != "1, 2" {
		t.Fatalf("expected DAV header '1, 2', got %q", optResp.Header.Get("DAV"))
	}
	if !strings.Contains(optResp.Header.Get("Allow"), "PROPFIND") {
		t.Fatalf("expected Allow header with PROPFIND, got %q", optResp.Header.Get("Allow"))
	}

	propReq := httptest.NewRequest("PROPFIND", "/webdav", nil)
	propReq.Header.Set("Depth", "1")
	propResp, err := app.Test(propReq)
	if err != nil {
		t.Fatalf("PROPFIND test failed: %v", err)
	}
	if propResp.StatusCode != 207 {
		t.Fatalf("expected status 207 Multi-Status, got %d", propResp.StatusCode)
	}
	if !strings.Contains(propResp.Header.Get("Content-Type"), "xml") {
		t.Fatalf("expected XML content-type, got %q", propResp.Header.Get("Content-Type"))
	}
}
