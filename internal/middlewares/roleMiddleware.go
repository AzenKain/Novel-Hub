package middlewares

import (
	"context"
	"slices"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/internal/services"
	"novelhub/pkg/constants"
)

func getRoles(c fiber.Ctx) ([]constants.RoleType, error) {
	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return nil, fiber.ErrUnauthorized
	}
	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return nil, fiber.ErrUnauthorized
	}
	return claims.Roles, nil
}

func RequireAnyRole(required ...constants.RoleType) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, err := getRoles(c)
		if err != nil {
			return err
		}
		for _, role := range userRoles {
			if slices.Contains(required, role) {
				return c.Next()
			}
		}
		return fiber.ErrForbidden
	}
}

type PermissionAttrResolver func(c fiber.Ctx) (map[string]any, error)

func RequireAnyPermission(permissionCache services.PermissionCache, permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if permissionCache == nil {
			return fiber.ErrForbidden
		}
		claims, ok := c.Locals("user_claims").(*response.JWTClaims)
		if !ok || claims == nil {
			return fiber.ErrUnauthorized
		}
		ctx := services.WithPermissionContext(context.Background(), services.PermissionContext{RoleIDs: claims.RoleIDs, Roles: claims.Roles})
		for _, permission := range permissions {
			if permissionCache.Can(ctx, claims.UId, permission, nil) {
				return c.Next()
			}
		}
		return fiber.ErrForbidden
	}
}

func RequirePermission(permissionCache services.PermissionCache, permission string, resolvers ...PermissionAttrResolver) fiber.Handler {
	return func(c fiber.Ctx) error {
		if permissionCache == nil {
			return fiber.ErrForbidden
		}
		claimsVal := c.Locals("user_claims")
		if claimsVal == nil {
			return fiber.ErrUnauthorized
		}
		claims, ok := claimsVal.(*response.JWTClaims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		attrs := map[string]any{}
		for _, resolver := range resolvers {
			if resolver == nil {
				continue
			}
			resolved, err := resolver(c)
			if err != nil {
				return err
			}
			for key, value := range resolved {
				attrs[key] = value
			}
		}

		ctx := services.WithPermissionContext(context.Background(), services.PermissionContext{
			RoleIDs: claims.RoleIDs,
			Roles:   claims.Roles,
		})
		if !permissionCache.Can(ctx, claims.UId, permission, attrs) {
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}

func LibraryIDParam(param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		value := c.Params(param)
		if value == "" {
			return nil, fiber.ErrBadRequest
		}
		return map[string]any{"library_id": value}, nil
	}
}

func BookLibraryAttr(bookRepo repositories.BookDBRepository, param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		bookID := c.Params(param)
		if bookID == "" {
			return nil, fiber.ErrBadRequest
		}
		book, err := bookRepo.GetBook(c.Context(), bookID)
		if err != nil {
			return nil, fiber.ErrNotFound
		}
		return map[string]any{"library_id": book.LibraryID}, nil
	}
}

func BookFileLibraryAttr(bookRepo repositories.BookDBRepository, param string) PermissionAttrResolver {
	return func(c fiber.Ctx) (map[string]any, error) {
		file, err := bookRepo.GetBookFileById(c.Context(), c.Params(param))
		if err != nil || file == nil {
			return nil, fiber.ErrNotFound
		}
		book, err := bookRepo.GetBook(c.Context(), file.BookID)
		if err != nil || book == nil {
			return nil, fiber.ErrNotFound
		}
		return map[string]any{"library_id": book.LibraryID}, nil
	}
}
