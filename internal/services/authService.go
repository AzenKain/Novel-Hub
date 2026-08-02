package services

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
)

type AuthService interface {
	Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error)
	ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*response.JWTClaims, error)
	Register(ctx context.Context, dto *request.RegisterDto) (*response.UserResponse, error)
	SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.UserResponse, error)
	RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, error)
	Logout(ctx context.Context, userID string) error
}

type authService struct {
	userRepo     repositories.UserRepository
	roleRepo     repositories.RoleRepository
	settingsRepo repositories.SettingsRepository
	txManager    database.TxManager
	settings     SettingsService
}

func NewAuthService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, txManager database.TxManager, settingsRepo repositories.SettingsRepository, settings SettingsService) AuthService {
	return &authService{userRepo: userRepo, roleRepo: roleRepo, txManager: txManager, settingsRepo: settingsRepo, settings: settings}
}

func refreshTokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func refreshTokenMatches(stored, token string) bool {
	actual := token
	if len(stored) >= len("sha256:") && stored[:len("sha256:")] == "sha256:" {
		actual = refreshTokenDigest(token)
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(actual)) == 1
}

func tokenClaims(user *models.UserEntity, tokenType string, duration time.Duration) *response.JWTClaims {
	now := time.Now()
	uid := user.ID
	return &response.JWTClaims{
		UId:          uid,
		Roles:        models.RolesEntityToRoleConstant(user.Roles),
		RoleIDs:      models.RolesEntityToRoleIDs(user.Roles),
		TokenVersion: user.TokenVersion,
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "novelhub",
			Subject:   uid,
			Audience:  jwt.ClaimStrings{"novelhub-" + tokenType},
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.Must(uuid.NewV7()).String(),
		},
	}
}

func (a *authService) genToken(user *models.UserEntity) (*response.AuthResponse, error) {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Missing JWT_SECRET")
	}
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Missing JWT_REFRESH_SECRET")
	}

	claimsAccess := tokenClaims(user, "access", constants.AccessTokenDuration)
	claimsRefresh := tokenClaims(user, "refresh", constants.RefreshTokenDuration)

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsAccess)
	access, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to sign access token")
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsRefresh)
	refresh, err := refreshToken.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to sign refresh token")
	}

	return &response.AuthResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (a *authService) authenticate(ctx context.Context, dto *request.SignInDto) (*models.UserEntity, error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid email format")
	}
	user, err := a.userRepo.GetAuthByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password)) != nil {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid email or password")
	}
	if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
		return nil, apperrors.New(apperrors.ErrForbidden, "User account is banned")
	}
	return user, nil
}

func (a *authService) ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*response.JWTClaims, error) {
	user, err := a.authenticate(ctx, dto)
	if err != nil {
		return nil, err
	}
	return tokenClaims(user, "access", constants.AccessTokenDuration), nil
}

func (a *authService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error) {
	user, err := a.authenticate(ctx, dto)
	if err != nil {
		return nil, err
	}

	tokens, tokenErr := a.genToken(user)
	if tokenErr != nil {
		return nil, tokenErr
	}

	refreshDigest := refreshTokenDigest(tokens.RefreshToken)
	if err := a.userRepo.UpdateRefreshToken(ctx, user.ID, &refreshDigest); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update refresh token")
	}

	return tokens, nil
}

func (a *authService) Register(ctx context.Context, dto *request.RegisterDto) (*response.UserResponse, error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid email format")
	}
	if err := constants.ValidatePassword(dto.Password); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	settings, err := a.settings.Public(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load settings")
	}
	if !settings.RegistrationEnabled {
		return nil, apperrors.New(apperrors.ErrForbidden, "Public registration is disabled")
	}

	existing, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.ErrConflict, "User already exists")
	}

	autoIDs, err := a.roleRepo.GetAutoAssignRoleIDs(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get auto roles")
	}
	var roles []*models.RoleEntity
	if len(autoIDs) > 0 {
		roles, err = a.roleRepo.GetByIDs(ctx, autoIDs)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch roles")
		}
	} else {
		role, err := a.roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get default role")
		}
		roles = []*models.RoleEntity{role}
	}

	tx, err := a.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	roleRepoTx := a.roleRepo.WithTx(tx)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}
	passwordHash := string(hashed)
	var fullName *string
	if dto.FullName != "" {
		fullName = &dto.FullName
	}

	user, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Email:        dto.Email,
		PasswordHash: convert.StrPtrToNullString(&passwordHash),
		AuthProvider: constants.LocalProvider.String(),
		FullName:     convert.StrPtrToNullString(fullName),
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
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user registration")
	}

	return user.ToResponse(), nil
}

func (a *authService) SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.UserResponse, error) {
	if a.settingsRepo == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Settings repository not configured")
	}
	if a.settings != nil {
		if !a.settings.SetupRequired(ctx) {
			return nil, apperrors.New(apperrors.ErrForbidden, "Setup has already been completed")
		}
	} else {
		completed, err := a.settingsRepo.GetSetupState(ctx, "completed")
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to check setup state")
		}
		if completed == "true" {
			return nil, apperrors.New(apperrors.ErrForbidden, "Setup has already been completed")
		}
	}
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid email format")
	}
	if err := constants.ValidatePassword(dto.Password); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	tx, err := a.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	roleRepoTx := a.roleRepo.WithTx(tx)
	settingsRepoTx := a.settingsRepo.WithTx(tx)

	claimed, err := settingsRepoTx.ClaimInitialSetup(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to claim initial setup")
	}
	if !claimed {
		return nil, apperrors.New(apperrors.ErrForbidden, "Setup has already been completed or is in progress")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}
	passwordHash := string(hashed)
	var fullName *string
	if dto.Username != "" {
		fullName = &dto.Username
	}

	user, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Email:        dto.Email,
		PasswordHash: convert.StrPtrToNullString(&passwordHash),
		AuthProvider: constants.LocalProvider.String(),
		FullName:     convert.StrPtrToNullString(fullName),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create root user")
	}

	adminRole, err := roleRepoTx.GetByName(ctx, constants.RoleTypeAdmin.String())
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get admin role")
	}
	if err := roleRepoTx.CreateUserRole(ctx, user.ID, adminRole.ID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign admin role")
	}
	user.Roles = []*models.RoleSimple{{ID: adminRole.ID, Name: adminRole.Name}}

	settings := buildSetupSettings(dto)
	for key, value := range settings {
		data, err := jsonx.Marshal(value)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to encode setting")
		}
		if err := settingsRepoTx.Upsert(ctx, key, string(data)); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save setting")
		}
	}

	if err := settingsRepoTx.UpsertSetupState(ctx, "completed", "true"); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to complete setup")
	}
	// Root admin identity is derived from this record, so a lost write leaves the install ownerless.
	if err := settingsRepoTx.UpsertSetupState(ctx, "root_admin_id", user.ID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to record root admin")
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit setup")
	}

	if a.settings != nil {
		_ = a.settings.Reload(ctx)
	}

	return user.ToResponse(), nil
}

func buildSetupSettings(dto *request.SetupDto) map[string]any {
	settings := map[string]any{
		"auth.registration_enabled": dto.Registration,
		"auth.login_required":        dto.LoginRequired,
	}

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

	if len(dto.SidebarVisibleItems) > 0 {
		settings["sidebar.visible_items"] = dto.SidebarVisibleItems
	}

	return settings
}

func (a *authService) RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, error) {
	id, err := convert.ParseID(userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid user ID")
	}
	user, err := a.userRepo.GetAuthByID(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil || user.RefreshToken == "" || !refreshTokenMatches(user.RefreshToken, refreshToken) {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid refresh token")
	}
	if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
		return nil, apperrors.New(apperrors.ErrForbidden, "User account is banned")
	}

	tokens, tokenErr := a.genToken(user)
	if tokenErr != nil {
		return nil, tokenErr
	}
	rotated, err := a.userRepo.RotateRefreshToken(ctx, id, user.RefreshToken, refreshTokenDigest(tokens.RefreshToken))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update refresh token")
	}
	if !rotated {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Refresh token has already been used")
	}

	return tokens, nil
}

func (a *authService) Logout(ctx context.Context, userID string) error {
	id, parseErr := convert.ParseID(userID)
	if parseErr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid user ID")
	}
	tx, err := a.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := a.userRepo.WithTx(tx)
	user, err := a.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	if err := userRepoTx.UpdateTokenVersion(ctx, id, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke session")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, id, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to clear refresh token")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit logout")
	}
	return nil
}
