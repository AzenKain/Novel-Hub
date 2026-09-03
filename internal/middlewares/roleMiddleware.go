package middlewares

import (
	"context"
	"slices"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

var (
	errUnauthorized = apperrors.New(apperrors.ErrUnauthorized, "Unauthorized")
	errForbidden    = apperrors.New(apperrors.ErrForbidden, "Permission denied")
)

func getRoles(c fiber.Ctx) ([]constants.RoleType, error) {
	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return nil, errUnauthorized
	}
	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return nil, errUnauthorized
	}
	return claims.Roles, nil
}

func RequireAnyRole(required ...constants.RoleType) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, err := getRoles(c)
		if err != nil {
			return apperrors.HandleError(c, err)
		}
		for _, role := range userRoles {
			if slices.Contains(required, role) {
				return c.Next()
			}
		}
		return apperrors.HandleError(c, errForbidden)
	}
}

type PermissionAttrResolver func(c fiber.Ctx) (map[string]any, error)

func RequireAniListTrackingEnabled(settings services.SettingsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		if settings == nil {
			return apperrors.HandleError(c, errForbidden)
		}
		current, err := settings.Public(context.Background())
		if err != nil || current == nil || !current.EnableAniListTracking {
			return apperrors.HandleError(c, apperrors.New(apperrors.ErrForbidden, "AniList tracking is disabled"))
		}
		return c.Next()
	}
}

func RequireAnyPermission(permissionCache services.PermissionCache, permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if permissionCache == nil {
			return apperrors.HandleError(c, errForbidden)
		}
		claims, ok := c.Locals("user_claims").(*response.JWTClaims)
		if !ok || claims == nil {
			return apperrors.HandleError(c, errUnauthorized)
		}
		ctx := services.WithPermissionContext(context.Background(), claims)
		for _, permission := range permissions {
			if permissionCache.Can(ctx, claims.UId, permission, nil) {
				return c.Next()
			}
		}
		return apperrors.HandleError(c, errForbidden)
	}
}

func RequireAnyPermissionAttr(permissionCache services.PermissionCache, permissions []string, resolvers ...PermissionAttrResolver) fiber.Handler {
	return func(c fiber.Ctx) error {
		if permissionCache == nil {
			return apperrors.HandleError(c, errForbidden)
		}
		claimsVal := c.Locals("user_claims")
		if claimsVal == nil {
			return apperrors.HandleError(c, errUnauthorized)
		}
		claims, ok := claimsVal.(*response.JWTClaims)
		if !ok || claims == nil {
			return apperrors.HandleError(c, errUnauthorized)
		}

		attrs := map[string]any{}
		for _, resolver := range resolvers {
			if resolver == nil {
				continue
			}
			resolved, err := resolver(c)
			if err != nil {
				return apperrors.HandleError(c, err)
			}
			for key, value := range resolved {
				attrs[key] = value
			}
		}

		ctx := services.WithPermissionContext(context.Background(), claims)
		for _, permission := range permissions {
			if permissionCache.Can(ctx, claims.UId, permission, attrs) {
				return c.Next()
			}
		}
		return apperrors.HandleError(c, errForbidden)
	}
}

func RequirePermission(permissionCache services.PermissionCache, permission string, resolvers ...PermissionAttrResolver) fiber.Handler {
	return func(c fiber.Ctx) error {
		if permissionCache == nil {
			return apperrors.HandleError(c, errForbidden)
		}
		claimsVal := c.Locals("user_claims")
		if claimsVal == nil {
			return apperrors.HandleError(c, errUnauthorized)
		}
		claims, ok := claimsVal.(*response.JWTClaims)
		if !ok {
			return apperrors.HandleError(c, errUnauthorized)
		}

		attrs := map[string]any{}
		for _, resolver := range resolvers {
			if resolver == nil {
				continue
			}
			resolved, err := resolver(c)
			if err != nil {
				return apperrors.HandleError(c, err)
			}
			for key, value := range resolved {
				attrs[key] = value
			}
		}

		ctx := services.WithPermissionContext(context.Background(), claims)
		if !permissionCache.Can(ctx, claims.UId, permission, attrs) {
			return apperrors.HandleError(c, errForbidden)
		}
		return c.Next()
	}
}

func LibraryIDParam(param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		value := c.Params(param)
		if value == "" {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Library ID is required")
		}
		return map[string]any{"library_id": value}, nil
	}
}

func BookLibraryAttr(bookRepo repositories.BookDBRepository, param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		bookID := c.Params(param)
		if bookID == "" {
			return nil, apperrors.New(apperrors.ErrBadRequest, "Book ID is required")
		}
		book, err := bookRepo.GetBook(c.Context(), bookID)
		if err != nil || book == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return map[string]any{"library_id": book.LibraryID}, nil
	}
}

func BookFileLibraryAttr(bookRepo repositories.BookDBRepository, param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		file, err := bookRepo.GetBookFileById(c.Context(), c.Params(param))
		if err != nil || file == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book file not found")
		}
		book, err := bookRepo.GetBook(c.Context(), file.BookID)
		if err != nil || book == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
		}
		return map[string]any{"library_id": book.LibraryID}, nil
	}
}

func PodcastLibraryAttr(podcastRepo repositories.PodcastRepository, param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		podcast, err := podcastRepo.GetPodcast(c.Context(), c.Params(param))
		if err != nil || podcast == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, "Podcast not found")
		}
		return map[string]any{"library_id": podcast.LibraryID}, nil
	}
}

func LibraryIDBody() PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		body := c.Body()
		if len(body) == 0 {
			return nil, nil
		}
		var payload struct {
			LibraryID string `json:"library_id"`
		}
		if err := jsonx.Unmarshal(body, &payload); err == nil && payload.LibraryID != "" {
			return map[string]any{"library_id": payload.LibraryID}, nil
		}
		return nil, nil
	}
}
