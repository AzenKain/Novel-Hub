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

// KoboAuthTokenLocal is where the resolved path token is stashed. Handlers need it to build
// URLs the device will call back on (library_sync, cover templates), because those URLs must
// embed the same token.
const KoboAuthTokenLocal = "kobo_auth_token"

// KoboAuth authenticates a Kobo device from a token in the URL path.
//
// This deliberately does NOT use JwtAccess. A Kobo reader exposes exactly one configurable
// setting — api_endpoint in Kobo eReader.conf — and sends no Authorization header, so JWT
// auth rejects every request a real device makes. calibre-web solved this by embedding a
// random token in the endpoint URL the user pastes into the device (cps/kobo_auth.py:
// "We pretty much ignore all of the above" about the real UserKey/Bearer flow), and this
// mirrors that.
//
// Security shape, stated plainly because it differs from the rest of the API:
//   - possession of the URL is the credential; there is no second factor
//   - it grants only the Kobo endpoints, and only for the user it maps to
//   - it is revoked by regenerating (which replaces the row) or deleting it
func KoboAuth(koboRepo repositories.KoboRepository, userRepo repositories.UserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := strings.TrimSpace(c.Params("kobo_token"))
		if token == "" {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Kobo token required"))
		}

		record, err := koboRepo.ResolveToken(c.Context(), token)
		if err != nil || record == nil {
			// Deliberately vague: the token is a secret, and distinguishing "no such token"
			// from "token for a deleted user" would let someone probe for valid ones.
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
		// A banned user's device stops syncing, same as the JWT path (jwtMiddleware.go).
		// Checked by role NAME, not role.IsBanned: GetUserRoles does not select is_banned, so
		// that field is always false on a hydrated user and a check against it never fires.
		if slices.Contains(claims.Roles, constants.RoleTypeBanned) {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrForbidden, "User account is banned"))
		}

		c.Locals("uid", user.ID)
		c.Locals("user_claims", claims)
		c.Locals(KoboAuthTokenLocal, token)

		// Best-effort: a failed touch must not break syncing, it only costs the "last used"
		// hint in the UI.
		_ = koboRepo.TouchToken(c.Context(), token)
		return c.Next()
	}
}

// KoboAuthTokenFrom returns the token the current request authenticated with.
func KoboAuthTokenFrom(c fiber.Ctx) string {
	token, _ := c.Locals(KoboAuthTokenLocal).(string)
	return token
}
