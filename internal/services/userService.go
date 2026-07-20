package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
)

type UserService interface {
	CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, *fiber.Error)
	GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, *fiber.Error)
	UpdateProfile(ctx context.Context, userID string, dto *request.UpdateProfileDto) (*response.UserResponse, *fiber.Error)
	ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) *fiber.Error
	DeleteUser(ctx context.Context, userID string) *fiber.Error
	ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, *fiber.Error)
	RestoreUser(ctx context.Context, userID string) (*response.UserResponse, *fiber.Error)
	GetUserByID(ctx context.Context, userID string) (*response.UserResponse, *fiber.Error)
	SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, *fiber.Error)
	AdminResetPassword(ctx context.Context, userID string, dto *request.ResetPasswordDto) *fiber.Error
}

type userService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	db       *sql.DB
}

func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, db *sql.DB) UserService {
	return &userService{userRepo: userRepo, roleRepo: roleRepo, db: db}
}

func ensureUserRole(roles []*models.RoleEntity) *fiber.Error {
	return nil
}

func parseID(value string) (int64, *fiber.Error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "Invalid ID")
	}
	return id, nil
}

func (u *userService) resolveRoles(ctx context.Context, roleIDs []int64) ([]*models.RoleEntity, *fiber.Error) {
	if len(roleIDs) == 0 {
		autoIDs, err := u.roleRepo.GetAutoAssignRoleIDs(ctx)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get auto roles")
		}
		if len(autoIDs) > 0 {
			return u.resolveRoles(ctx, autoIDs)
		}
		role, err := u.roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
		if err != nil {
			return []*models.RoleEntity{}, nil
		}
		return []*models.RoleEntity{role}, nil
	}
	roles, err := u.roleRepo.GetByIDs(ctx, roleIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch roles")
	}
	if len(roles) != len(roleIDs) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "One or more roles were not found")
	}
	return roles, nil
}

func (u *userService) CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, *fiber.Error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}
	if err := constants.ValidatePassword(dto.Password); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	existing, err := u.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if existing != nil {
		return nil, fiber.NewError(fiber.StatusConflict, "User already exists")
	}

	roles, ferr := u.resolveRoles(ctx, dto.RoleIDs)
	if ferr != nil {
		return nil, ferr
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := u.userRepo.WithTx(tx)
	roleRepoTx := u.roleRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}
	passwordHash := string(hashed)
	var fullName *string
	if dto.FullName != "" {
		fullName = &dto.FullName
	}
	var avatarURL *string
	if dto.AvatarUrl != "" {
		avatarURL = &dto.AvatarUrl
	}

	user, err := userRepoTx.UpsertUser(ctx, repositories.UpsertUserParams{
		Email:        dto.Email,
		PasswordHash: &passwordHash,
		AuthProvider: constants.LocalProvider.String(),
		FullName:     fullName,
		AvatarURL:    avatarURL,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user")
	}

	user.Roles = make([]*models.RoleSimple, 0, len(roles))
	for _, role := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, user.ID, role.ID); err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to assign roles")
		}
		user.Roles = append(user.Roles, role.ToRoleSimple())
	}

	if err := tx.Commit(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit user creation")
	}

	return user.ToResponse(), nil
}

func (u *userService) GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, *fiber.Error) {
	return u.GetUserByID(ctx, userID)
}

func (u *userService) UpdateProfile(ctx context.Context, userID string, dto *request.UpdateProfileDto) (*response.UserResponse, *fiber.Error) {
	id, ferr := parseID(userID)
	if ferr != nil {
		return nil, ferr
	}
	user, err := u.userRepo.UpdateProfile(ctx, repositories.UpdateProfileParams{
		ID:        id,
		FullName:  dto.FullName,
		AvatarURL: dto.AvatarUrl,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update profile")
	}
	return user.ToResponse(), nil
}

func (u *userService) ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) *fiber.Error {
	if err := constants.ValidatePassword(dto.NewPassword); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	id, ferr := parseID(userID)
	if ferr != nil {
		return ferr
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.OldPassword)); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid old password")
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()
	userRepoTx := u.userRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}
	if err := userRepoTx.UpdatePassword(ctx, id, string(hashed)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to revoke sessions")
	}
	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to commit password change")
	}
	return nil
}

func (u *userService) AdminResetPassword(ctx context.Context, userID string, dto *request.ResetPasswordDto) *fiber.Error {
	if err := constants.ValidatePassword(dto.NewPassword); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	id, ferr := parseID(userID)
	if ferr != nil {
		return ferr
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()
	userRepoTx := u.userRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}
	if err := userRepoTx.UpdatePassword(ctx, id, string(hashed)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to revoke sessions")
	}
	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to commit password reset")
	}
	return nil
}

func (u *userService) ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, *fiber.Error) {
	id, ferr := parseID(userID)
	if ferr != nil {
		return nil, ferr
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	roles, ferr := u.resolveRoles(ctx, dto.Roles)
	if ferr != nil {
		return nil, ferr
	}

	hasAdmin := false
	hasBanned := false
	for _, role := range roles {
		if role.IsAdmin || role.Name == constants.RoleTypeAdmin.String() {
			hasAdmin = true
		}
		if role.Name == constants.RoleTypeBanned.String() {
			hasBanned = true
		}
	}

	isSelf := userID == claims.UId
	if isSelf && hasBanned {
		return nil, fiber.NewError(fiber.StatusForbidden, "You cannot ban yourself")
	}
	if slices.Contains(claims.Roles, constants.RoleTypeAdmin) && isSelf && !hasAdmin {
		return nil, fiber.NewError(fiber.StatusForbidden, "You cannot remove your own ADMIN role")
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()
	userRepoTx := u.userRepo.WithTx(tx)
	roleRepoTx := u.roleRepo.WithTx(tx)

	user.Roles = make([]*models.RoleSimple, 0, len(roles))
	for _, role := range roles {
		user.Roles = append(user.Roles, role.ToRoleSimple())
	}

	if err := roleRepoTx.BulkDeleteRolesFromUser(ctx, id); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to clear old roles")
	}
	for _, role := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, id, role.ID); err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to assign roles")
		}
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to revoke sessions")
	}
	user.TokenVersion++

	if err := tx.Commit(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit role change")
	}

	return user.ToResponse(), nil
}

func (u *userService) DeleteUser(ctx context.Context, userID string) *fiber.Error {
	id, ferr := parseID(userID)
	if ferr != nil {
		return ferr
	}
	if err := u.userRepo.Delete(ctx, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete user")
	}
	return nil
}

func (u *userService) RestoreUser(ctx context.Context, userID string) (*response.UserResponse, *fiber.Error) {
	id, ferr := parseID(userID)
	if ferr != nil {
		return nil, ferr
	}
	user, err := u.userRepo.GetByIDWithoutDeleted(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	if err := u.userRepo.Restore(ctx, id); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to restore user")
	}
	user.IsDeleted = false
	return user.ToResponse(), nil
}

func (u *userService) GetUserByID(ctx context.Context, userID string) (*response.UserResponse, *fiber.Error) {
	id, ferr := parseID(userID)
	if ferr != nil {
		return nil, ferr
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	return user.ToResponse(), nil
}

func (u *userService) fillSearchArgs(arg *repositories.UserSearchParams, dto *request.SearchUserDto) {
	if dto.AuthProvider != "" {
		arg.AuthProvider = &dto.AuthProvider
	}
	if dto.CreatedFrom != nil {
		createdFrom := dto.CreatedFrom.Format("2006-01-02 15:04:05")
		arg.CreatedFrom = &createdFrom
	}
	if dto.CreatedTo != nil {
		createdTo := dto.CreatedTo.Format("2006-01-02 15:04:05")
		arg.CreatedTo = &createdTo
	}
	if dto.IsDeleted != nil {
		arg.IsDeleted = dto.IsDeleted
	}
	if len(dto.RoleIDs) > 0 {
		arg.RoleID = &dto.RoleIDs[0]
	}
	if dto.Search != "" {
		arg.SearchText = &dto.Search
	}
}

func (s *userService) SearchUsers(ctx context.Context, params repositories.UserSearchParams) ([]*models.UserEntity, error) {
	return s.userRepo.Search(ctx, params)
}

func (u *userService) SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, *fiber.Error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit == 0 {
		dto.Limit = 20
	}
	offset := (dto.Page - 1) * dto.Limit
	arg := repositories.UserSearchParams{Limit: int64(dto.Limit), Offset: int64(offset)}
	u.fillSearchArgs(&arg, dto)

	var users []*models.UserEntity
	var total int64

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		users, err = u.userRepo.Search(gCtx, arg)
		return err
	})
	g.Go(func() error {
		var err error
		total, err = u.userRepo.Count(gCtx, arg.UserSearchFilter)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search users")
	}

	return response.BuildPaginatedResponse(models.UsersEntityToResponse(users), total, dto.Page, dto.Limit), nil
}
