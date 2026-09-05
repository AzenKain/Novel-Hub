package repositories

import (
	"context"
	"database/sql"
	"testing"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/constants"
)

// The old LEFT JOIN + DISTINCT became an EXISTS subquery (367ms -> 11ms at 200k users).
func TestSearchUsersDeduplicatesMultiRoleUsers(t *testing.T) {
	repo, db, _ := newUserTestRepo(t)
	ctx := context.Background()

	if _, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "u1", Email: "u1@example.com", AuthProvider: constants.LocalProvider.String(),
	}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"r1", "r2", "r3"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO roles (id, name) VALUES (?, ?)`, role, "role-"+role); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, "u1", role); err != nil {
			t.Fatal(err)
		}
	}

	notDeleted := sql.NullInt64{Int64: 0, Valid: true}
	users, err := repo.Search(ctx, sqlc.SearchUserIDsParams{IsDeleted: notDeleted, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("user with 3 roles must appear once, got %d rows", len(users))
	}

	total, err := repo.Count(ctx, sqlc.CountUsersParams{IsDeleted: notDeleted})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("count must not fan out over roles, got %d", total)
	}
}

// The role filter is why the join existed: match on any one role, exclude without.
func TestSearchUsersRoleFilterMatchesAnyRole(t *testing.T) {
	repo, db, _ := newUserTestRepo(t)
	ctx := context.Background()

	for _, id := range []string{"u1", "u2"} {
		if _, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
			ID: id, Email: id + "@example.com", AuthProvider: constants.LocalProvider.String(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO roles (id, name) VALUES ('r1','role-r1'), ('r2','role-r2')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ('u1','r1'), ('u1','r2'), ('u2','r2')`); err != nil {
		t.Fatal(err)
	}

	r1 := sql.NullString{String: "r1", Valid: true}
	users, err := repo.Search(ctx, sqlc.SearchUserIDsParams{RoleID: r1, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != "u1" {
		t.Fatalf("role filter r1 must return only u1, got %#v", users)
	}

	total, err := repo.Count(ctx, sqlc.CountUsersParams{RoleID: r1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("count with role filter r1 must be 1, got %d", total)
	}
}
