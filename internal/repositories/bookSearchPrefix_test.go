package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func TestBookSearchPrefixAndMiddleWords(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "search_prefix.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO books (id, library_id, title) VALUES
		('b1', 'lib-1', 'Assassin''s Creed: Renaissance'),
		('b2', 'lib-1', 'Hôm nay là thứ 6'),
		('b3', 'lib-1', 'Harry Potter và Hòn đá Phù thủy'),
		('b4', 'lib-1', 'Chúa tể những chiếc nhẫn')
	`); err != nil {
		t.Fatal(err)
	}

	repo := NewBookDBRepository(db, cache.NewRamCache())

	tests := []struct {
		name        string
		query       string
		wantBookIDs []string
	}{
		{
			name:        "Incomplete word (assass -> Assassin's Creed)",
			query:       "assass",
			wantBookIDs: []string{"b1"},
		},
		{
			name:        "Middle word in sentence (thứ -> Hôm nay là thứ 6)",
			query:       "thứ",
			wantBookIDs: []string{"b2"},
		},
		{
			name:        "Middle word in sentence (thủy -> Harry Potter và Hòn đá Phù thủy)",
			query:       "thủy",
			wantBookIDs: []string{"b3"},
		},
		{
			name:        "Middle word (là -> Hôm nay là thứ 6)",
			query:       "là",
			wantBookIDs: []string{"b2"},
		},
		{
			name:        "Multiple words disordered (creed assassin -> Assassin's Creed)",
			query:       "creed assassin",
			wantBookIDs: []string{"b1"},
		},
		{
			name:        "First and last words (hôm 6 -> Hôm nay là thứ 6)",
			query:       "hôm 6",
			wantBookIDs: []string{"b2"},
		},
		{
			name:        "Incomplete multi-word (harry pot -> Harry Potter)",
			query:       "harry pot",
			wantBookIDs: []string{"b3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := tc.query
			res, err := repo.SearchBooks(ctx, nil, &q, "", "", "", "", "", "", "", 20, "u1")
			if err != nil {
				t.Fatalf("SearchBooks(%q) error: %v", tc.query, err)
			}
			if len(res) != len(tc.wantBookIDs) {
				t.Fatalf("SearchBooks(%q) got %d results, want %d", tc.query, len(res), len(tc.wantBookIDs))
			}
			for i, wantID := range tc.wantBookIDs {
				if res[i].ID != wantID {
					t.Errorf("SearchBooks(%q) result[%d] = %s, want %s", tc.query, i, res[i].ID, wantID)
				}
			}
		})
	}
}

func BenchmarkBookSearchPrefix(b *testing.B) {
	dir := b.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "search_bench.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO libraries (id, name) VALUES ('lib-1', 'Main')`); err != nil {
		b.Fatal(err)
	}

	// Seed 500 books
	tx, _ := db.Begin()
	st, _ := tx.Prepare(`INSERT INTO books (id, library_id, title) VALUES (?, 'lib-1', ?)`)
	for i := 0; i < 500; i++ {
		st.Exec("b-"+string(rune(i)), "Assassin's Book Collection Volume "+string(rune(i)))
	}
	st.Close()
	tx.Commit()

	repo := NewBookDBRepository(db, nil)
	query := "assass"

	
	b.ReportAllocs()
	for b.Loop() {
		_, err := repo.SearchBooks(ctx, nil, &query, "", "", "", "", "", "", "", 20, "u1")
		if err != nil {
			b.Fatal(err)
		}
	}
}
