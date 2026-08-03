package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

// Fixed UUIDv7 literals from db/schema/90_seed_roles.sql — roles are seeded by the
// schema, so a fresh test DB already has ADMIN/USER/BANNED.
const (
	seedRoleUser   = "01920000-0000-7000-8000-000000000001"
	seedRoleAdmin  = "01920000-0000-7000-8000-000000000002"
	seedRoleBanned = "01920000-0000-7000-8000-000000000004"
)

func newUserSvc(t *testing.T) (UserService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db, filepath.Join("..", "..", "db", "schema")); err != nil {
		t.Fatal(err)
	}
	c := cache.NewRamCache()
	svc := NewUserService(
		repositories.NewUserRepository(db, c),
		repositories.NewRoleRepository(db, c),
		repositories.NewSettingsRepository(db, c),
		database.NewTxManager(db),
	)
	return svc, db
}

func seedOwner(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider, token_version) VALUES ('01920000-0000-7000-8000-000000000bbb','owner@n.h','LOCAL',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ('01920000-0000-7000-8000-000000000bbb', ?)`, seedRoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO setup_state (key, value) VALUES ('root_admin_id','01920000-0000-7000-8000-000000000bbb')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO setup_state (key, value) VALUES ('setup_complete','true')`); err != nil {
		t.Fatal(err)
	}
}

// The owner is seeded from setup_state.root_admin_id. Every privileged mutation must
// refuse to strip or delete that account.
func TestOwnerCannotBeDeletedOrDemoted(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	ownerClaims := &response.JWTClaims{UId: "owner", Roles: []constants.RoleType{constants.RoleTypeAdmin}, TokenVersion: 1}
	if err := svc.DeleteUser(ctx, "01920000-0000-7000-8000-000000000bbb", ownerClaims); err == nil {
		t.Fatal("owner deleted by self")
	}

	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider, token_version) VALUES ('other','o@n.h','LOCAL',1)`); err != nil {
		t.Fatal(err)
	}
	otherClaims := &response.JWTClaims{UId: "other", Roles: []constants.RoleType{constants.RoleTypeAdmin}, TokenVersion: 1}
	if err := svc.DeleteUser(ctx, "01920000-0000-7000-8000-000000000bbb", otherClaims); err == nil {
		t.Fatal("owner deleted by a non-owner admin")
	}

	if _, err := svc.ChangeRoleUser(ctx, "01920000-0000-7000-8000-000000000bbb", otherClaims, &request.ChangeRoleDto{Roles: []string{seedRoleUser}}); err == nil {
		t.Fatal("non-owner demoted the owner")
	}
}

// ChangeRoleUser bumps token_version so the revoked JWT is rejected at the next
// request — claims.Roles cannot be stale because the middleware re-checks the version.
func TestChangeRoleUserBumpsTokenVersion(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider, token_version) VALUES ('01920000-0000-7000-8000-000000000aaa','v@n.h','LOCAL',5)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ('01920000-0000-7000-8000-000000000aaa', ?)`, seedRoleUser); err != nil {
		t.Fatal(err)
	}
	ownerClaims := &response.JWTClaims{UId: "owner", Roles: []constants.RoleType{constants.RoleTypeAdmin}, TokenVersion: 1}

	res, err := svc.ChangeRoleUser(ctx, "01920000-0000-7000-8000-000000000aaa", ownerClaims, &request.ChangeRoleDto{Roles: []string{seedRoleBanned}})
	if err != nil {
		t.Fatal(err)
	}
	if res.TokenVersion != 6 {
		t.Fatalf("token version = %d after role change, want 6", res.TokenVersion)
	}
	var dbVer int64
	if err := db.QueryRow(`SELECT token_version FROM users WHERE id='01920000-0000-7000-8000-000000000aaa'`).Scan(&dbVer); err != nil {
		t.Fatal(err)
	}
	if dbVer != 6 {
		t.Fatalf("DB token_version = %d, want 6", dbVer)
	}
}

// Deleting a user used to leave token_version and refresh_token untouched, so a later
// Restore resurrected every credential captured before the deletion. DeleteUser must now
// bump token_version and clear refresh_token; RestoreUser must bump again so a stale JWT
// from before the restore is rejected.
func TestDeleteUserRevokesCredentials(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	const victim = "01920000-0000-7000-8000-000000000ccc"
	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider, token_version, refresh_token) VALUES (?, 'v@n.h', 'LOCAL', 5, 'old-rt')`, victim); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, victim, seedRoleUser); err != nil {
		t.Fatal(err)
	}
	ownerClaims := &response.JWTClaims{UId: "01920000-0000-7000-8000-000000000bbb", Roles: []constants.RoleType{constants.RoleTypeAdmin}, TokenVersion: 1}

	if err := svc.DeleteUser(ctx, victim, ownerClaims); err != nil {
		t.Fatal(err)
	}

	var tv int64
	var rt sql.NullString
	if err := db.QueryRow(`SELECT token_version, refresh_token FROM users WHERE id=?`, victim).Scan(&tv, &rt); err != nil {
		t.Fatal(err)
	}
	if tv != 6 {
		t.Fatalf("token_version = %d after delete, want 6", tv)
	}
	if rt.Valid && rt.String != "" {
		t.Fatalf("refresh_token = %q after delete, want cleared", rt.String)
	}
}

// A captured refresh token from before the delete must not work after a restore: RestoreUser
// bumps token_version and clears refresh_token again.
func TestRestoreUserDoesNotResurrectCredentials(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	const victim = "01920000-0000-7000-8000-000000000ddd"
	if _, err := db.Exec(`INSERT INTO users (id, email, auth_provider, token_version, refresh_token, is_deleted) VALUES (?, 'v@n.h', 'LOCAL', 5, 'stolen-rt', 1)`, victim); err != nil {
		t.Fatal(err)
	}
	// Pre-deletion refresh token an attacker captured.
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, victim, seedRoleUser); err != nil {
		t.Fatal(err)
	}

	res, err := svc.RestoreUser(ctx, victim)
	if err != nil {
		t.Fatal(err)
	}
	if res.TokenVersion != 6 {
		t.Fatalf("token_version = %d after restore, want 6", res.TokenVersion)
	}

	var rt sql.NullString
	if err := db.QueryRow(`SELECT refresh_token FROM users WHERE id=?`, victim).Scan(&rt); err != nil {
		t.Fatal(err)
	}
	if rt.Valid && rt.String != "" {
		t.Fatalf("refresh_token = %q after restore, want cleared (stolen-rt must not survive)", rt.String)
	}
}
