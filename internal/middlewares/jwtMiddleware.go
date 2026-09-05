package middlewares

import (
	"errors"
	"slices"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

func JwtAccess(userRepo repositories.UserRepository) fiber.Handler {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return func(c fiber.Ctx) error {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "JWT_SECRET is missing"})
		}
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(jwtSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess(userRepo, false, "access"),
		Extractor: extractors.Chain(
			extractors.FromAuthHeader("Bearer"),
			extractors.FromCookie("access_token"),
		),
		Claims: &response.JWTClaims{},
	})
}

func GuestClaims() *response.JWTClaims {
	return &response.JWTClaims{
		UId:   "0",
		Roles: []constants.RoleType{constants.RoleTypeGuest},
	}
}

func continueAsGuest(c fiber.Ctx) error {
	claims := GuestClaims()
	c.Locals("user_claims", claims)
	c.Locals("uid", claims.UId)
	return c.Next()
}

// No token is a guest; a token that fails to parse is a failed login and must 401, or the frontend refresh interceptor never fires.
func OptionalJwtAccess(userRepo repositories.UserRepository) fiber.Handler {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return continueAsGuest
	}

	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(jwtSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if errors.Is(err, extractors.ErrNotFound) {
				return continueAsGuest(c)
			}
			return jwtError(c, err)
		},
		SuccessHandler: jwtSuccess(userRepo, true, "access"),
		Extractor: extractors.Chain(
			extractors.FromAuthHeader("Bearer"),
			extractors.FromCookie("access_token"),
		),
		Claims: &response.JWTClaims{},
	})
}

func JwtRefresh(userRepo repositories.UserRepository) fiber.Handler {
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return func(c fiber.Ctx) error {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "JWT_REFRESH_SECRET is missing"})
		}
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(jwtRefreshSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess(userRepo, false, "refresh"),
		Extractor: extractors.Chain(
			extractors.FromCookie("refresh_token"),
			extractors.FromAuthHeader("Bearer"),
		),
		Claims: &response.JWTClaims{},
	})
}

func jwtSuccess(userRepo repositories.UserRepository, optional bool, expectedType string) fiber.Handler {
	return func(c fiber.Ctx) error {
		unauthorized := func() error {
			return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Invalid or missing token"})
		}

		jwtToken := jwtware.FromContext(c)
		if jwtToken == nil {
			return unauthorized()
		}
		claims, ok := jwtToken.Claims.(*response.JWTClaims)
		if !ok || claims.TokenType != expectedType || claims.Issuer != "novelhub" || claims.Subject != claims.UId || claims.Subject == "" {
			return unauthorized()
		}
		expectedAudience := "novelhub-" + expectedType
		if !slices.Contains([]string(claims.Audience), expectedAudience) {
			return unauthorized()
		}
		if slices.Contains(claims.Roles, constants.RoleTypeBanned) {
			if optional {
				return continueAsGuest(c)
			}
			return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "User account is banned"})
		}

		userID, err := convert.ParseID(claims.UId)
		if err != nil {
			return unauthorized()
		}
		tokenVersion, err := userRepo.GetTokenVersion(c.Context(), userID)
		if err != nil {
			return unauthorized()
		}
		if tokenVersion != claims.TokenVersion {
			return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Token has been invalidated"})
		}

		c.Locals("uid", claims.UId)
		c.Locals("user_claims", claims)
		return c.Next()
	}
}

func jwtError(c fiber.Ctx, err error) error {
	if err != nil && err.Error() == "Missing or malformed JWT" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Missing or malformed JWT"})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Invalid or expired JWT"})
}
