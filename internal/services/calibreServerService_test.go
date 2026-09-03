package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
)

type mockCalibreBookRepo struct {
	repositories.BookDBRepository
	books []*models.BookEntity
}

func (m *mockCalibreBookRepo) SearchBooks(_ context.Context, _ *string, _ *string, _, _, _, _, _ string, _ string, _ string, _ int64, _ string) ([]*models.BookEntity, error) {
	return m.books, nil
}

func (m *mockCalibreBookRepo) ListBookIDsByLibrary(_ context.Context, _ string, _ int64) ([]string, error) {
	ids := make([]string, len(m.books))
	for i, b := range m.books {
		ids[i] = b.ID
	}
	return ids, nil
}

func (m *mockCalibreBookRepo) GetBooksByIDs(_ context.Context, ids []string) ([]*models.BookEntity, error) {
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	var res []*models.BookEntity
	for _, b := range m.books {
		if idMap[b.ID] {
			res = append(res, b)
		}
	}
	return res, nil
}

func (m *mockCalibreBookRepo) GetAuthorByName(_ context.Context, name string) (*models.AuthorEntity, error) {
	return &models.AuthorEntity{ID: "auth-1", Name: name}, nil
}
func (m *mockCalibreBookRepo) GetSeriesByName(_ context.Context, name string) (*models.SeriesEntity, error) {
	return &models.SeriesEntity{ID: "series-1", Name: name}, nil
}
func (m *mockCalibreBookRepo) GetTagByName(_ context.Context, name string) (*models.TagEntity, error) {
	return &models.TagEntity{ID: "tag-1", Name: name}, nil
}
func (m *mockCalibreBookRepo) GetPublisherByName(_ context.Context, name string) (*models.PublisherEntity, error) {
	return &models.PublisherEntity{ID: "pub-1", Name: name}, nil
}
func (m *mockCalibreBookRepo) GetLanguageByName(_ context.Context, name string) (*models.LanguageEntity, error) {
	return &models.LanguageEntity{ID: "lang-1", Name: name}, nil
}

type mockCalibreBookFileRepo struct {
	repositories.BookFileRepository
	coverPath string
}

func (m *mockCalibreBookFileRepo) ResolveCoverPath(_ context.Context, bookID, coverURL string) (string, error) {
	return m.coverPath, nil
}

type mockCalibreBookService struct {
	BookService
	books map[string]*models.BookEntity
}

func (m *mockCalibreBookService) GetBook(_ context.Context, id string) (*models.BookEntity, error) {
	b, ok := m.books[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

func (m *mockCalibreBookService) CanReadBook(_ context.Context, _ *models.BookEntity, _ *response.JWTClaims) bool {
	return true
}

func (m *mockCalibreBookService) CanDownloadBook(_ context.Context, _ *models.BookEntity, _ *response.JWTClaims) bool {
	return true
}

func (m *mockCalibreBookService) GetBookFileForDownload(_ context.Context, bookID string, fileID string, _ *response.JWTClaims) (string, string, error) {
	b, ok := m.books[bookID]
	if !ok {
		return "", "", os.ErrNotExist
	}
	for _, f := range b.Files {
		if f.ID == fileID {
			return f.Path, b.Title + ".epub", nil
		}
	}
	return "", "", os.ErrNotExist
}

func (m *mockCalibreBookService) FilterReadableBooks(_ context.Context, books []*models.BookEntity, _ *response.JWTClaims) ([]*models.BookEntity, bool) {
	return books, false
}

type mockCalibreMetadataService struct {
	MetadataService
	authors    []*response.MetadataCountResponse
	series     []*response.MetadataCountResponse
	tags       []*response.MetadataCountResponse
	formats    []*response.MetadataCountResponse
	publishers []*response.MetadataCountResponse
}

func (m *mockCalibreMetadataService) ListAuthors(_ context.Context, _ *request.MetadataFacetDto, _ *response.JWTClaims) (*response.PaginatedResponse, error) {
	return response.BuildPaginatedResponse(m.authors, int64(len(m.authors)), 1, 100), nil
}
func (m *mockCalibreMetadataService) ListSeries(_ context.Context, _ *request.MetadataFacetDto, _ *response.JWTClaims) (*response.PaginatedResponse, error) {
	return response.BuildPaginatedResponse(m.series, int64(len(m.series)), 1, 100), nil
}
func (m *mockCalibreMetadataService) ListTags(_ context.Context, _ *request.MetadataFacetDto, _ *response.JWTClaims) (*response.PaginatedResponse, error) {
	return response.BuildPaginatedResponse(m.tags, int64(len(m.tags)), 1, 100), nil
}
func (m *mockCalibreMetadataService) ListFormats(_ context.Context, _ *request.MetadataFacetDto, _ *response.JWTClaims) (*response.PaginatedResponse, error) {
	return response.BuildPaginatedResponse(m.formats, int64(len(m.formats)), 1, 100), nil
}
func (m *mockCalibreMetadataService) ListPublishers(_ context.Context, _ *request.MetadataFacetDto, _ *response.JWTClaims) (*response.PaginatedResponse, error) {
	return response.BuildPaginatedResponse(m.publishers, int64(len(m.publishers)), 1, 100), nil
}

type mockCalibreLibraryService struct {
	LibraryService
	readableIDs []string
	libraries   map[string]*response.LibraryResponse
}

func (m *mockCalibreLibraryService) ReadableLibraryIDs(_ context.Context, _ *response.JWTClaims) ([]string, error) {
	return m.readableIDs, nil
}
func (m *mockCalibreLibraryService) GetLibrary(_ context.Context, id string, _ *response.JWTClaims) (*response.LibraryResponse, error) {
	lib, ok := m.libraries[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return lib, nil
}

func setupCalibreServerTest(t *testing.T) (CalibreServerService, string) {
	tmpDir := t.TempDir()
	now := time.Now()

	bookID := "book-uuid-1"
	bookDir := filepath.Join(tmpDir, bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}

	epubPath := filepath.Join(bookDir, "test.epub")
	if err := os.WriteFile(epubPath, []byte("fake epub content"), 0644); err != nil {
		t.Fatalf("failed to write epub: %v", err)
	}

	coverPath := filepath.Join(bookDir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("fake image data"), 0644); err != nil {
		t.Fatalf("failed to write cover: %v", err)
	}

	coverURL := "/covers/" + bookID + ".jpg"
	authorName := "Arthur Conan Doyle"
	description := "A mystery novel by Arthur Conan Doyle"
	metaJSON := `{"creator":"Arthur Conan Doyle","series":"Sherlock Holmes","seriesIndex":2,"publisher":"Standard Books","subject":["Mystery","Classic"]}`

	book := &models.BookEntity{
		ID:           bookID,
		Title:        "The Sign of Four",
		Description:  &description,
		AuthorName:   &authorName,
		CoverURL:     &coverURL,
		MetadataJSON: &metaJSON,
		Files: []*models.BookFileEntity{
			{
				ID:        "file-1",
				BookID:    bookID,
				Path:      filepath.Join(bookID, "test.epub"),
				Format:    "EPUB",
				SizeBytes: 1234,
				ModTime:   now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	bookRepo := &mockCalibreBookRepo{
		books: []*models.BookEntity{book},
	}
	diskRepo := &mockCalibreBookFileRepo{coverPath: coverPath}
	bookSvc := &mockCalibreBookService{books: map[string]*models.BookEntity{bookID: book}}
	metaSvc := &mockCalibreMetadataService{
		authors: []*response.MetadataCountResponse{
			{Name: "Arthur Conan Doyle", BookCount: 1},
		},
		series: []*response.MetadataCountResponse{
			{Name: "Sherlock Holmes", BookCount: 1},
		},
		tags: []*response.MetadataCountResponse{
			{Name: "Mystery", BookCount: 1},
			{Name: "Classic", BookCount: 1},
		},
		formats: []*response.MetadataCountResponse{
			{Name: "EPUB", BookCount: 1},
		},
		publishers: []*response.MetadataCountResponse{
			{Name: "Standard Books", BookCount: 1},
		},
	}
	libSvc := &mockCalibreLibraryService{
		readableIDs: []string{"lib-1"},
		libraries: map[string]*response.LibraryResponse{
			"lib-1": {ID: "lib-1", Name: "Main Library"},
		},
	}

	svc := NewCalibreServerService(bookRepo, diskRepo, bookSvc, metaSvc, libSvc, tmpDir)
	return svc, bookID
}

func TestCalibreServerService_GetLibraryInfo(t *testing.T) {
	svc, _ := setupCalibreServerTest(t)
	ctx := context.Background()

	info, err := svc.GetLibraryInfo(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.DefaultLibrary != "lib-1" {
		t.Errorf("expected default library 'lib-1', got %q", info.DefaultLibrary)
	}
	if info.LibraryMap["lib-1"] != "Main Library" {
		t.Errorf("expected 'Main Library', got %q", info.LibraryMap["lib-1"])
	}
}

func TestCalibreServerService_GetCategories(t *testing.T) {
	svc, _ := setupCalibreServerTest(t)
	ctx := context.Background()

	cats, err := svc.GetCategories(ctx, "lib-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCats := []string{"allbooks", "authors", "series", "tags", "formats", "publishers"}
	for _, c := range expectedCats {
		item, ok := cats[c]
		if !ok {
			t.Errorf("expected category %q in result", c)
			continue
		}
		if item.Count <= 0 {
			t.Errorf("expected non-zero count for category %q, got %d", c, item.Count)
		}
		if item.URL == "" {
			t.Errorf("expected URL for category %q", c)
		}
	}
}

func TestCalibreServerService_GetCategory_Authors(t *testing.T) {
	svc, _ := setupCalibreServerTest(t)
	ctx := context.Background()

	detail, err := svc.GetCategory(ctx, "lib-1", "617574686f7273", 50, 0, "name", "asc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.CategoryName != "authors" {
		t.Errorf("expected category name 'authors', got %q", detail.CategoryName)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("expected 1 author item, got %d", len(detail.Items))
	}
	if detail.Items[0].Name != "Arthur Conan Doyle" {
		t.Errorf("expected author 'Arthur Conan Doyle', got %q", detail.Items[0].Name)
	}
}

func TestCalibreServerService_GetCategory_NotFound(t *testing.T) {
	svc, _ := setupCalibreServerTest(t)
	ctx := context.Background()

	_, err := svc.GetCategory(ctx, "lib-1", "unknown_category", 50, 0, "name", "asc", nil)
	if err == nil {
		t.Fatalf("expected error for unknown category, got nil")
	}
}

func TestCalibreServerService_GetBooksInCategory(t *testing.T) {
	svc, bookID := setupCalibreServerTest(t)
	ctx := context.Background()

	res, err := svc.GetBooksInCategory(ctx, "lib-1", "617574686f7273", "41727468757220436f6e616e20446f796c65", 50, 0, "title", "asc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalNum != 1 {
		t.Errorf("expected total_num 1, got %d", res.TotalNum)
	}
	if len(res.BookIDs) != 1 || res.BookIDs[0] != bookID {
		t.Errorf("expected book ID %q, got %v", bookID, res.BookIDs)
	}
}

func TestCalibreServerService_SearchBooks(t *testing.T) {
	svc, bookID := setupCalibreServerTest(t)
	ctx := context.Background()

	res, err := svc.SearchBooks(ctx, "lib-1", "Sign of Four", 50, 0, "title", "asc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalNum != 1 {
		t.Errorf("expected total_num 1, got %d", res.TotalNum)
	}
	if len(res.BookIDs) != 1 || res.BookIDs[0] != bookID {
		t.Errorf("expected book ID %q, got %v", bookID, res.BookIDs)
	}
}

func TestCalibreServerService_GetBooksMetadata(t *testing.T) {
	svc, bookID := setupCalibreServerTest(t)
	ctx := context.Background()

	metaMap, err := svc.GetBooksMetadata(ctx, "lib-1", []string{bookID}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	meta, ok := metaMap[bookID]
	if !ok {
		t.Fatalf("expected metadata for %q", bookID)
	}
	if meta.Title != "The Sign of Four" {
		t.Errorf("expected title 'The Sign of Four', got %q", meta.Title)
	}
	if len(meta.Authors) != 1 || meta.Authors[0] != "Arthur Conan Doyle" {
		t.Errorf("unexpected authors: %v", meta.Authors)
	}
	if meta.Series == nil || *meta.Series != "Sherlock Holmes" {
		t.Errorf("unexpected series: %v", meta.Series)
	}
	if meta.Cover == "" || meta.Thumbnail == "" {
		t.Errorf("expected cover and thumbnail URLs")
	}
	if len(meta.Formats) != 1 || meta.Formats[0] != "EPUB" {
		t.Errorf("expected formats ['EPUB'], got %v", meta.Formats)
	}
}

func TestCalibreServerService_GetBookMetadata(t *testing.T) {
	svc, bookID := setupCalibreServerTest(t)
	ctx := context.Background()

	meta, err := svc.GetBookMetadata(ctx, "lib-1", bookID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "The Sign of Four" {
		t.Errorf("expected title 'The Sign of Four', got %q", meta.Title)
	}
}

func TestCalibreServerService_GetBookCover(t *testing.T) {
	svc, bookID := setupCalibreServerTest(t)
	ctx := context.Background()

	path, err := svc.GetBookCover(ctx, bookID, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cover file should exist on disk: %v", err)
	}
}

func TestCalibreServerService_GetBookFile(t *testing.T) {
	svc, bookID := setupCalibreServerTest(t)
	ctx := context.Background()

	path, filename, err := svc.GetBookFile(ctx, bookID, "epub", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "The Sign of Four.epub" {
		t.Errorf("expected filename 'The Sign of Four.epub', got %q", filename)
	}
	if path == "" {
		t.Errorf("expected non-empty path")
	}

	// Missing format
	_, _, err = svc.GetBookFile(ctx, bookID, "pdf", nil)
	if err == nil {
		t.Fatalf("expected error for missing format 'pdf', got nil")
	}
}
