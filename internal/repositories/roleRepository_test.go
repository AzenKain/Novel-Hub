package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// CountActiveAdminUsers is the guard that stops the last admin from being removed.
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

func TestRoleEntityCompositeCachingAndPermissionInvalidation(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ramCache := cache.NewRamCache()
	repo := NewRoleRepository(db, ramCache)
	ctx := context.Background()

	if _, err := db.Exec(`
		INSERT OR IGNORE INTO permissions (key, description) VALUES 
		('book.read', 'Permission to read books'),
		('book.write', 'Permission to write books'),
		('book.delete', 'Permission to delete books');
		INSERT INTO roles (id, name, is_admin, is_system, position) VALUES ('r_test','TEST_ROLE',0,0,10);
	`); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceRolePermissions(ctx, "r_test", []*models.RolePermissionEntity{
		{
			PermissionKey:  "book.read",
			Effect:         "allow",
			ConditionsJSON: "{}",
		},
	}); err != nil {
		t.Fatal(err)
	}

	role1, err := repo.GetByID(ctx, "r_test")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if len(role1.Permissions) != 1 || role1.Permissions[0].PermissionKey != "book.read" {
		t.Fatalf("expected 1 permission (book.read), got %v", role1.Permissions)
	}

	roles, err := repo.GetByIDs(ctx, []string{"r_test"})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}
	if len(roles) != 1 || len(roles[0].Permissions) != 1 || roles[0].Permissions[0].PermissionKey != "book.read" {
		t.Fatalf("expected cached role with book.read, got %v", roles[0].Permissions)
	}

	if err := repo.ReplaceRolePermissions(ctx, "r_test", []*models.RolePermissionEntity{
		{
			PermissionKey:  "book.write",
			Effect:         "allow",
			ConditionsJSON: "{}",
		},
		{
			PermissionKey:  "book.delete",
			Effect:         "allow",
			ConditionsJSON: "{}",
		},
	}); err != nil {
		t.Fatalf("ReplaceRolePermissions failed: %v", err)
	}

	role2, err := repo.GetByID(ctx, "r_test")
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if len(role2.Permissions) != 2 {
		t.Fatalf("expected 2 updated permissions, got %d", len(role2.Permissions))
	}
	permKeys := map[string]bool{
		role2.Permissions[0].PermissionKey: true,
		role2.Permissions[1].PermissionKey: true,
	}
	if !permKeys["book.write"] || !permKeys["book.delete"] {
		t.Fatalf("expected permissions [book.write, book.delete], got %v", role2.Permissions)
	}

	rolesUpdated, err := repo.GetByIDs(ctx, []string{"r_test"})
	if err != nil {
		t.Fatalf("GetByIDs after update failed: %v", err)
	}
	if len(rolesUpdated) != 1 || len(rolesUpdated[0].Permissions) != 2 {
		t.Fatalf("expected 2 permissions from GetByIDs, got %d", len(rolesUpdated[0].Permissions))
	}

	roleByName, err := repo.GetByName(ctx, "TEST_ROLE")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if len(roleByName.Permissions) != 2 {
		t.Fatalf("expected 2 permissions from GetByName, got %d", len(roleByName.Permissions))
	}
}
