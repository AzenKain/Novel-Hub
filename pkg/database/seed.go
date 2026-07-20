package database

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
)

func SeedSuperAdmin(db *sql.DB) error {
	ctx := context.Background()

	displayName, err := config.GetConfig("ADMIN_DISPLAY_NAME")
	if err != nil {
		return err
	}
	email, err := config.GetConfig("ADMIN_EMAIL")
	if err != nil {
		return err
	}
	password, err := config.GetConfig("ADMIN_PASSWORD")
	if err != nil {
		return err
	}

	q := sqlc.New(db)
	_, err = q.GetUserByEmail(ctx, email)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	repo := q.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user, err := repo.UpsertUser(ctx, sqlc.UpsertUserParams{
		Email:        email,
		PasswordHash: sql.NullString{String: string(hashed), Valid: true},
		FullName:     sql.NullString{String: displayName, Valid: displayName != ""},
		AuthProvider: constants.LocalProvider.String(),
	})
	if err != nil {
		return err
	}

	adminRole, err := q.GetRoleByName(ctx, constants.RoleTypeAdmin.String())
	if err != nil {
		return err
	}
	userRole, err := q.GetRoleByName(ctx, constants.RoleTypeUser.String())
	if err != nil {
		return err
	}

	for _, roleID := range []int64{adminRole.ID, userRole.ID} {
		if err := repo.CreateUserRole(ctx, sqlc.CreateUserRoleParams{
			UserID: user.ID,
			RoleID: roleID,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}
