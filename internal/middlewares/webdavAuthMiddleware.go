package middlewares

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
)

func WebDAVAuth(
	authService services.AuthService,
	settingsService services.SettingsService,
	permissionCache services.PermissionCache,
	userRepo ...repositories.UserRepository,
) fiber.Handler {
	var repo repositories.UserRepository
	if len(userRepo) > 0 {
		repo = userRepo[0]
	}

	return func(c fiber.Ctx) error {
		// OPTIONS discovery must never require authentication (RFC 4918 §10.1)
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		tokenQuery := strings.TrimSpace(c.Query("token"))

		if strings.HasPrefix(authHeader, "Basic ") {
			payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
			if err == nil {
				email, password, found := strings.Cut(string(payload), ":")
				if found {
					// Support percent-encoded username/email from client URLs (e.g. user%40example.com)
					if unescaped, err := url.QueryUnescape(email); err == nil {
						email = unescaped
					}

					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					claims, err := authService.ValidateCredentials(ctx, &request.SignInDto{
						Email:    email,
						Password: password,
					})
					if err == nil && claims != nil {
						c.Locals("user_claims", claims)
						c.Locals("uid", claims.UId)
						return c.Next()
					}
				}
			}

			c.Set("DAV", "1, 2")
			c.Set("WWW-Authenticate", `Basic realm="NovelHub WebDAV"`)
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Invalid credentials"))
		}

		hasToken := strings.HasPrefix(authHeader, "Bearer ") || tokenQuery != "" || c.Cookies("access_token") != ""
		if hasToken && repo != nil {
			if !strings.HasPrefix(authHeader, "Bearer ") && tokenQuery != "" {
				c.Request().Header.Set("Authorization", "Bearer "+tokenQuery)
			}
			return JwtAccess(repo)(c)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		settings, err := settingsService.Public(ctx)
		if err == nil && !settings.GuestLoginRequired && permissionCache != nil {
			if permissionCache.CanRoles(nil, []constants.RoleType{constants.RoleTypeGuest}, constants.PermWebDAVRead, nil) {
				claims := GuestClaims()
				c.Locals("user_claims", claims)
				c.Locals("uid", claims.UId)
				return c.Next()
			}
		}

		c.Set("DAV", "1, 2")
		c.Set("WWW-Authenticate", `Basic realm="NovelHub WebDAV"`)
		return apperrors.HandleError(c, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
	}
}
