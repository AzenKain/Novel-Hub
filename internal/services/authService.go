package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type AuthService interface {
	Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, *fiber.Error)
	Register(ctx context.Context, dto *request.RegisterDto) (*response.UserResponse, *fiber.Error)
	SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.UserResponse, *fiber.Error)
	RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, *fiber.Error)
	Logout(ctx context.Context, userID string) *fiber.Error
}

type authService struct {
	userRepo     repositories.UserRepository
	roleRepo     repositories.RoleRepository
	settingsRepo repositories.SettingsRepository
	db           *sql.DB
	settings     SettingsService
}

func NewAuthService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, db *sql.DB, settingsRepo repositories.SettingsRepository, settings SettingsService) AuthService {
	return &authService{userRepo: userRepo, roleRepo: roleRepo, db: db, settingsRepo: settingsRepo, settings: settings}
}

func (a *authService) genToken(user *models.UserEntity) (*response.AuthResponse, *fiber.Error) {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Missing JWT_SECRET")
	}
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Missing JWT_REFRESH_SECRET")
	}

	roles := models.RolesEntityToRoleConstant(user.Roles)
	roleIDs := models.RolesEntityToRoleIDs(user.Roles)
	claimsAccess := &response.JWTClaims{
		UId:          strconv.FormatInt(user.ID, 10),
		Roles:        roles,
		RoleIDs:      roleIDs,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(constants.AccessTokenDuration)),
		},
	}
	claimsRefresh := &response.JWTClaims{
		UId:          strconv.FormatInt(user.ID, 10),
		Roles:        roles,
		RoleIDs:      roleIDs,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(constants.RefreshTokenDuration)),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsAccess)
	access, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to sign access token")
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsRefresh)
	refresh, err := refreshToken.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to sign refresh token")
	}

	return &response.AuthResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (a *authService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, *fiber.Error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if user == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid email or password")
	}
	if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
		return nil, fiber.NewError(fiber.StatusForbidden, "User account is banned")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password)); err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid email or password")
	}

	tokens, tokenErr := a.genToken(user)
	if tokenErr != nil {
		return nil, tokenErr
	}

	if err := a.userRepo.UpdateRefreshToken(ctx, user.ID, &tokens.RefreshToken); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
	}

	return tokens, nil
}

func (a *authService) Register(ctx context.Context, dto *request.RegisterDto) (*response.UserResponse, *fiber.Error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}
	if err := constants.ValidatePassword(dto.Password); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	settings, err := a.settings.Public(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to load settings")
	}
	if !settings.RegistrationEnabled {
		return nil, fiber.NewError(fiber.StatusForbidden, "Public registration is disabled")
	}

	existing, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if existing != nil {
		return nil, fiber.NewError(fiber.StatusConflict, "User already exists")
	}

	autoIDs, err := a.roleRepo.GetAutoAssignRoleIDs(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get auto roles")
	}
	var roles []*models.RoleEntity
	if len(autoIDs) > 0 {
		roles, err = a.roleRepo.GetByIDs(ctx, autoIDs)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch roles")
		}
	} else {
		role, err := a.roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get default role")
		}
		roles = []*models.RoleEntity{role}
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	roleRepoTx := a.roleRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}
	passwordHash := string(hashed)
	var fullName *string
	if dto.FullName != "" {
		fullName = &dto.FullName
	}

	user, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		Email:        dto.Email,
		PasswordHash: convert.StrPtrToNullString(&passwordHash),
		AuthProvider: constants.LocalProvider.String(),
		FullName:     convert.StrPtrToNullString(fullName),
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit user registration")
	}

	return user.ToResponse(), nil
}

func (a *authService) SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.UserResponse, *fiber.Error) {
	if a.settingsRepo == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Settings repository not configured")
	}
	if a.settings != nil {
		// Trust the service's notion of setup completion, which also accounts
		// for whether an administrator account exists. This lets the wizard
		// run again on databases left in an inconsistent state (flag set but no
		// admin) instead of being permanently blocked.
		if !a.settings.SetupRequired(ctx) {
			return nil, fiber.NewError(fiber.StatusForbidden, "Setup has already been completed")
		}
	} else {
		completed, err := a.settingsRepo.GetSetupState(ctx, "completed")
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to check setup state")
		}
		if completed == "true" {
			return nil, fiber.NewError(fiber.StatusForbidden, "Setup has already been completed")
		}
	}
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}
	if err := constants.ValidatePassword(dto.Password); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	roleRepoTx := a.roleRepo.WithTx(tx)
	settingsRepoTx := a.settingsRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}
	passwordHash := string(hashed)
	var fullName *string
	if dto.Username != "" {
		fullName = &dto.Username
	}

	user, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		Email:        dto.Email,
		PasswordHash: convert.StrPtrToNullString(&passwordHash),
		AuthProvider: constants.LocalProvider.String(),
		FullName:     convert.StrPtrToNullString(fullName),
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create root user")
	}

	adminRole, err := roleRepoTx.GetByName(ctx, constants.RoleTypeAdmin.String())
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get admin role")
	}
	if err := roleRepoTx.CreateUserRole(ctx, user.ID, adminRole.ID); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to assign admin role")
	}
	user.Roles = []*models.RoleSimple{{ID: adminRole.ID, Name: adminRole.Name}}

	settings := buildSetupSettings(dto)
	for key, value := range settings {
		data, err := jsonx.Marshal(value)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to encode setting")
		}
		if err := settingsRepoTx.Upsert(ctx, key, string(data)); err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to save setting")
		}
	}

	if err := settingsRepoTx.UpsertSetupState(ctx, "completed", "true"); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to complete setup")
	}

	if err := tx.Commit(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit setup")
	}

	if a.settings != nil {
		_ = a.settings.Reload(ctx)
	}

	return user.ToResponse(), nil
}

func buildSetupSettings(dto *request.SetupDto) map[string]any {
	settings := map[string]any{"auth.registration_enabled": dto.Registration}

	if dto.SiteTitle != "" {
		settings["site.title"] = dto.SiteTitle
	}
	if dto.SiteDescription != "" {
		settings["site.description"] = dto.SiteDescription
	}
	if dto.Favicon != "" {
		settings["site.favicon"] = dto.Favicon
	}
	if dto.Logo != "" {
		settings["site.logo"] = dto.Logo
	}

	guestMode := dto.GuestMode
	if guestMode == "" {
		guestMode = "all"
	}
	settings["guest_access.mode"] = guestMode
	if len(dto.GuestLibraryIDs) > 0 {
		settings["guest_access.library_ids"] = dto.GuestLibraryIDs
	}

	for _, p := range []struct{ mode, libraryIDs string }{
		{dto.DownloadMode, "download"},
		{dto.BookmarkMode, "bookmark"},
		{dto.CollectionMode, "collection"},
		{dto.ReviewMode, "review"},
	} {
		mode := p.mode
		if mode == "" {
			mode = "all"
		}
		settings[p.libraryIDs+".mode"] = mode
	}

	if len(dto.SidebarVisibleItems) > 0 {
		settings["sidebar.visible_items"] = dto.SidebarVisibleItems
	}

	return settings
}

func (a *authService) RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, *fiber.Error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || id < 1 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID")
	}
	user, err := a.userRepo.GetByID(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if user == nil || user.RefreshToken == "" || user.RefreshToken != refreshToken {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid refresh token")
	}
	if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
		return nil, fiber.NewError(fiber.StatusForbidden, "User account is banned")
	}

	tokens, tokenErr := a.genToken(user)
	if tokenErr != nil {
		return nil, tokenErr
	}
	if err := a.userRepo.UpdateRefreshToken(ctx, id, &tokens.RefreshToken); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
	}

	return tokens, nil
}

func (a *authService) Logout(ctx context.Context, userID string) *fiber.Error {
	id, parseErr := strconv.ParseInt(userID, 10, 64)
	if parseErr != nil || id < 1 {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	user, err := a.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to revoke session")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, id, nil); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to clear refresh token")
	}
	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to commit logout")
	}
	return nil
}
