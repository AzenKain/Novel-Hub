package anki

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestGenerateCSV(t *testing.T) {
	cards := []Flashcard{
		{
			Front:   "It was the best of times, it was the worst of times.",
			Back:    "Famous opening line of A Tale of Two Cities.",
			Context: "A Tale of Two Cities - Chapter 1 (Charles Dickens)",
			Tags:    []string{"NovelHub", "Literature"},
		},
		{
			Front:   "Line with\nnewline",
			Back:    "Note with\nnewline",
			Context: "Context 2",
			Tags:    nil,
		},
	}

	csvStr, err := GenerateCSV(cards)
	if err != nil {
		t.Fatalf("GenerateCSV failed: %v", err)
	}

	if !strings.HasPrefix(csvStr, "#separator:tab\n#html:true\n#tags column:4\n") {
		t.Fatalf("unexpected CSV headers:\n%s", csvStr)
	}

	if !strings.Contains(csvStr, "Famous opening line") || !strings.Contains(csvStr, "Charles Dickens") {
		t.Fatalf("CSV missing expected card content:\n%s", csvStr)
	}

	if !strings.Contains(csvStr, "Line with<br>newline") {
		t.Fatalf("CSV did not convert newlines to <br>:\n%s", csvStr)
	}
}

func TestGenerateApkg_Success(t *testing.T) {
	cards := []Flashcard{
		{
			Front:   "Call me Ishmael.",
			Back:    "Opening sentence of Moby-Dick.",
			Context: "Moby-Dick (Herman Melville)",
			Tags:    []string{"Classics"},
		},
		{
			Front:   "All happy families are alike; each unhappy family is unhappy in its own way.",
			Back:    "Opening sentence of Anna Karenina.",
			Context: "Anna Karenina (Leo Tolstoy)",
			Tags:    []string{"RussianLit"},
		},
	}

	apkgBytes, err := GenerateApkg(cards, DeckOptions{
		DeckName:    "NovelHub::World Literature",
		Description: "Highlights and quotes exported from NovelHub",
	})
	if err != nil {
		t.Fatalf("GenerateApkg failed: %v", err)
	}

	if len(apkgBytes) == 0 {
		t.Fatal("GenerateApkg returned empty bytes")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(apkgBytes), int64(len(apkgBytes)))
	if err != nil {
		t.Fatalf("failed to read generated zip: %v", err)
	}

	foundAnki2 := false
	foundMedia := false
	var anki2Data []byte

	for _, f := range zipReader.File {
		if f.Name == "collection.anki2" {
			foundAnki2 = true
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open collection.anki2 in zip: %v", err)
			}
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			anki2Data = buf.Bytes()
		}
		if f.Name == "media" {
			foundMedia = true
		}
	}

	if !foundAnki2 {
		t.Fatal("collection.anki2 not found in .apkg zip")
	}
	if !foundMedia {
		t.Fatal("media file not found in .apkg zip")
	}

	tmpDb, err := os.CreateTemp("", "test-anki-verify-*.db")
	if err != nil {
		t.Fatalf("failed to create temp verify db: %v", err)
	}
	defer os.Remove(tmpDb.Name())

	if _, err := tmpDb.Write(anki2Data); err != nil {
		t.Fatalf("failed to write extracted db: %v", err)
	}
	tmpDb.Close()

	db, err := sql.Open("sqlite", tmpDb.Name())
	if err != nil {
		t.Fatalf("failed to open extracted db: %v", err)
	}
	defer db.Close()

	var noteCount int
	if err := db.QueryRow("SELECT count(*) FROM notes").Scan(&noteCount); err != nil {
		t.Fatalf("failed to query notes count: %v", err)
	}
	if noteCount != 2 {
		t.Fatalf("expected 2 notes in db, got %d", noteCount)
	}

	var cardCount int
	if err := db.QueryRow("SELECT count(*) FROM cards").Scan(&cardCount); err != nil {
		t.Fatalf("failed to query cards count: %v", err)
	}
	if cardCount != 2 {
		t.Fatalf("expected 2 cards in db, got %d", cardCount)
	}

	var decksJSON string
	if err := db.QueryRow("SELECT decks FROM col WHERE id = 1").Scan(&decksJSON); err != nil {
		t.Fatalf("failed to query decks from col: %v", err)
	}
	if !strings.Contains(decksJSON, "NovelHub::World Literature") {
		t.Fatalf("decks JSON does not contain custom deck name: %s", decksJSON)
	}
}

func TestGenerateApkg_EmptyCards(t *testing.T) {
	apkgBytes, err := GenerateApkg([]Flashcard{}, DeckOptions{DeckName: "Empty Deck"})
	if err != nil {
		t.Fatalf("GenerateApkg with empty cards failed: %v", err)
	}
	if len(apkgBytes) == 0 {
		t.Fatal("expected non-empty zip for empty deck")
	}
}
