package services

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
)

type stubDoctorBookRepo struct {
	repositories.BookDBRepository
	book  *models.BookEntity
	files []*models.BookFileEntity
}

func (s *stubDoctorBookRepo) GetBook(_ context.Context, _ string) (*models.BookEntity, error) {
	return s.book, nil
}

func (s *stubDoctorBookRepo) ListBookIDs(_ context.Context, _ *time.Time, _ string, _ int64) ([]string, error) {
	if s.book != nil {
		return []string{s.book.ID}, nil
	}
	return nil, nil
}

func (s *stubDoctorBookRepo) GetFilesByBookId(_ context.Context, _ string) ([]*models.BookFileEntity, error) {
	return s.files, nil
}

func (s *stubDoctorBookRepo) GetBooksByIDs(_ context.Context, _ []string) ([]*models.BookEntity, error) {
	if s.book != nil {
		return []*models.BookEntity{s.book}, nil
	}
	return nil, nil
}

type stubDoctorPermCache struct {
	PermissionCache
	allow bool
}

func (p *stubDoctorPermCache) Can(_ context.Context, _ string, _ string, _ map[string]any) bool {
	return p.allow
}

func (p *stubDoctorPermCache) CanRoles(_ []string, _ []constants.RoleType, _ string, _ map[string]any) bool {
	return p.allow
}

func (p *stubDoctorPermCache) IsAdmin(_ []string, _ []constants.RoleType) bool {
	return p.allow
}

type stubDoctorSettings struct {
	SettingsService
}

func (s *stubDoctorSettings) GuestAllows(_ string) bool {
	return true
}

func (s *stubDoctorSettings) Public(_ context.Context) (*models.PublicSettings, error) {
	return &models.PublicSettings{
		GuestLoginRequired: false,
	}, nil
}

func TestBookService_ValidateAndRepairBookEPUB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-book-doctor-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	epubPath := filepath.Join(tmpDir, "book.epub")

	// Create test EPUB with compressed mimetype and missing dc:language
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}
	zw := zip.NewWriter(f)

	mw, _ := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Deflate})
	_, _ = mw.Write([]byte("application/epub+zip"))

	cw, _ := zw.Create("META-INF/container.xml")
	_, _ = cw.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	ow, _ := zw.Create("OEBPS/content.opf")
	_, _ = ow.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Doctor Test</dc:title></metadata><manifest><item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="ch1"/></spine></package>`))

	ch1w, _ := zw.Create("OEBPS/ch1.xhtml")
	_, _ = ch1w.Write([]byte(`<html><body><h1>Hello</h1><p>Test&nbsp;Content<br><img src="cover.jpg"></p></body></html>`))

	_ = zw.Close()
	_ = f.Close()

	book := &models.BookEntity{
		ID:        "book-doctor-1",
		Title:     "Doctor Test",
		LibraryID: "lib-1",
	}
	file := &models.BookFileEntity{
		ID:     "file-1",
		BookID: "book-doctor-1",
		Path:   epubPath,
		Format: "epub",
	}

	repo := &stubDoctorBookRepo{
		book:  book,
		files: []*models.BookFileEntity{file},
	}
	perms := &stubDoctorPermCache{allow: true}
	settings := &stubDoctorSettings{}

	s := &bookService{
		bookRepo:    repo,
		permissions: perms,
		settings:    settings,
	}

	// 1. Test Validate
	report, err := s.ValidateBookEPUB(context.Background(), "book-doctor-1", "file-1", nil)
	if err != nil {
		t.Fatalf("ValidateBookEPUB failed: %v", err)
	}

	if report.Warnings == 0 && report.Errors == 0 {
		t.Fatal("expected issues to be detected in broken EPUB")
	}

	// 2. Test Repair
	res, err := s.RepairBookEPUB(context.Background(), "book-doctor-1", "file-1", nil, nil)
	if err != nil {
		t.Fatalf("RepairBookEPUB failed: %v", err)
	}

	if !res.Success {
		t.Fatal("expected repair to succeed")
	}

	if res.FixedCount == 0 {
		t.Fatal("expected fixes to be applied")
	}

	// Post-repair report should be valid
	if !res.Report.Valid {
		t.Fatalf("expected post-repair to be valid, got errors: %d", res.Report.Errors)
	}

	// 3. Test Batch Repair Job
	if err := s.ExecuteBatchRepairBooksJob(context.Background(), `{"library_id":"lib-1"}`); err != nil {
		t.Fatalf("ExecuteBatchRepairBooksJob failed: %v", err)
	}

	// 4. Test Permission Denial
	perms.allow = false
	if _, err := s.RepairBookEPUB(context.Background(), "book-doctor-1", "file-1", nil, nil); err == nil {
		t.Fatal("expected permission error when user is not allowed")
	}
}
