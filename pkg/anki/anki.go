package anki

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"novelhub/pkg/jsonx"
)

// Flashcard represents a single flashcard item derived from a highlight or note.
type Flashcard struct {
	Front   string
	Back    string
	Context string
	Tags    []string
}

// DeckOptions contains metadata for the generated deck.
type DeckOptions struct {
	DeckName    string
	Description string
}

// GenerateCSV generates an Anki-compatible TSV/CSV format with UTF-8 encoding.
func GenerateCSV(cards []Flashcard) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("#separator:tab\n")
	buf.WriteString("#html:true\n")
	buf.WriteString("#tags column:4\n")

	writer := csv.NewWriter(&buf)
	writer.Comma = '\t'

	for _, card := range cards {
		front := strings.TrimSpace(card.Front)
		if front == "" {
			continue
		}
		frontHTML := strings.ReplaceAll(html.EscapeString(front), "\n", "<br>")
		backHTML := strings.ReplaceAll(html.EscapeString(strings.TrimSpace(card.Back)), "\n", "<br>")
		if backHTML == "" {
			backHTML = "—"
		}
		contextHTML := strings.ReplaceAll(html.EscapeString(strings.TrimSpace(card.Context)), "\n", "<br>")

		tagsStr := strings.Join(card.Tags, " ")
		if tagsStr == "" {
			tagsStr = "NovelHub"
		}

		if err := writer.Write([]string{frontHTML, backHTML, contextHTML, tagsStr}); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GenerateApkg creates a fully compliant Anki 2.0 package (.apkg) containing an SQLite database and media map.
func GenerateApkg(cards []Flashcard, opts DeckOptions) ([]byte, error) {
	if opts.DeckName == "" {
		opts.DeckName = "NovelHub Reading Deck"
	}

	tmpFile, err := os.CreateTemp("", "anki-col-*.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp sqlite file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	db, err := sql.Open("sqlite", tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp sqlite: %w", err)
	}
	defer db.Close()

	if err := initAnkiSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init anki schema: %w", err)
	}

	now := time.Now()
	nowSec := now.Unix()
	nowMilli := now.UnixMilli()

	deckID := nowMilli
	modelID := nowMilli + 1

	if err := insertCollection(db, nowSec, nowMilli, deckID, modelID, opts); err != nil {
		return nil, fmt.Errorf("failed to insert collection: %w", err)
	}

	if err := insertCardsAndNotes(db, cards, deckID, modelID, nowSec, nowMilli); err != nil {
		return nil, fmt.Errorf("failed to insert notes and cards: %w", err)
	}

	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("failed to close sqlite: %w", err)
	}

	dbBytes, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sqlite file: %w", err)
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	anki2Entry, err := zipWriter.Create("collection.anki2")
	if err != nil {
		return nil, fmt.Errorf("failed to create collection.anki2 entry: %w", err)
	}
	if _, err := anki2Entry.Write(dbBytes); err != nil {
		return nil, fmt.Errorf("failed to write collection.anki2 data: %w", err)
	}

	mediaEntry, err := zipWriter.Create("media")
	if err != nil {
		return nil, fmt.Errorf("failed to create media entry: %w", err)
	}
	if _, err := mediaEntry.Write([]byte("{}")); err != nil {
		return nil, fmt.Errorf("failed to write media data: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}

	return zipBuf.Bytes(), nil
}

func initAnkiSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE col (
		id              integer primary key,
		crt             integer not null,
		mod             integer not null,
		scm             integer not null,
		ver             integer not null,
		dty             integer not null,
		usn             integer not null,
		ls              integer not null,
		conf            text not null,
		models          text not null,
		decks           text not null,
		dconf           text not null,
		tags            text not null
	);

	CREATE TABLE notes (
		id              integer primary key,
		guid            text not null,
		mid             integer not null,
		mod             integer not null,
		usn             integer not null,
		tags            text not null,
		flds            text not null,
		sfld            text not null,
		csum            integer not null,
		flags           integer not null,
		data            text not null
	);

	CREATE TABLE cards (
		id              integer primary key,
		nid             integer not null,
		did             integer not null,
		ord             integer not null,
		mod             integer not null,
		usn             integer not null,
		type            integer not null,
		queue           integer not null,
		due             integer not null,
		ivl             integer not null,
		factor          integer not null,
		reps            integer not null,
		lapses          integer not null,
		left            integer not null,
		odue            integer not null,
		odid            integer not null,
		flags           integer not null,
		data            text not null
	);

	CREATE TABLE revlog (
		id              integer primary key,
		cid             integer not null,
		usn             integer not null,
		ease            integer not null,
		ivl             integer not null,
		lastIvl         integer not null,
		factor          integer not null,
		time            integer not null,
		type            integer not null
	);

	CREATE TABLE graves (
		usn             integer not null,
		oid             integer not null,
		type            integer not null
	);

	CREATE INDEX IF NOT EXISTS ix_notes_usn ON notes (usn);
	CREATE INDEX IF NOT EXISTS ix_cards_usn ON cards (usn);
	CREATE INDEX IF NOT EXISTS ix_cards_nid ON cards (nid);
	CREATE INDEX IF NOT EXISTS ix_cards_sched ON cards (did, queue, due);
	CREATE INDEX IF NOT EXISTS ix_revlog_usn ON revlog (usn);
	CREATE INDEX IF NOT EXISTS ix_revlog_cid ON revlog (cid);
	`
	_, err := db.Exec(schema)
	return err
}

func insertCollection(db *sql.DB, nowSec, nowMilli, deckID, modelID int64, opts DeckOptions) error {
	conf := map[string]any{
		"activeDecks":   []int64{1},
		"addToCur":      true,
		"collapseTime":  1200,
		"curDeck":       1,
		"curModel":      fmt.Sprintf("%d", modelID),
		"dueCounts":     true,
		"estTimes":      true,
		"newBury":       true,
		"newSpread":     0,
		"nextPos":       1,
		"sortBackwards": false,
		"sortType":      "noteFld",
		"timeLim":       0,
	}
	confJSON, _ := jsonx.MarshalString(conf)

	dconf := map[string]any{
		"1": map[string]any{
			"id":       1,
			"mod":      0,
			"name":     "Default",
			"usn":      0,
			"maxTaken": 60,
			"autoplay": true,
			"timer":    0,
			"replayq":  true,
			"new": map[string]any{
				"bury":          true,
				"delays":        []int{1, 10},
				"initialFactor": 2500,
				"ints":          []int{1, 4, 7},
				"order":         1,
				"perDay":        20,
				"separate":      true,
			},
			"lapse": map[string]any{
				"delays":      []int{10},
				"leechAction": 0,
				"leechFails":  8,
				"minInt":      1,
				"mult":        0,
			},
			"rev": map[string]any{
				"bury":     true,
				"ease4":    1.3,
				"fuzz":     0.05,
				"ivlFct":   1,
				"maxIvl":   36500,
				"perDay":   100,
				"minSpace": 1,
			},
		},
	}
	dconfJSON, _ := jsonx.MarshalString(dconf)

	decks := map[string]any{
		"1": map[string]any{
			"collapsed": false,
			"conf":      1,
			"desc":      "",
			"dyn":       0,
			"extendNew": 10,
			"extendRev": 50,
			"id":        1,
			"lrnToday":  []int{0, 0},
			"mod":       nowSec,
			"name":      "Default",
			"newToday":  []int{0, 0},
			"revToday":  []int{0, 0},
			"timeToday": []int{0, 0},
			"usn":       0,
		},
		fmt.Sprintf("%d", deckID): map[string]any{
			"collapsed": false,
			"conf":      1,
			"desc":      opts.Description,
			"dyn":       0,
			"extendNew": 10,
			"extendRev": 50,
			"id":        deckID,
			"lrnToday":  []int{0, 0},
			"mod":       nowSec,
			"name":      opts.DeckName,
			"newToday":  []int{0, 0},
			"revToday":  []int{0, 0},
			"timeToday": []int{0, 0},
			"usn":       -1,
		},
	}
	decksJSON, _ := jsonx.MarshalString(decks)

	cardCSS := `.card {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  font-size: 16px;
  line-height: 1.6;
  text-align: left;
  color: #1f2937;
  background-color: #ffffff;
  padding: 20px 24px;
  border-radius: 12px;
  max-width: 650px;
  margin: 0 auto;
}
.front { margin-bottom: 12px; }
.quote-mark { font-size: 28px; line-height: 1; color: #3b82f6; font-family: Georgia, serif; margin-bottom: -6px; }
.quote-text { font-size: 17px; font-weight: 500; color: #111827; }
.context { font-size: 12px; color: #6b7280; margin-top: 10px; padding-top: 8px; border-top: 1px dashed #e5e7eb; }
hr#answer { border: none; border-top: 1px solid #e5e7eb; margin: 16px 0; }
.back { margin-top: 10px; }
.note-label { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: #3b82f6; margin-bottom: 4px; }
.note-content { font-size: 15px; color: #374151; background-color: #f8fafc; padding: 10px 14px; border-left: 3px solid #3b82f6; border-radius: 4px; }
.nightMode.card { background-color: #1f2937; color: #f3f4f6; }
.nightMode .quote-text { color: #f9fafb; }
.nightMode .context { color: #9ca3af; border-color: #374151; }
.nightMode hr#answer { border-color: #374151; }
.nightMode .note-content { background-color: #111827; color: #e5e7eb; border-left-color: #60a5fa; }`

	qfmt := `<div class="card front">
  <div class="quote-mark">“</div>
  <div class="quote-text">{{Front}}</div>
</div>
{{#Context}}
<div class="context">📖 {{Context}}</div>
{{/Context}}`

	afmt := `{{FrontSide}}
<hr id="answer">
<div class="card back">
  <div class="note-label">📝 Note / Reflection</div>
  <div class="note-content">{{Back}}</div>
</div>`

	models := map[string]any{
		fmt.Sprintf("%d", modelID): map[string]any{
			"id":        modelID,
			"name":      "NovelHub Highlight Card",
			"type":      0,
			"mod":       nowSec,
			"usn":       -1,
			"sortf":     0,
			"did":       deckID,
			"latexPre":  "\\documentclass[12pt]{article}\n\\special{papersize=3in,5in}\n\\usepackage[utf8]{inputenc}\n\\usepackage{amssymb,amsmath}\n\\pagestyle{empty}\n\\setlength{\\parindent}{0in}\n\\begin{document}\n",
			"latexPost": "\\end{document}",
			"latexsvg":  false,
			"req":       [][]any{{0, "all", []int{0}}},
			"tags":      []any{},
			"vers":      []any{},
			"css":       cardCSS,
			"tmpls": []map[string]any{
				{
					"name":  "Card 1",
					"ord":   0,
					"qfmt":  qfmt,
					"afmt":  afmt,
					"bqfmt": "",
					"bafmt": "",
					"bfont": "",
					"bsize": 0,
					"did":   nil,
				},
			},
			"flds": []map[string]any{
				{"name": "Front", "ord": 0, "sticky": false, "rtl": false, "font": "Liberation Sans", "size": 20, "media": []any{}},
				{"name": "Back", "ord": 1, "sticky": false, "rtl": false, "font": "Liberation Sans", "size": 16, "media": []any{}},
				{"name": "Context", "ord": 2, "sticky": false, "rtl": false, "font": "Liberation Sans", "size": 13, "media": []any{}},
			},
		},
	}
	modelsJSON, _ := jsonx.MarshalString(models)

	_, err := db.Exec(`
		INSERT INTO col (id, crt, mod, scm, ver, dty, usn, ls, conf, models, decks, dconf, tags)
		VALUES (1, ?, ?, ?, 11, 0, 0, 0, ?, ?, ?, ?, '{}')
	`, nowSec, nowMilli, nowMilli, confJSON, modelsJSON, decksJSON, dconfJSON)

	return err
}

func insertCardsAndNotes(db *sql.DB, cards []Flashcard, deckID, modelID, nowSec, nowMilli int64) error {
	noteStmt, err := db.Prepare(`
		INSERT INTO notes (id, guid, mid, mod, usn, tags, flds, sfld, csum, flags, data)
		VALUES (?, ?, ?, ?, -1, ?, ?, ?, ?, 0, '')
	`)
	if err != nil {
		return err
	}
	defer noteStmt.Close()

	cardStmt, err := db.Prepare(`
		INSERT INTO cards (id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data)
		VALUES (?, ?, ?, 0, ?, -1, 0, 0, ?, 0, 0, 0, 0, 0, 0, 0, 0, '')
	`)
	if err != nil {
		return err
	}
	defer cardStmt.Close()

	cardPos := 1
	for i, card := range cards {
		front := strings.TrimSpace(card.Front)
		if front == "" {
			continue
		}

		frontHTML := strings.ReplaceAll(html.EscapeString(front), "\n", "<br>")
		back := strings.TrimSpace(card.Back)
		if back == "" {
			back = "—"
		}
		backHTML := strings.ReplaceAll(html.EscapeString(back), "\n", "<br>")
		contextHTML := strings.ReplaceAll(html.EscapeString(strings.TrimSpace(card.Context)), "\n", "<br>")

		noteID := nowMilli + int64(i*2)
		cardID := nowMilli + int64(i*2+1)

		guid := generateGUID()
		tagsStr := " " + strings.Join(card.Tags, " ") + " "
		if strings.TrimSpace(tagsStr) == "" {
			tagsStr = " NovelHub "
		}

		flds := frontHTML + "\x1f" + backHTML + "\x1f" + contextHTML
		sfld := front
		h := sha1.Sum([]byte(sfld))
		csum := int64(binary.BigEndian.Uint32(h[:4]))

		if _, err := noteStmt.Exec(noteID, guid, modelID, nowSec, tagsStr, flds, sfld, csum); err != nil {
			return err
		}

		if _, err := cardStmt.Exec(cardID, noteID, deckID, nowSec, cardPos); err != nil {
			return err
		}
		cardPos++
	}

	return nil
}

func generateGUID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
