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

// CountActiveAdminUsers is the guard that stops the last admin from being removed. It
// joined roles without filtering is_deleted, so an admin whose only admin role was
// soft-deleted still counted. roleService.DeleteRole refuses is_admin roles today, so
// this is reachable only by a caller that goes straight to the repository — the guard
// query should not depend on that check living one layer up.
func TestCountActiveAdminUsersIgnoresDeletedRoles(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRoleRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider) VALUES ('u1','a@b.c','LOCAL')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO roles (id, name, is_admin, is_system, position) VALUES ('r1','CUSTOM_ADMIN',1,0,5)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ('u1','r1')`); err != nil {
		t.Fatal(err)
	}

	n, err := repo.CountActiveAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("baseline CountActiveAdminUsers = %d, want 1", n)
	}

	if err := repo.Delete(ctx, "r1"); err != nil {
		t.Fatal(err)
	}

	n, err = repo.CountActiveAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("CountActiveAdminUsers = %d after its only admin role was deleted, want 0", n)
	}
}
