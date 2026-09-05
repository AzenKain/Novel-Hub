package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
)

func newUserTestRepo(t *testing.T) (UserRepository, *sql.DB, cache.Cache) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	c := cache.NewRamCache()
	return NewUserRepository(db, c), db, c
}

// Token version is the forced-logout mechanism: jwtMiddleware rejects a JWT whose version is below the stored one.
func TestUpdateTokenVersionDefersInvalidationInsideTx(t *testing.T) {
	repo, db, c := newUserTestRepo(t)
	ctx := context.Background()

	created, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "u1", Email: "u1@example.com", AuthProvider: constants.LocalProvider.String(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetTokenVersion(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	tokenKey := cache.BuildKey("user", "token", created.ID)
	var cached int32
	if err := c.Get(ctx, tokenKey, &cached); err != nil {
		t.Fatalf("token version was not cached: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WithTx(tx).UpdateTokenVersion(ctx, created.ID, 7); err != nil {
		t.Fatal(err)
	}

	if err := c.Get(ctx, tokenKey, &cached); err != nil {
		t.Error("cache was invalidated before commit — a concurrent reader can now re-cache the pre-commit version")
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	repo.InvalidateUserCache(ctx, created.ID, created.Email)

	if err := c.Get(ctx, tokenKey, &cached); err == nil {
		t.Errorf("token key survived InvalidateUserCache (value %d) — revoked JWTs stay valid", cached)
	}

	got, err := repo.GetTokenVersion(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 {
		t.Errorf("GetTokenVersion = %d after commit, want 7", got)
	}
}

// CountAdminUsers is the guard behind setup-required.
func TestAdminCountInvalidatedOnRoleChange(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := cache.NewRamCache()
	settingsRepo := NewSettingsRepository(db, c)
	roleRepo := NewRoleRepository(db, c)

	userRepo := NewUserRepository(db, c)
	if _, err := userRepo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "admin-1", Email: "admin@example.com", AuthProvider: constants.LocalProvider.String(),
	}); err != nil {
		t.Fatal(err)
	}
	var adminRoleID string
	if err := db.QueryRowContext(ctx,
		`SELECT id FROM roles WHERE is_admin = 1 LIMIT 1`).Scan(&adminRoleID); err != nil {
		t.Fatal(err)
	}
	if err := roleRepo.CreateUserRole(ctx, "admin-1", adminRoleID); err != nil {
		t.Fatal(err)
	}

	n, err := settingsRepo.CountAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("CountAdminUsers = %d, want 1", n)
	}
	var cachedCount int64
	if err := c.Get(ctx, constants.CacheKeySettingsAdminCount, &cachedCount); err != nil {
		t.Fatalf("admin count was not cached: %v", err)
	}

	if err := roleRepo.BulkDeleteRolesFromUser(ctx, "admin-1"); err != nil {
		t.Fatal(err)
	}

	after, err := settingsRepo.CountAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Errorf("CountAdminUsers = %d after removing the last admin, want 0 — setup would report complete with no admin", after)
	}
}

// GetUsersByIDs has no is_deleted filter, so GetByIDs caches soft-deleted rows under user:id:<id> — the same key GetByID reads.
func TestGetByIDsDoesNotCacheSoftDeletedUsers(t *testing.T) {
	repo, _, _ := newUserTestRepo(t)
	ctx := context.Background()

	u, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "u-del", Email: "del@example.com", AuthProvider: constants.LocalProvider.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetByIDs(ctx, []string{u.ID}); err != nil {
		t.Fatal(err)
	}

	if got, err := repo.GetByID(ctx, u.ID); err == nil {
		t.Fatalf("GetByID returned soft-deleted user %q (is_deleted=%v), want sql.ErrNoRows", got.ID, got.IsDeleted)
	}
}

// UpsertUser on conflict updates full_name/avatar_url of an existing row, but only swept the search/count lists — user:id:<id> and user:email:<email> kept the old profile for the full TTL.
func TestUpsertUserInvalidatesEntityKeys(t *testing.T) {
	repo, _, _ := newUserTestRepo(t)
	ctx := context.Background()

	name := "Old Name"
	if _, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "u-up", Email: "up@example.com", AuthProvider: constants.LocalProvider.String(),
		FullName: sql.NullString{String: name, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByEmail(ctx, "up@example.com"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID: "ignored", Email: "up@example.com", AuthProvider: constants.LocalProvider.String(),
		FullName: sql.NullString{String: "New Name", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByEmail(ctx, "up@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.FullName != "New Name" {
		t.Fatalf("GetByEmail returned stale profile %q, want New Name", got.FullName)
	}
}
