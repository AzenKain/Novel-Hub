package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"

	"novelhub/pkg/convert"
)

type UserService interface {
	CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, error)
	GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, error)
	UpdateProfile(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.UpdateProfileDto) (*response.UserResponse, error)
	ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) error
	DeleteUser(ctx context.Context, userID string, claims *response.JWTClaims) error
	ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, error)
	RestoreUser(ctx context.Context, userID string) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, userID string) (*response.UserResponse, error)
	SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error)
	AdminResetPassword(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ResetPasswordDto) error
}

type userService struct {
	userRepo     repositories.UserRepository
	roleRepo     repositories.RoleRepository
	settingsRepo repositories.SettingsRepository
	txManager    database.TxManager
}

func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, settingsRepo repositories.SettingsRepository, txManager database.TxManager) UserService {
	return &userService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		settingsRepo: settingsRepo,
		txManager:    txManager,
	}
}

// rootAdminID returns the owner account id recorded during setup, or "" when it
// cannot be determined. Returning "" is safe: every "only the owner may ..."
// check below compares the caller against this value, so an unknown owner makes
// those comparisons fail and the privileged action is denied. The owner is
// always an admin, so the admin guards still protect the owner account itself.
func (u *userService) rootAdminID(ctx context.Context) string {
	if u.settingsRepo == nil {
		return ""
	}
	id, err := u.settingsRepo.GetSetupState(ctx, "root_admin_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(id)
}

func (u *userService) resolveRoles(ctx context.Context, roleIDs []string) ([]*models.RoleEntity, error) {
	if len(roleIDs) == 0 {
		autoIDs, err := u.roleRepo.GetAutoAssignRoleIDs(ctx)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get auto roles")
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
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch roles")
	}
	if len(roles) != len(roleIDs) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "One or more roles were not found")
	}
	return roles, nil
}

func (u *userService) CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid email format")
	}
	if err := constants.ValidatePassword(dto.Password); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	existing, err := u.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.ErrConflict, "User already exists")
	}

	roles, ferr := u.resolveRoles(ctx, dto.RoleIDs)
	if ferr != nil {
		return nil, ferr
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := u.userRepo.WithTx(tx)
	roleRepoTx := u.roleRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
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

	user, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Email:        dto.Email,
		PasswordHash: convert.StrPtrToNullString(&passwordHash),
		AuthProvider: constants.LocalProvider.String(),
		FullName:     convert.StrPtrToNullString(fullName),
		AvatarUrl:    convert.StrPtrToNullString(avatarURL),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create user")
	}

	user.Roles = make([]*models.RoleSimple, 0, len(roles))
	for _, role := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, user.ID, role.ID); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign roles")
		}
		user.Roles = append(user.Roles, role.ToRoleSimple())
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user creation")
	}

	return user.ToResponse(), nil
}

func (u *userService) GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, error) {
	return u.GetUserByID(ctx, userID)
}

func (u *userService) UpdateProfile(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.UpdateProfileDto) (*response.UserResponse, error) {
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}

	rootID := u.rootAdminID(ctx)
	callerID := claims.UId
	isRoot := rootID != "" && callerID == rootID
	if rootID != "" && id == rootID && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify the owner account")
	}

	userObj, err := u.userRepo.GetByID(ctx, id)
	if err == nil && userObj != nil {
		if userObj.IsAdmin() && !isRoot && userID != claims.UId {
			return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify other admin accounts")
		}
	}

	user, err := u.userRepo.UpdateProfile(ctx, sqlc.UpdateProfileParams{
		ID:        id,
		FullName:  convert.StrPtrToNullString(dto.FullName),
		AvatarUrl: convert.StrPtrToNullString(dto.AvatarUrl),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update profile")
	}
	res := user.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) error {
	if err := constants.ValidatePassword(dto.NewPassword); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetAuthByID(ctx, id)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.OldPassword)); err != nil {
		return apperrors.New(apperrors.ErrUnauthorized, "Invalid old password")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()
	userRepoTx := u.userRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}
	if err := userRepoTx.UpdatePassword(ctx, id, string(hashed)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit password change")
	}
	u.userRepo.InvalidateUserCache(ctx, id, user.Email)
	return nil
}

func (u *userService) AdminResetPassword(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ResetPasswordDto) error {
	if err := constants.ValidatePassword(dto.NewPassword); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims.UId == rootID
	if rootID != "" && id == rootID && !isRoot {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can modify the owner account")
	}

	if user.IsAdmin() && !isRoot && userID != claims.UId {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can modify other admin accounts")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()
	userRepoTx := u.userRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}
	if err := userRepoTx.UpdatePassword(ctx, id, string(hashed)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit password reset")
	}
	u.userRepo.InvalidateUserCache(ctx, id, user.Email)
	return nil
}
func (u *userService) ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, error) {
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims.UId == rootID
	if rootID != "" && id == rootID && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify the owner account")
	}

	targetIsAdmin := false
	for _, r := range user.Roles {
		if r != nil && (r.IsAdmin || r.Name == "ADMIN") {
			targetIsAdmin = true
		}
	}

	if targetIsAdmin && !isRoot && userID != claims.UId {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify other admin accounts")
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

	if hasAdmin && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can grant the ADMIN role")
	}

	isSelf := userID == claims.UId
	if isSelf && hasBanned {
		return nil, apperrors.New(apperrors.ErrForbidden, "You cannot ban yourself")
	}
	if slices.Contains(claims.Roles, constants.RoleTypeAdmin) && isSelf && !hasAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "You cannot remove your own ADMIN role")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
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
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to clear old roles")
	}
	for _, role := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, id, role.ID); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign roles")
		}
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	user.TokenVersion++

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit role change")
	}
	u.userRepo.InvalidateUserCache(ctx, id, user.Email)

	res := user.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) DeleteUser(ctx context.Context, userID string, claims *response.JWTClaims) error {
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}

	rootID := u.rootAdminID(ctx)
	if rootID != "" && id == rootID {
		return apperrors.New(apperrors.ErrForbidden, "The owner account cannot be deleted")
	}

	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	isRoot := rootID != "" && claims.UId == rootID
	if user.IsAdmin() && !isRoot {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can delete other admin accounts")
	}

	// Soft-deleting a user hides the row from GetUserTokenVersion (which filters
	// is_deleted = 0), so the attacker's captured JWT/refresh token fail. But the row keeps
	// the same token_version and refresh_token — a Restore flips is_deleted back to 0 and
	// resurrects every credential captured before the deletion. Bump token_version and
	// clear refresh_token on the way down so a future restore cannot un-revoke them.
	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	userRepoTx := u.userRepo.WithTx(tx)

	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, id, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to clear refresh token")
	}
	if err := userRepoTx.Delete(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete user")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit deletion")
	}
	u.userRepo.InvalidateUserCache(ctx, id, user.Email)
	return nil
}

func (u *userService) RestoreUser(ctx context.Context, userID string) (*response.UserResponse, error) {
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByIDWithoutDeleted(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	// Restore flips is_deleted back to 0 with the SAME token_version and refresh_token.
	// DeleteUser now bumps token_version and clears refresh_token on the way down, so the
	// restored row already carries revoked credentials. To stay safe even against a
	// deletion made before that guard existed, bump again here: any JWT captured before
	// this restore is rejected, and the cleared refresh_token stops rotation.
	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	userRepoTx := u.userRepo.WithTx(tx)

	if err := userRepoTx.Restore(ctx, id); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to restore user")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, id, nil); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to clear refresh token")
	}
	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit restore")
	}
	u.userRepo.InvalidateUserCache(ctx, id, user.Email)

	user.IsDeleted = false
	user.TokenVersion++
	user.RefreshToken = ""
	return user.ToResponse(), nil
}

func (u *userService) GetUserByID(ctx context.Context, userID string) (*response.UserResponse, error) {
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}
	res := user.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

// markOwner flags the root admin so the UI can gate owner-only actions without
// inferring ownership from the id.
func (u *userService) markOwner(ctx context.Context, users ...*response.UserResponse) {
	rootID := u.rootAdminID(ctx)
	if rootID == "" {
		return
	}
	for _, user := range users {
		if user != nil && user.ID == rootID {
			user.IsOwner = true
		}
	}
}

func (u *userService) fillSearchArgs(dto *request.SearchUserDto) (sqlc.SearchUserIDsParams, sqlc.CountUsersParams) {
	var isDeleted any
	if dto.IsDeleted != nil {
		if *dto.IsDeleted {
			isDeleted = int64(1)
		} else {
			isDeleted = int64(0)
		}
	}
	var roleID any
	if len(dto.RoleIDs) > 0 {
		roleID = dto.RoleIDs[0]
	}
	var authProvider any
	if dto.AuthProvider != "" {
		authProvider = dto.AuthProvider
	}
	var createdFrom any
	if dto.CreatedFrom != nil {
		createdFrom = dto.CreatedFrom.Format("2006-01-02 15:04:05")
	}
	var createdTo any
	if dto.CreatedTo != nil {
		createdTo = dto.CreatedTo.Format("2006-01-02 15:04:05")
	}
	var searchText any
	if dto.Search != "" {
		searchText = dto.Search
	}

	countParams := sqlc.CountUsersParams{
		IsDeleted:    isDeleted,
		RoleID:       roleID,
		AuthProvider: authProvider,
		CreatedFrom:  createdFrom,
		CreatedTo:    createdTo,
		SearchText:   searchText,
	}

	searchParams := sqlc.SearchUserIDsParams{
		IsDeleted:    isDeleted,
		RoleID:       roleID,
		AuthProvider: authProvider,
		CreatedFrom:  createdFrom,
		CreatedTo:    createdTo,
		SearchText:   searchText,
	}

	return searchParams, countParams
}

func (u *userService) SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit <= 0 || dto.Limit > 100 {
		dto.Limit = 20
	}
	searchParams, countParams := u.fillSearchArgs(dto)

	if dto.Cursor != "" {
		parts := convert.DecodeCursor(dto.Cursor)
		if len(parts) == 2 {
			searchParams.CursorCreatedAt = parts[0]
			if parts[1] != "" {
				// UUIDv7 is lexicographically time-ordered, so `id > cursor` still pages correctly.
				searchParams.CursorID = sql.NullString{String: parts[1], Valid: true}
			}
		}
	}
	searchParams.Limit = int64(dto.Limit)

	var users []*models.UserEntity
	var total int64

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		users, err = u.userRepo.Search(gCtx, searchParams)
		return err
	})
	g.Go(func() error {
		var err error
		total, err = u.userRepo.Count(gCtx, countParams)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to search users")
	}

	var nextCursor string
	if len(users) > 0 {
		lastUser := users[len(users)-1]
		nextCursor = convert.EncodeCursor(lastUser.CreatedAt, lastUser.ID)
	}
	items := models.UsersEntityToResponse(users)
	u.markOwner(ctx, items...)
	return response.BuildCursorPaginatedResponse(items, total, dto.Limit, nextCursor), nil
}
