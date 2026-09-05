package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/constants"
)

// FTS must return exactly what LIKE did.
func TestUserSearchFTSMatchesLikeSemantics(t *testing.T) {
	repo, db, _ := newUserTestRepo(t)
	ctx := context.Background()

	seed := []struct{ id, email, name string }{
		{"u1", "omiya.yuu@example.com", "Omiya Yuu"},
		{"u2", "nguyen.van.a@corp.local", "Nguyen Van A"},
		{"u3", "zz@gmail.com", "Person With \"Quotes\""},
		{"u4", "ab@gmail.com", "AB Short"},
		{"u5", "starred@gmail.com", "Star * Name"},
	}
	for _, s := range seed {
		if _, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
			ID: s.id, Email: s.email, FullName: sql.NullString{String: s.name, Valid: true},
			AuthProvider: constants.LocalProvider.String(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	notDeleted := sql.NullInt64{Int64: 0, Valid: true}
	terms := []string{
		"omiya",
		"miya",
		"gmail",
		"zzqx",
		"ab",
		"a",
		`"`,
		`a"b`,
		"Star *",
		"OR 1=1",
		"NEAR(a)",
		"u1",
		"nguyen",
	}

	for _, term := range terms {
		t.Run(fmt.Sprintf("term=%q", term), func(t *testing.T) {
			wantIDs, err := db.QueryContext(ctx, `SELECT u.id FROM users u
				WHERE (u.is_deleted = 0)
				AND (CAST(u.id AS TEXT) LIKE '%'||?1||'%'
				  OR lower(u.email) LIKE '%'||lower(?1)||'%'
				  OR lower(COALESCE(u.full_name,'')) LIKE '%'||lower(?1)||'%')
				ORDER BY u.created_at DESC, u.id ASC LIMIT 50`, term)
			if err != nil {
				t.Fatal(err)
			}
			var want []string
			for wantIDs.Next() {
				var id string
				if err := wantIDs.Scan(&id); err != nil {
					t.Fatal(err)
				}
				want = append(want, id)
			}
			if err := wantIDs.Err(); err != nil {
				t.Fatal(err)
			}
			wantIDs.Close()

			got, err := repo.Search(ctx, sqlc.SearchUserIDsParams{
				IsDeleted: notDeleted, SearchText: term, Limit: 50,
			})
			if err != nil {
				t.Fatalf("search failed: %v", err)
			}
			gotIDs := make([]string, 0, len(got))
			for _, u := range got {
				gotIDs = append(gotIDs, u.ID)
			}
			if fmt.Sprint(gotIDs) != fmt.Sprint(want) {
				t.Errorf("rows differ from LIKE baseline:\n  want %v\n  got  %v", want, gotIDs)
			}

			total, err := repo.Count(ctx, sqlc.CountUsersParams{
				IsDeleted: notDeleted, SearchText: term,
			})
			if err != nil {
				t.Fatalf("count failed: %v", err)
			}
			if int(total) != len(want) {
				t.Errorf("count %d does not match LIKE baseline %d", total, len(want))
			}
		})
	}
}

// Drives the DB directly and clears the cache by hand: the triggers are what's under test, and repository-level invalidation would mask a stale index.
func TestUserSearchFTSStaysInSyncOnUpdateAndDelete(t *testing.T) {
	repo, db, c := newUserTestRepo(t)
	ctx := context.Background()

	if _, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "u1", Email: "before@example.com",
		FullName:     sql.NullString{String: "Before Name", Valid: true},
		AuthProvider: constants.LocalProvider.String(),
	}); err != nil {
		t.Fatal(err)
	}

	find := func(term string) int {
		_ = c.DelByPattern(ctx, "user:search*")
		users, err := repo.Search(ctx, sqlc.SearchUserIDsParams{SearchText: term, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		return len(users)
	}

	if n := find("before"); n != 1 {
		t.Fatalf("baseline search should find the seeded user, got %d", n)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET email = 'after@example.com', full_name = 'After Name' WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	if n := find("before"); n != 0 {
		t.Errorf("stale FTS row: old email still matches (%d hits)", n)
	}
	if n := find("after"); n != 1 {
		t.Errorf("updated email not indexed (%d hits)", n)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	if n := find("after"); n != 0 {
		t.Errorf("deleted user still in FTS index (%d hits)", n)
	}
}
