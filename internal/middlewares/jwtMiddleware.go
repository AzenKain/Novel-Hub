package middlewares

import (
	"slices"
	"strconv"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
)

func JwtAccess(userRepo repositories.UserRepository) fiber.Handler {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return func(c fiber.Ctx) error {
			return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{Status: false, Message: "JWT_SECRET is missing"})
		}
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess(userRepo, false),
		Extractor: extractors.Chain(
			extractors.FromAuthHeader("Bearer"),
			extractors.FromCookie("access_token"),
		),
		Claims: &response.JWTClaims{},
	})
}

func GuestClaims() *response.JWTClaims {
	return &response.JWTClaims{
		UId:     "0",
		Roles:   []constants.RoleType{constants.RoleTypeGuest},
		RoleIDs: []int64{constants.SystemRoleIDGuest},
	}
}

func OptionalJwtAccess(userRepo repositories.UserRepository) fiber.Handler {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return func(c fiber.Ctx) error {
			claims := GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}
	}

	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			claims := GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		},
		SuccessHandler: jwtSuccess(userRepo, true),
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
		SigningKey:     jwtware.SigningKey{Key: []byte(jwtRefreshSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess(userRepo, false),
		Extractor: extractors.Chain(
			extractors.FromCookie("refresh_token"),
			extractors.FromAuthHeader("Bearer"),
		),
		Claims: &response.JWTClaims{},
	})
}

func jwtSuccess(userRepo repositories.UserRepository, optional bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		fallbackToGuest := func() error {
			claims := GuestClaims()
			c.Locals("user_claims", claims)
			c.Locals("uid", claims.UId)
			return c.Next()
		}

		unauthorized := func() error {
			if optional {
				return fallbackToGuest()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{Status: false, Message: "Invalid or missing token"})
		}

		jwtToken := jwtware.FromContext(c)
		if jwtToken == nil {
			return unauthorized()
		}
		claims, ok := jwtToken.Claims.(*response.JWTClaims)
		if !ok {
			return unauthorized()
		}
		if slices.Contains(claims.Roles, constants.RoleTypeBanned) {
			if optional {
				return fallbackToGuest()
			}
			return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{Status: false, Message: "User account is banned"})
		}

		userID, err := strconv.ParseInt(claims.UId, 10, 64)
		if err != nil || userID < 1 {
			return unauthorized()
		}
		tokenVersion, err := userRepo.GetTokenVersion(c.Context(), userID)
		if err != nil {
			return unauthorized()
		}
		if tokenVersion != claims.TokenVersion {
			if optional {
				return fallbackToGuest()
			}
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
