package services

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/plain"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
	"novelhub/pkg/worker"
)

func newConvertTestService(t *testing.T, booksDir string) (*bookService, *sql.DB, repositories.BookFileRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "convert.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	fileRepo, err := repositories.NewBookFileRepository(booksDir)
	if err != nil {
		t.Fatal(err)
	}
	reg := bookparser.NewRegistry()
	reg.Register(plain.NewParser(), "txt")
	bookRepo := repositories.NewBookDBRepository(db, cache.NewRamCache())
	svc := NewBookService(bookRepo, nil, nil, fileRepo, reg, database.NewTxManager(db), nil, nil, nil, nil)
	return svc.(*bookService), db, fileRepo
}

func seedConvertSource(t *testing.T, db *sql.DB, fileRepo repositories.BookFileRepository, booksDir string) (string, string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1', 'L')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('b-1', 'lib-1', 'Convert Me', 'ready')`); err != nil {
		t.Fatal(err)
	}
	content := "Convert Me\n\nAlpha chapter\n\nFirst paragraph of prose.\nSecond line.\n"
	saved, err := fileRepo.SaveBook(context.Background(), "b-1", "source.txt", strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES ('bf-1', 'b-1', ?, 'txt', ?, ?)`,
		saved.Path, saved.SizeBytes, saved.ModTime.Format("2006-01-02 15:04:05")); err != nil {
		t.Fatal(err)
	}
	return "b-1", "bf-1"
}

func zipHas(t *testing.T, data []byte, name string) bool {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	for _, f := range r.File {
		if f.Name == name {
			return true
		}
	}
	return false
}

func TestConvertBookRejectsFileFromOtherBook(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	_, bfID := seedConvertSource(t, db, fileRepo, t.TempDir())
	if _, err := svc.ConvertBook(context.Background(), "some-other-book", bfID, "epub"); err == nil {
		t.Fatal("ConvertBook(file from another book): expected error")
	}
}

func TestConvertBookRejectsUnsupportedTarget(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	_, bfID := seedConvertSource(t, db, fileRepo, t.TempDir())
	if _, err := svc.ConvertBook(context.Background(), "b-1", bfID, "pdf"); err == nil {
		t.Fatal("ConvertBook(pdf): expected error")
	}
}

func TestConvertBookRejectsSameFormat(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	_, bfID := seedConvertSource(t, db, fileRepo, t.TempDir())
	if _, err := svc.ConvertBook(context.Background(), "b-1", bfID, "txt"); err == nil {
		t.Fatal("ConvertBook(txt → txt): expected error")
	}
}

func TestConvertBookRejectsMissingFile(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	seedConvertSource(t, db, fileRepo, t.TempDir())
	if _, err := svc.ConvertBook(context.Background(), "b-1", "does-not-exist", "epub"); err == nil {
		t.Fatal("ConvertBook(missing file): expected error")
	}
}

func TestConvertBookSynchronous(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	_, bfID := seedConvertSource(t, db, fileRepo, booksDir)

	jobID, err := svc.ConvertBook(context.Background(), "b-1", bfID, "epub")
	if err != nil {
		t.Fatalf("ConvertBook: %v", err)
	}
	if jobID == "" {
		t.Fatal("jobID is empty")
	}

	var (
		format string
		path   string
		size   int64
	)
	if err := db.QueryRow(`SELECT format, path, size_bytes FROM book_files WHERE book_id = 'b-1' AND id != 'bf-1'`).Scan(&format, &path, &size); err != nil {
		t.Fatalf("converted file row: %v", err)
	}
	if format != "epub" {
		t.Errorf("format = %q, want epub", format)
	}
	if size == 0 {
		t.Error("converted file size is 0")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read converted file: %v", err)
	}
	if !zipHas(t, data, "mimetype") {
		t.Error("converted file is not a valid epub (missing mimetype)")
	}
	if !zipHas(t, data, "OEBPS/content.opf") {
		t.Error("converted file is not a valid epub (missing content.opf)")
	}
}

func TestConvertBookEnqueuesJob(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	_, bfID := seedConvertSource(t, db, fileRepo, t.TempDir())

	q := worker.NewQueue(1)
	defer q.Stop()
	var (
		mu sync.Mutex
		gotPayloads []string
	)
	q.RegisterHandler(convertBookJobType, func(_ context.Context, _ string, payload string) error {
		mu.Lock()
		gotPayloads = append(gotPayloads, payload)
		mu.Unlock()
		return nil
	})
	q.Start()
	svc.jobQueue = q

	jobID, err := svc.ConvertBook(context.Background(), "b-1", bfID, "docx")
	if err != nil {
		t.Fatalf("ConvertBook: %v", err)
	}
	if jobID == "" {
		t.Fatal("jobID is empty")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(gotPayloads)
		mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotPayloads) != 1 {
		t.Fatalf("enqueued %d jobs, want 1", len(gotPayloads))
	}
	var payload convertBookPayload
	if err := json.Unmarshal([]byte(gotPayloads[0]), &payload); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if payload.BookID != "b-1" || payload.FileID != bfID || payload.TargetFormat != "docx" {
		t.Errorf("payload = %+v", payload)
	}
}

func TestExecuteConvertBookJobInvalidPayload(t *testing.T) {
	svc, _, _ := newConvertTestService(t, t.TempDir())
	if err := svc.ExecuteConvertBookJob(context.Background(), "{not json"); err == nil {
		t.Fatal("invalid payload: expected error")
	}
}

func TestExecuteConvertBookJobUnsupportedTarget(t *testing.T) {
	svc, _, _ := newConvertTestService(t, t.TempDir())
	payload, _ := json.Marshal(convertBookPayload{BookID: "b-1", FileID: "bf-1", TargetFormat: "mobi"})
	if err := svc.ExecuteConvertBookJob(context.Background(), string(payload)); err == nil {
		t.Fatal("unsupported target: expected error")
	}
}

func TestExecuteConvertBookJobMissingFile(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	bookID, _ := seedConvertSource(t, db, fileRepo, t.TempDir())
	payload, _ := json.Marshal(convertBookPayload{BookID: bookID, FileID: "missing", TargetFormat: "epub"})
	if err := svc.ExecuteConvertBookJob(context.Background(), string(payload)); err == nil {
		t.Fatal("missing file: expected error")
	}
}

func TestExecuteConvertBookJobFull(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	bookID, bfID := seedConvertSource(t, db, fileRepo, booksDir)

	payload, _ := json.Marshal(convertBookPayload{BookID: bookID, FileID: bfID, TargetFormat: "fb2"})
	if err := svc.ExecuteConvertBookJob(context.Background(), string(payload)); err != nil {
		t.Fatalf("ExecuteConvertBookJob: %v", err)
	}

	var (
		format string
		path   string
	)
	if err := db.QueryRow(`SELECT format, path FROM book_files WHERE book_id = ? AND id != 'bf-1'`, bookID).Scan(&format, &path); err != nil {
		t.Fatalf("converted file row: %v", err)
	}
	if format != "fb2" {
		t.Errorf("format = %q, want fb2", format)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read converted file: %v", err)
	}
	if !strings.Contains(string(data), "<FictionBook") {
		t.Errorf("output is not FB2: %.200s", data)
	}
	if !strings.Contains(string(data), "First paragraph of prose.") {
		t.Errorf("output missing prose: %.200s", data)
	}
	// physical file must sit under the book dir, named from the source stem
	if filepath.Base(path) != "source.fb2" {
		t.Errorf("output filename = %q, want source.fb2", filepath.Base(path))
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("converted file on disk: %v", err)
	}
}

func TestExecuteConvertBookJobCleanupOnFailure(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	bookID, bfID := seedConvertSource(t, db, fileRepo, booksDir)
	// Point the source row at a nonexistent path so Convert fails mid-job.
	if _, err := db.Exec(`UPDATE book_files SET path = '/nonexistent/source.txt' WHERE id = 'bf-1'`); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(convertBookPayload{BookID: bookID, FileID: bfID, TargetFormat: "epub"})
	if err := svc.ExecuteConvertBookJob(context.Background(), string(payload)); err == nil {
		t.Fatal("expected failure for missing source path")
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM book_files WHERE id != 'bf-1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("orphan file row created on failure: %d", n)
	}
}

func TestConvertBookDoesNotDoubleOnDuplicatePath(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	bookID, bfID := seedConvertSource(t, db, fileRepo, booksDir)
	payload, _ := json.Marshal(convertBookPayload{BookID: bookID, FileID: bfID, TargetFormat: "epub"})
	if err := svc.ExecuteConvertBookJob(context.Background(), string(payload)); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// second run: SaveBook appends -2 suffix → different path, so both rows exist
	if err := svc.ExecuteConvertBookJob(context.Background(), string(payload)); err != nil {
		t.Fatalf("second run: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM book_files WHERE book_id = ? AND id != 'bf-1'`, bookID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 converted file rows (one per run), got %d", n)
	}
}

func TestConvertBookNilParsersGuard(t *testing.T) {
	svc, db, fileRepo := newConvertTestService(t, t.TempDir())
	_, bfID := seedConvertSource(t, db, fileRepo, t.TempDir())
	svc.parsers = nil
	if _, err := svc.ConvertBook(context.Background(), "b-1", bfID, "epub"); err == nil {
		t.Fatal("nil parsers registry: expected error")
	}
}

func TestConvertBookTargetUppercase(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	_, bfID := seedConvertSource(t, db, fileRepo, booksDir)
	jobID, err := svc.ConvertBook(context.Background(), "b-1", bfID, "EPUB")
	if err != nil {
		t.Fatalf("ConvertBook(EPUB): %v", err)
	}
	if jobID == "" {
		t.Fatal("jobID is empty")
	}
	var format string
	if err := db.QueryRow(`SELECT format FROM book_files WHERE book_id = 'b-1' AND id != 'bf-1'`).Scan(&format); err != nil {
		t.Fatalf("converted file row: %v", err)
	}
	if format != "epub" {
		t.Errorf("format = %q, want normalized epub", format)
	}
}

func TestConvertBookSourceFormatFromExt(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	// seed a book_file whose format field is blank; Convert must derive it from the path
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-2', 'L')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title, status) VALUES ('b-2', 'lib-2', 'T', 'ready')`); err != nil {
		t.Fatal(err)
	}
	saved, err := fileRepo.SaveBook(context.Background(), "b-2", "src.txt", strings.NewReader("T\n\nBody.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES ('bf-2', 'b-2', ?, '', ?, ?)`,
		saved.Path, saved.SizeBytes, saved.ModTime.Format("2006-01-02 15:04:05")); err != nil {
		t.Fatal(err)
	}
	jobID, err := svc.ConvertBook(context.Background(), "b-2", "bf-2", "epub")
	if err != nil {
		t.Fatalf("ConvertBook(blank format): %v", err)
	}
	if jobID == "" {
		t.Fatal("jobID is empty")
	}
	var format string
	if err := db.QueryRow(`SELECT format FROM book_files WHERE book_id = 'b-2' AND id != 'bf-2'`).Scan(&format); err != nil {
		t.Fatalf("converted file row: %v", err)
	}
	if format != "epub" {
		t.Errorf("format = %q, want epub", format)
	}
}

func TestConvertBookDoesNotTouchSourceFile(t *testing.T) {
	booksDir := t.TempDir()
	svc, db, fileRepo := newConvertTestService(t, booksDir)
	_, bfID := seedConvertSource(t, db, fileRepo, booksDir)
	var srcPath string
	if err := db.QueryRow(`SELECT path FROM book_files WHERE id = 'bf-1'`).Scan(&srcPath); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConvertBook(context.Background(), "b-1", bfID, "epub"); err != nil {
		t.Fatalf("ConvertBook: %v", err)
	}
	after, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("source file was modified by conversion")
	}
}