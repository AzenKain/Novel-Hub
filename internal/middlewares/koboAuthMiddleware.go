package middlewares

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
)

// KoboAuthTokenLocal is where the resolved path token is stashed.
const KoboAuthTokenLocal = "kobo_auth_token"

// KoboAuth authenticates a Kobo device from a token in the URL path.
func KoboAuth(koboRepo repositories.KoboRepository, userRepo repositories.UserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := strings.TrimSpace(c.Params("kobo_token"))
		if token == "" {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Kobo token required"))
		}

		record, err := koboRepo.ResolveToken(c.Context(), token)
		if err != nil || record == nil {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid Kobo token"))
		}

		user, err := userRepo.GetByID(c.Context(), record.UserID)
		if err != nil || user == nil || user.IsDeleted {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid Kobo token"))
		}

		claims := &response.JWTClaims{UId: user.ID, TokenVersion: user.TokenVersion}
		for _, role := range user.Roles {
			if role == nil {
				continue
			}
			claims.RoleIDs = append(claims.RoleIDs, role.ID)
			claims.Roles = append(claims.Roles, constants.RoleType(strings.ToUpper(role.Name)))
		}
		if slices.Contains(claims.Roles, constants.RoleTypeBanned) {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrForbidden, "User account is banned"))
		}

		c.Locals("uid", user.ID)
		c.Locals("user_claims", claims)
		c.Locals(KoboAuthTokenLocal, token)

		_ = koboRepo.TouchToken(c.Context(), token)
		return c.Next()
	}
}

// KoboAuthTokenFrom returns the token the current request authenticated with.
func KoboAuthTokenFrom(c fiber.Ctx) string {
	token, _ := c.Locals(KoboAuthTokenLocal).(string)
	return token
}
