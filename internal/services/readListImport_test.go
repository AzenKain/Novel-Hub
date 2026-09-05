package services

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func newReadListService(t *testing.T) (ReadListService, *sql.DB, *response.JWTClaims) {
	t.Helper()
	t.Setenv("SQLITE_DB_PATH", filepath.Join(t.TempDir(), "readlist.db"))
	db, err := database.NewSQLiteDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	seed := []string{
		`INSERT INTO users (id,email,password_hash,full_name) VALUES ('u-1','u@e.com','x','U')`,
		`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`,
	}
	for _, stmt := range seed {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	c := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, c)
	txManager := database.NewTxManager(db)
	settingsService := NewSettingsService(repositories.NewSettingsRepository(db, c), txManager, allowAllPermissions{})
	bookService := NewBookService(bookRepo, nil, nil, nil, bookparser.NewRegistry(), txManager, settingsService, allowAllPermissions{}, nil, nil)
	svc := NewReadListService(repositories.NewReadListRepository(db, c), bookRepo, bookService, txManager)
	claims := &response.JWTClaims{UId: "u-1", Roles: []constants.RoleType{constants.RoleTypeAdmin}}
	return svc, db, claims
}

func seedIssue(t *testing.T, db *sql.DB, bookID, seriesID, seriesName, index string) {
	t.Helper()
	if _, err := db.Exec(`INSERT OR IGNORE INTO series (id,name) VALUES (?,?)`, seriesID, seriesName); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id,library_id,title,status) VALUES (?,'lib-1',?,'active')`, bookID, bookID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO book_series (book_id,series_id,series_index) VALUES (?,?,?)`, bookID, seriesID, index); err != nil {
		t.Fatal(err)
	}
}

const civilWarCBL = `<?xml version="1.0"?>
<ReadingList>
  <Name>MV2 - Civil War</Name>
  <Books>
    <Book Series="Amazing Spider-Man" Number="529" Volume="1963" Year="2006" />
    <Book Series="civil war" Number="01" Volume="2006" Year="2006" />
    <Book Series="Captain America" Number="25" />
    <Book Series="Wolverine" Number="42" />
    <Book Series="Civil War" Number="7" />
  </Books>
</ReadingList>`

// The whole point of an import is that reading order survives it.
func TestImportCBLKeepsOrderAndReportsGaps(t *testing.T) {
	svc, db, claims := newReadListService(t)
	ctx := context.Background()

	seedIssue(t, db, "bk-asm529", "s-asm", "Amazing Spider-Man", "529")
	seedIssue(t, db, "bk-cw1", "s-cw", "Civil War", "1")
	seedIssue(t, db, "bk-cw7", "s-cw", "Civil War", "7")
	seedIssue(t, db, "bk-cap25", "s-cap", "Captain America", "25")

	result, err := svc.ImportCBL(ctx, "u-1", strings.NewReader(civilWarCBL), "ignored.cbl")
	if err != nil {
		t.Fatal(err)
	}
	if result.ReadList.Name != "MV2 - Civil War" {
		t.Errorf("name = %q, want the name inside the file", result.ReadList.Name)
	}
	if result.Total != 5 || result.Matched != 4 {
		t.Errorf("total/matched = %d/%d, want 5/4", result.Total, result.Matched)
	}
	if len(result.Unmatched) != 1 || result.Unmatched[0].Series != "Wolverine" || result.Unmatched[0].Number != "42" {
		t.Errorf("unmatched = %+v, want just Wolverine #42", result.Unmatched)
	}

	books, err := svc.GetReadListBooks(ctx, result.ReadList.ID, "u-1", claims)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(books))
	for i, entry := range books {
		got = append(got, entry.Book.ID)
		if entry.Position != int64(i) {
			t.Errorf("entry %d reports position %d", i, entry.Position)
		}
	}
	want := []string{"bk-asm529", "bk-cw1", "bk-cap25", "bk-cw7"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("order = %v, want %v — the .cbl document order is the reading order", got, want)
	}
	if result.ReadList.BookCount != 4 {
		t.Errorf("book_count = %d, want 4", result.ReadList.BookCount)
	}
}

// "1", "01" and "1.0" name the same issue; "1A" and "1B" do not.
func TestImportCBLNumberMatching(t *testing.T) {
	svc, db, _ := newReadListService(t)
	ctx := context.Background()
	seedIssue(t, db, "bk-a", "s-x", "Test Series", "1.0")
	seedIssue(t, db, "bk-b", "s-x", "Test Series", "2")
	seedIssue(t, db, "bk-c", "s-x", "Test Series", "3A")

	body := `<ReadingList><Name>N</Name><Books>
		<Book Series="TEST SERIES" Number="1" />
		<Book Series="test series" Number=" 02 " />
		<Book Series="Test Series" Number="3A" />
		<Book Series="Test Series" Number="3B" />
	</Books></ReadingList>`

	result, err := svc.ImportCBL(ctx, "u-1", strings.NewReader(body), "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Matched != 3 {
		t.Errorf("matched = %d, want 3 (1 == 1.0, 02 == 2, 3A exact)", result.Matched)
	}
	if len(result.Unmatched) != 1 || result.Unmatched[0].Number != "3B" {
		t.Errorf("unmatched = %+v, want only 3B — a variant letter is not a numeric match", result.Unmatched)
	}
}

// A .cbl whose series are all missing still produces a list: the user gets an empty shell plus the full report of what to go and add, which is more useful than an error with no detail.
func TestImportCBLWithNoMatchesStillReports(t *testing.T) {
	svc, _, _ := newReadListService(t)
	ctx := context.Background()
	body := `<ReadingList><Books><Book Series="Nothing Here" Number="1" /></Books></ReadingList>`

	result, err := svc.ImportCBL(ctx, "u-1", strings.NewReader(body), "fallback-name.cbl")
	if err != nil {
		t.Fatal(err)
	}
	if result.ReadList.Name != "fallback-name.cbl" {
		t.Errorf("name = %q, want the filename when the file carries no <Name>", result.ReadList.Name)
	}
	if result.Matched != 0 || len(result.Unmatched) != 1 {
		t.Errorf("matched=%d unmatched=%+v, want 0 matched and one report line", result.Matched, result.Unmatched)
	}

	if _, err := svc.ImportCBL(ctx, "u-1", strings.NewReader(`not xml at all`), ""); err == nil {
		t.Error("a non-XML upload imported without error")
	}
}

// Reorder is the one multi-row write, so it is the one that must be all-or-nothing: an order that does not name every stored book is rejected before a single position is touched.
func TestReorderRejectsAnIncompleteOrder(t *testing.T) {
	svc, db, claims := newReadListService(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		seedIssue(t, db, fmt.Sprintf("bk-%d", i), "s-x", "S", fmt.Sprint(i))
	}
	list, err := svc.CreateReadList(ctx, "u-1", request.CreateReadListDto{Name: "Order"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := svc.AddBook(ctx, list.ID, "u-1", fmt.Sprintf("bk-%d", i), claims); err != nil {
			t.Fatal(err)
		}
	}

	for name, order := range map[string][]string{
		"too short":     {"bk-2", "bk-0"},
		"unknown book":  {"bk-2", "bk-0", "bk-99"},
		"duplicated id": {"bk-2", "bk-0", "bk-0"},
	} {
		if err := svc.Reorder(ctx, list.ID, "u-1", order); err == nil {
			t.Errorf("%s: accepted, want a bad request", name)
		}
	}

	books, err := svc.GetReadListBooks(ctx, list.ID, "u-1", claims)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(books))
	for _, entry := range books {
		got = append(got, entry.Book.ID)
	}
	if want := []string{"bk-0", "bk-1", "bk-2"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("a rejected reorder still moved rows: %v, want %v", got, want)
	}

	if err := svc.Reorder(ctx, list.ID, "u-1", []string{"bk-2", "bk-1", "bk-0"}); err != nil {
		t.Fatal(err)
	}
	books, err = svc.GetReadListBooks(ctx, list.ID, "u-1", claims)
	if err != nil {
		t.Fatal(err)
	}
	got = got[:0]
	for _, entry := range books {
		got = append(got, entry.Book.ID)
	}
	if want := []string{"bk-2", "bk-1", "bk-0"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// Another user's list is not merely invisible, it is untouchable: every entry point checks ownership before it reads or writes, so a guessed id cannot leak titles or reshuffle someone else's order.
func TestReadListRejectsAnotherUser(t *testing.T) {
	svc, db, claims := newReadListService(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO users (id,email,password_hash,full_name) VALUES ('u-2','b@e.com','x','B')`); err != nil {
		t.Fatal(err)
	}
	seedIssue(t, db, "bk-0", "s-x", "S", "1")
	list, err := svc.CreateReadList(ctx, "u-1", request.CreateReadListDto{Name: "Private"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddBook(ctx, list.ID, "u-1", "bk-0", claims); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GetReadList(ctx, list.ID, "u-2"); err == nil {
		t.Error("another user read the list metadata")
	}
	if _, err := svc.GetReadListBooks(ctx, list.ID, "u-2", claims); err == nil {
		t.Error("another user read the list contents")
	}
	if err := svc.AddBook(ctx, list.ID, "u-2", "bk-0", claims); err == nil {
		t.Error("another user added a book")
	}
	if err := svc.RemoveBook(ctx, list.ID, "u-2", "bk-0"); err == nil {
		t.Error("another user removed a book")
	}
	if err := svc.Reorder(ctx, list.ID, "u-2", []string{"bk-0"}); err == nil {
		t.Error("another user reordered the list")
	}
	if _, err := svc.NextInOrder(ctx, list.ID, "u-2", "", claims); err == nil {
		t.Error("another user walked the list")
	}
	if err := svc.DeleteReadList(ctx, list.ID, "u-2"); err == nil {
		t.Error("another user deleted the list")
	}

	books, err := svc.GetReadListBooks(ctx, list.ID, "u-1", claims)
	if err != nil || len(books) != 1 {
		t.Fatalf("the owner's list did not survive: books=%d err=%v", len(books), err)
	}
}

// Walking off the end is the normal way a read list finishes, so it answers has_next=false rather than an error the reader would have to render as a failure.
func TestNextInOrderWalksToTheEnd(t *testing.T) {
	svc, db, claims := newReadListService(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		seedIssue(t, db, fmt.Sprintf("bk-%d", i), "s-x", "S", fmt.Sprint(i))
	}
	list, err := svc.CreateReadList(ctx, "u-1", request.CreateReadListDto{Name: "Walk"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := svc.AddBook(ctx, list.ID, "u-1", fmt.Sprintf("bk-%d", i), claims); err != nil {
			t.Fatal(err)
		}
	}

	walked := make([]string, 0, 3)
	after := ""
	for step := 0; step < 5; step++ {
		next, err := svc.NextInOrder(ctx, list.ID, "u-1", after, claims)
		if err != nil {
			t.Fatal(err)
		}
		if !next.HasNext {
			break
		}
		if next.Position != int64(len(walked)) {
			t.Errorf("step %d reported position %d", len(walked), next.Position)
		}
		walked = append(walked, next.Book.ID)
		after = next.Book.ID
	}
	if want := []string{"bk-0", "bk-1", "bk-2"}; fmt.Sprint(walked) != fmt.Sprint(want) {
		t.Errorf("walked %v, want %v", walked, want)
	}
}
