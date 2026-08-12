package middlewares

import (
	"context"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/database"
)

// ProxyAuth middleware extracts identity from HTTP headers forwarded by a trusted reverse proxy
func ProxyAuth(
	settingsService services.SettingsService,
	authService services.AuthService,
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	txManager database.TxManager,
) fiber.Handler {
	return func(c fiber.Ctx) error {
		// 1. Fetch current settings from cache
		settings, err := settingsService.Admin(c.Context())
		if err != nil || !settings.ProxyAuth.Enabled {
			return c.Next()
		}

		// 2. Verify Trusted Proxy Client IP
		clientIP := c.IP()
		isTrusted := false
		for _, ipStr := range settings.ProxyAuth.TrustedProxies {
			if ipStr == clientIP {
				isTrusted = true
				break
			}
			if _, ipNet, err := net.ParseCIDR(ipStr); err == nil {
				if ip := net.ParseIP(clientIP); ip != nil && ipNet.Contains(ip) {
					isTrusted = true
					break
				}
			}
		}

		if !isTrusted {
			log.Debug().Str("client_ip", clientIP).Msg("ProxyAuth: Client IP is not in trusted proxies list, skipping proxy auth")
			return c.Next()
		}

		// 3. Scan headers for identity
		var identityValue string
		for _, headerName := range settings.ProxyAuth.HeaderNames {
			val := strings.TrimSpace(c.Get(headerName))
			if val != "" {
				identityValue = val
				break
			}
		}

		if identityValue == "" {
			return c.Next()
		}

		// Normalize to valid email address if it's only a username
		emailVal := strings.ToLower(identityValue)
		if !strings.Contains(emailVal, "@") {
			emailVal = emailVal + "@proxy.local"
		}

		// 4. Retrieve or Provision User
		user, err := userRepo.GetByEmail(c.Context(), emailVal)
		if err != nil && !apperrors.IsNotFound(err) {
			log.Error().Err(err).Str("email", emailVal).Msg("ProxyAuth: Failed to query user by email")
			return c.Next()
		}

		if user == nil {
			if !settings.ProxyAuth.AutoCreate {
				log.Warn().Str("email", emailVal).Msg("ProxyAuth: User not found and auto_create is disabled")
				return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
					Status:  false,
					Message: "Proxy authentication failed: user not found",
				})
			}

			// Auto-provision user
			user, err = autoProvisionUser(c.Context(), emailVal, userRepo, roleRepo, txManager)
			if err != nil {
				log.Error().Err(err).Str("email", emailVal).Msg("ProxyAuth: Failed to auto-provision user")
				return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
					Status:  false,
					Message: "Failed to automatically provision user",
				})
			}
			log.Info().Str("email", emailVal).Str("user_id", user.ID).Msg("ProxyAuth: Automatically provisioned user from proxy headers")
		}

		if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
			return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
				Status:  false,
				Message: "User account is banned",
			})
		}

		// 5. Generate Auth Tokens and Inject
		authRes, err := authService.GenToken(user)
		if err != nil {
			log.Error().Err(err).Str("user_id", user.ID).Msg("ProxyAuth: Failed to generate authentication token")
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
				Status:  false,
				Message: "Failed to generate session tokens",
			})
		}

		// Inject Authorization header for downstream JwtAccess middleware
		c.Request().Header.Set("Authorization", "Bearer "+authRes.AccessToken)

		// Set access/refresh tokens in browser cookies
		secure := c.Scheme() == "https"
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    authRes.AccessToken,
			Expires:  time.Now().Add(constants.AccessTokenDuration),
			MaxAge:   int(constants.AccessTokenDuration.Seconds()),
			HTTPOnly: true,
			Secure:   secure,
			SameSite: "Lax",
			Path:     "/",
		})
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    authRes.RefreshToken,
			Expires:  time.Now().Add(constants.RefreshTokenDuration),
			MaxAge:   int(constants.RefreshTokenDuration.Seconds()),
			HTTPOnly: true,
			Secure:   secure,
			SameSite: "Lax",
			Path:     "/",
		})

		return c.Next()
	}
}

func autoProvisionUser(
	ctx context.Context,
	email string,
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	txManager database.TxManager,
) (*models.UserEntity, error) {
	autoIDs, err := roleRepo.GetAutoAssignRoleIDs(ctx)
	if err != nil {
		return nil, err
	}

	var roles []*models.RoleEntity
	if len(autoIDs) > 0 {
		roles, err = roleRepo.GetByIDs(ctx, autoIDs)
		if err != nil {
			return nil, err
		}
	} else {
		role, err := roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
		if err != nil {
			return nil, err
		}
		roles = []*models.RoleEntity{role}
	}

	tx, err := txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepoTx := userRepo.WithTx(tx)
	roleRepoTx := roleRepo.WithTx(tx)

	// Create user with empty password (cannot login via normal password forms)
	user, err := userRepoTx.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Email:        email,
		PasswordHash: convert.StrPtrToNullString(nil),
		AuthProvider: constants.LocalProvider.String(),
		FullName:     convert.StrPtrToNullString(nil),
	})
	if err != nil {
		return nil, err
	}

	user.Roles = make([]*models.RoleSimple, 0, len(roles))
	for _, r := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, user.ID, r.ID); err != nil {
			return nil, err
		}
		user.Roles = append(user.Roles, r.ToRoleSimple())
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}
