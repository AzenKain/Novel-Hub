package services

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
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
	"novelhub/pkg/bookparser"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/database"
	"novelhub/pkg/localfs"

	"novelhub/pkg/convert"
	"novelhub/pkg/worker"
)

type UserService interface {
	CreateUser(ctx context.Context, dto *request.CreateUserDto) (*response.UserResponse, error)
	GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, error)
	UpdateProfile(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.UpdateProfileDto) (*response.UserResponse, error)
	ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) error
	DeleteUser(ctx context.Context, userID string, claims *response.JWTClaims) error
	ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, error)
	RestoreUser(ctx context.Context, userID string, claims *response.JWTClaims) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, userID string) (*response.UserResponse, error)
	SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error)
	AdminResetPassword(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ResetPasswordDto) error
	SendEmail(ctx context.Context, userID string, dto *request.SendUserEmailDto) error
	ExecuteSendUserEmailJob(ctx context.Context, payloadJSON string) error
	SetJobQueue(jobQueue *worker.Queue)
	UploadAvatar(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (string, error)
}

type userService struct {
	userRepo     repositories.UserRepository
	roleRepo     repositories.RoleRepository
	settingsRepo repositories.SettingsRepository
	txManager    database.TxManager
	settings     SettingsService
	permissions  PermissionCache
	jobQueue     *worker.Queue
}

func (u *userService) SetJobQueue(jobQueue *worker.Queue) {
	u.jobQueue = jobQueue
}

func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, settingsRepo repositories.SettingsRepository, txManager database.TxManager, permissions PermissionCache, settings ...SettingsService) UserService {
	var settingsService SettingsService
	if len(settings) > 0 {
		settingsService = settings[0]
	}
	return &userService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		settingsRepo: settingsRepo,
		txManager:    txManager,
		settings:     settingsService,
		permissions:  permissions,
	}
}

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
	if err != nil && !apperrors.IsNotFound(err) {
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
	isRoot := rootID != "" && claims.UId == rootID
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
	u.describeRoles(res)
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
	u.describeRoles(res)
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

func (u *userService) RestoreUser(ctx context.Context, userID string, claims *response.JWTClaims) (*response.UserResponse, error) {
	id, ferr := convert.ParseID(userID)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid ID")
	}
	user, err := u.userRepo.GetByIDWithoutDeleted(ctx, id)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := claims != nil && rootID != "" && claims.UId == rootID
	if user.IsAdmin() && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can restore other admin accounts")
	}

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
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}
	res := user.ToResponse()
	u.markOwner(ctx, res)
	u.describeRoles(res)
	return res, nil
}

// The frontend evaluates permissions locally, so the payload has to carry what each role grants.
// GetUserRoles projects only id and name, which left hasPermission() returning false for every
// custom role — the name === "ADMIN" shortcut in permission.ts was all that still worked.
// Read from the permission cache rather than the database: it is the same snapshot the server's
// own checks use, so the two cannot disagree, and no query lands on the auth path.
func (u *userService) describeRoles(users ...*response.UserResponse) {
	if u.permissions == nil {
		return
	}
	for _, user := range users {
		if user == nil || len(user.Roles) == 0 {
			continue
		}
		ids := make([]string, 0, len(user.Roles))
		for _, role := range user.Roles {
			if role != nil {
				ids = append(ids, role.ID)
			}
		}
		described := u.permissions.DescribeRoles(ids)
		if len(described) == 0 {
			continue
		}
		roles := make([]*response.RoleSimpleResponse, 0, len(described))
		for _, role := range described {
			roles = append(roles, role.ToResponse())
		}
		user.Roles = roles
	}
}

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
			cursorTime := convert.CursorTimeString(parts[0])
			searchParams.CursorCreatedAt = convert.StrPtrToNullStringNonEmpty(&cursorTime)
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
	// Only a full page can have anything after it. Emitting a cursor for a short page — as this
	// did for every non-empty result — renders one more page in the admin list that is always
	// empty. Every sibling list guards the same way (auditService, metadataService, opdsService).
	if len(users) > 0 && len(users) == dto.Limit {
		lastUser := users[len(users)-1]
		nextCursor = convert.EncodeCursor(lastUser.CreatedAt, lastUser.ID)
	}
	items := models.UsersEntityToResponse(users)
	u.markOwner(ctx, items...)
	u.describeRoles(items...)
	return response.BuildCursorPaginatedResponse(items, total, dto.Limit, nextCursor), nil
}

func (s *userService) UploadAvatar(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "No file uploaded")
	}

	limit := s.settings.Limits().SiteAssetBytes
	if fileHeader.Size > limit {
		return "", apperrors.New(apperrors.ErrBadRequest, "Uploaded file exceeds size limit")
	}

	f, err := fileHeader.Open()
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to open uploaded file")
	}
	defer f.Close()

	fileData, err := io.ReadAll(f)
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to read uploaded file")
	}

	ext, err := bookparser.ValidateImage(fileData, limit)
	if err != nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "Uploaded file must be a valid JPEG, PNG, or GIF image")
	}

	publicDir := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to create public directory")
	}

	// Always overwrite the same file to prevent accumulation of old user avatars (junk files)
	outFilename := fmt.Sprintf("avatar_%s%s", userID, ext)
	destPath, err := localfs.SafeJoin(publicDir, outFilename)
	if err != nil {
		return "", apperrors.New(apperrors.ErrBadRequest, "Invalid file destination path")
	}

	if err := os.WriteFile(destPath, fileData, 0644); err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to save avatar file")
	}

	id, ferr := convert.ParseID(userID)
	if ferr == nil {
		if oldUser, err := s.userRepo.GetByID(ctx, id); err == nil && oldUser != nil && oldUser.AvatarUrl != "" {
			newAvatarURL := "/public/" + outFilename
			// If extension has changed, delete the old file with the old extension
			if oldUser.AvatarUrl != newAvatarURL && strings.HasPrefix(oldUser.AvatarUrl, "/public/avatar_") {
				oldFilename := filepath.Base(oldUser.AvatarUrl)
				oldPath, err := localfs.SafeJoin(publicDir, oldFilename)
				if err == nil {
					_ = os.Remove(oldPath)
				}
			}
		}
	}

	return "/public/" + outFilename, nil
}
