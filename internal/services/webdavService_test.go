package services

import (
	"context"
	"testing"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
)

type stubWebDAVLibraryService struct {
	LibraryService
	libraries []*response.LibraryResponse
}

func (s *stubWebDAVLibraryService) ListLibraries(_ context.Context, _ *response.JWTClaims) ([]*response.LibraryResponse, error) {
	return s.libraries, nil
}

type stubWebDAVBookService struct {
	BookService
	books []*models.BookEntity
	files []*models.BookFileEntity
}

func (s *stubWebDAVBookService) SearchBooks(_ context.Context, _ *string, _ *string, _, _, _, _, _ string, _ string, _ string, _ int64, _ string) ([]*models.BookEntity, error) {
	return s.books, nil
}

func (s *stubWebDAVBookService) ListBookFiles(_ context.Context, _ string) ([]*models.BookFileEntity, error) {
	return s.files, nil
}

func (s *stubWebDAVBookService) CanReadBook(_ context.Context, _ *models.BookEntity, _ *response.JWTClaims) bool {
	return true
}

func (s *stubWebDAVBookService) SafeDownloadFilename(title string, ext string) string {
	return title + ext
}

func TestWebDAVService_ResolvePathAndGetBookFile(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)

	lib := &response.LibraryResponse{
		ID:        "lib-1",
		Name:      "Light Novels",
		UpdatedAt: now,
	}

	book := &models.BookEntity{
		ID:        "book-1",
		Title:     "Overlord Vol 1",
		LibraryID: "lib-1",
		UpdatedAt: now,
	}

	file := &models.BookFileEntity{
		ID:        "file-1",
		BookID:    "book-1",
		Path:      "/data/books/overlord.epub",
		Format:    "epub",
		SizeBytes: 1048576,
		ModTime:   now,
	}

	libService := &stubWebDAVLibraryService{libraries: []*response.LibraryResponse{lib}}
	bookService := &stubWebDAVBookService{
		books: []*models.BookEntity{book},
		files: []*models.BookFileEntity{file},
	}
	perms := &stubDoctorPermCache{allow: true}
	settings := &stubDoctorSettings{}

	svc := NewWebDAVService(libService, bookService, perms, settings)
	ctx := context.Background()

	// 1. Test Root Path (Depth 0)
	rootNodes, err := svc.ResolvePath(ctx, "/", nil, 0)
	if err != nil {
		t.Fatalf("ResolvePath root failed: %v", err)
	}
	if len(rootNodes) != 1 || !rootNodes[0].IsDir || rootNodes[0].Href != "/webdav/" {
		t.Fatalf("expected 1 root node, got: %#v", rootNodes)
	}

	// 2. Test Root Path (Depth 1)
	rootChildren, err := svc.ResolvePath(ctx, "/", nil, 1)
	if err != nil {
		t.Fatalf("ResolvePath root depth 1 failed: %v", err)
	}
	if len(rootChildren) != 2 { // root + 1 library
		t.Fatalf("expected 2 nodes, got %d", len(rootChildren))
	}
	if rootChildren[1].Href != "/webdav/Light Novels/" || rootChildren[1].DisplayName != "Light Novels" {
		t.Fatalf("unexpected library node: %#v", rootChildren[1])
	}

	// 3. Test Library Path (Depth 1)
	libNodes, err := svc.ResolvePath(ctx, "/Light Novels", nil, 1)
	if err != nil {
		t.Fatalf("ResolvePath library failed: %v", err)
	}
	if len(libNodes) != 2 { // library + 1 book file
		t.Fatalf("expected 2 nodes for library, got %d", len(libNodes))
	}
	if libNodes[1].Href != "/webdav/Light Novels/Overlord Vol 1.epub" {
		t.Fatalf("unexpected file href: %s", libNodes[1].Href)
	}
	if libNodes[1].ContentType != "application/epub+zip" {
		t.Fatalf("unexpected ContentType: %s", libNodes[1].ContentType)
	}

	// 4. Test Single File Path
	fileNodes, err := svc.ResolvePath(ctx, "/Light Novels/Overlord Vol 1.epub", nil, 0)
	if err != nil {
		t.Fatalf("ResolvePath file failed: %v", err)
	}
	if len(fileNodes) != 1 || fileNodes[0].DisplayName != "Overlord Vol 1.epub" {
		t.Fatalf("unexpected single file node: %#v", fileNodes)
	}

	// 5. Test GetBookFile
	filePath, mimeType, downloadName, err := svc.GetBookFile(ctx, "/Light Novels/Overlord Vol 1.epub", nil)
	if err != nil {
		t.Fatalf("GetBookFile failed: %v", err)
	}
	if filePath != "/data/books/overlord.epub" {
		t.Fatalf("expected /data/books/overlord.epub, got %s", filePath)
	}
	if mimeType != "application/epub+zip" {
		t.Fatalf("expected application/epub+zip, got %s", mimeType)
	}
	if downloadName != "Overlord Vol 1.epub" {
		t.Fatalf("expected Overlord Vol 1.epub, got %s", downloadName)
	}

	// 6. Test WebDAV Permission Denial
	perms.allow = false
	if _, err := svc.ResolvePath(ctx, "/", nil, 0); err == nil {
		t.Fatal("expected forbidden error when user lacks webdav.read permission")
	}
	if _, _, _, err := svc.GetBookFile(ctx, "/Light Novels/Overlord Vol 1.epub", nil); err == nil {
		t.Fatal("expected forbidden error when user lacks webdav.download permission")
	}
}
