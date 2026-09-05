package services

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
)

func TestBookService_SearchInBookFTS(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, schema := range []string{"../../db/schema/20_books.sql", "../../db/schema/40_search.sql"} {
		contents, err := os.ReadFile(schema)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			t.Fatal(err)
		}
	}

	repo := repositories.NewBookDBRepository(db, cache.NewRamCache())
	for i := int64(0); i < 55; i++ {
		bookID := "book-1"
		chapterID := "chapter-" + string(rune('A'+i))
		chapter := &models.ChapterEntity{ID: chapterID, BookID: bookID, Title: "Chapter", ChapterIndex: i}
		if err := repo.CreateChapter(context.Background(), chapter); err != nil {
			t.Fatal(err)
		}
		if err := repo.InsertFTSChapter(context.Background(), bookID, chapterID, chapter.Title, `<img src=x onerror=alert(1)> alpha OR beta`); err != nil {
			t.Fatal(err)
		}
	}
	other := &models.ChapterEntity{ID: "other-chapter", BookID: "book-2", Title: "Other", ChapterIndex: 0}
	if err := repo.CreateChapter(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertFTSChapter(context.Background(), other.BookID, other.ID, other.Title, "alpha OR beta"); err != nil {
		t.Fatal(err)
	}

	service := &bookService{bookRepo: repo}
	results, err := service.SearchInBook(context.Background(), "book-1", " alpha\tOR\nbeta ")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 50 {
		t.Fatalf("got %d results, want 50", len(results))
	}
	for _, result := range results {
		if result.ChapterID == other.ID {
			t.Fatal("search returned a chapter from another book")
		}
		if result.Snippet == "" || result.Snippet[0] == '<' {
			t.Fatalf("snippet was not safely escaped: %q", result.Snippet)
		}
	}
}

func TestNormalizeFTSQuery(t *testing.T) {
	tests := map[string]string{
		" alpha OR beta ": `"alpha OR beta"`,
		`alpha " beta`:    `"alpha "" beta"`,
		"\x00\n\t":        "",
	}
	for input, want := range tests {
		if got := normalizeFTSQuery(input); got != want {
			t.Errorf("normalizeFTSQuery(%q) = %q, want %q", input, got, want)
		}
	}
}
