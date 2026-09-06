package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func newRoleSvc(t *testing.T) (RoleService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	c := cache.NewRamCache()
	roleRepo := repositories.NewRoleRepository(db, c)
	svc := NewRoleService(
		roleRepo,
		NewPermissionCache(roleRepo),
		database.NewTxManager(db),
	)
	return svc, db
}

func TestUpdateRolePermissions_CannotModifySystemRoles(t *testing.T) {
	svc, _ := newRoleSvc(t)
	ctx := context.Background()

	systemRoles := []string{
		"01920000-0000-7000-8000-000000000001", // USER
		"01920000-0000-7000-8000-000000000002", // ADMIN
		"01920000-0000-7000-8000-000000000003", // MOD
		"01920000-0000-7000-8000-000000000004", // BANNED
		"01920000-0000-7000-8000-000000000005", // GUEST
	}

	for _, roleID := range systemRoles {
		_, err := svc.UpdateRolePermissions(ctx, roleID, &request.UpdateRolePermissionsDto{
			Permissions: []request.RolePermissionDto{
				{PermissionKey: constants.PermAdminAccess, Effect: "allow"},
			},
		})
		if err == nil {
			t.Fatalf("expected error when modifying permissions of system role %s, got nil", roleID)
		}
	}
}

func TestCreateRole_CannotAutoAssignAdminPermissions(t *testing.T) {
	svc, _ := newRoleSvc(t)
	ctx := context.Background()

	// Should reject creating auto-assign role with admin.access
	_, err := svc.CreateRole(ctx, &request.CreateRoleDto{
		Name:        "AUTO_ADMIN",
		Description: "Malicious auto-assign role",
		AutoAssign:  true,
		Permissions: []request.RolePermissionDto{
			{PermissionKey: constants.PermAdminAccess, Effect: "allow"},
		},
	})
	if err == nil {
		t.Fatal("expected error creating auto-assign role with admin.access, got nil")
	}

	// Should allow creating auto-assign role with regular permissions
	res, err := svc.CreateRole(ctx, &request.CreateRoleDto{
		Name:        "READ_ONLY",
		Description: "Normal auto role",
		AutoAssign:  true,
		Permissions: []request.RolePermissionDto{
			{PermissionKey: constants.PermBookRead, Effect: "allow"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create safe auto-assign role: %v", err)
	}
	if res == nil || res.Name != "READ_ONLY" {
		t.Fatalf("unexpected role response: %+v", res)
	}
}

func TestCreateRole_CannotUseReservedSystemName(t *testing.T) {
	svc, _ := newRoleSvc(t)
	ctx := context.Background()

	reservedNames := []string{"ADMIN", "USER", "MOD", "BANNED", "GUEST"}
	for _, name := range reservedNames {
		_, err := svc.CreateRole(ctx, &request.CreateRoleDto{
			Name:        name,
			Description: "Attempt to shadow system role",
		})
		if err == nil {
			t.Fatalf("expected error creating role with reserved name %s, got nil", name)
		}
	}
}
