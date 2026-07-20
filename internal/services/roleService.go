package services

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type RoleService interface {
	GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, *fiber.Error)
	GetAllRole(ctx context.Context) ([]*response.RoleResponse, *fiber.Error)
	GetPermissions(ctx context.Context) ([]*response.PermissionResponse, *fiber.Error)
	CreateRole(ctx context.Context, dto *request.CreateRoleDto) (*response.RoleResponse, *fiber.Error)
	UpdateRole(ctx context.Context, id string, dto *request.UpdateRoleDto) (*response.RoleResponse, *fiber.Error)
	UpdateRolePermissions(ctx context.Context, id string, dto *request.UpdateRolePermissionsDto) (*response.RoleResponse, *fiber.Error)
	DeleteRole(ctx context.Context, id string) *fiber.Error
}

type roleService struct {
	roleRepo        repositories.RoleRepository
	permissionCache PermissionCache
}

func NewRoleService(roleRepo repositories.RoleRepository, permissionCache PermissionCache) RoleService {
	return &roleService{roleRepo: roleRepo, permissionCache: permissionCache}
}

func (r *roleService) GetAllRole(ctx context.Context) ([]*response.RoleResponse, *fiber.Error) {
	roles, err := r.roleRepo.All(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get roles")
	}
	if err := r.attachPermissions(ctx, roles...); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get role permissions")
	}
	return models.RolesEntityToResponse(roles), nil
}

func (r *roleService) GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, *fiber.Error) {
	roleID, parseErr := strconv.ParseInt(id, 10, 64)
	if parseErr != nil || roleID < 1 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid role ID")
	}
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}
	if role == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Role not found")
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) GetPermissions(ctx context.Context) ([]*response.PermissionResponse, *fiber.Error) {
	permissions, err := r.roleRepo.ListPermissions(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get permissions")
	}
	return models.PermissionsToResponse(permissions), nil
}

func (r *roleService) CreateRole(ctx context.Context, dto *request.CreateRoleDto) (*response.RoleResponse, *fiber.Error) {
	name := normalizeRoleName(dto.Name)
	if name == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Role name is required")
	}
	role, err := r.roleRepo.Create(ctx, sqlc.CreateRoleParams{
		Name:        name,
		Description: strings.TrimSpace(dto.Description),
		IsSystem:    0,
		IsAdmin:     0,
		AutoAssign:  convert.BoolToInt64(dto.AutoAssign),
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Failed to create role")
	}
	if err := r.replacePermissions(ctx, role, dto.Permissions); err != nil {
		return nil, err
	}
	if err := r.reloadPermissionCache(ctx); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) UpdateRole(ctx context.Context, id string, dto *request.UpdateRoleDto) (*response.RoleResponse, *fiber.Error) {
	roleID, ferr := parseRoleID(id)
	if ferr != nil {
		return nil, ferr
	}
	existing, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil || existing == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Role not found")
	}
	if existing.IsAdmin {
		return nil, fiber.NewError(fiber.StatusForbidden, "Admin role cannot be modified")
	}

	var role *models.RoleEntity
	if existing.IsSystem {
		role, err = r.roleRepo.UpdateSystemRoleDescription(ctx, sqlc.UpdateSystemRoleDescriptionParams{
			ID:          roleID,
			Description: strings.TrimSpace(dto.Description),
		})
	} else {
		role, err = r.roleRepo.Update(ctx, sqlc.UpdateRoleParams{
			ID:          roleID,
			Name:        normalizeRoleName(dto.Name),
			Description: strings.TrimSpace(dto.Description),
			AutoAssign:  convert.BoolToInt64(dto.AutoAssign),
		})
	}
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Failed to update role")
	}
	if !existing.IsSystem {
		if err := r.replacePermissions(ctx, role, dto.Permissions); err != nil {
			return nil, err
		}
	}
	if err := r.reloadPermissionCache(ctx); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) UpdateRolePermissions(ctx context.Context, id string, dto *request.UpdateRolePermissionsDto) (*response.RoleResponse, *fiber.Error) {
	roleID, ferr := parseRoleID(id)
	if ferr != nil {
		return nil, ferr
	}
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Role not found")
	}
	if role.IsAdmin {
		return nil, fiber.NewError(fiber.StatusForbidden, "Admin role permissions cannot be modified")
	}
	if err := r.replacePermissions(ctx, role, dto.Permissions); err != nil {
		return nil, err
	}
	if err := r.reloadPermissionCache(ctx); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) DeleteRole(ctx context.Context, id string) *fiber.Error {
	roleID, ferr := parseRoleID(id)
	if ferr != nil {
		return ferr
	}
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return fiber.NewError(fiber.StatusNotFound, "Role not found")
	}
	if role.IsSystem || role.IsAdmin {
		return fiber.NewError(fiber.StatusForbidden, "System roles cannot be deleted")
	}
	if err := r.roleRepo.Delete(ctx, roleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete role")
	}
	return r.reloadPermissionCache(ctx)
}

func (r *roleService) attachPermissions(ctx context.Context, roles ...*models.RoleEntity) error {
	for _, role := range roles {
		if role == nil {
			continue
		}
		permissions, err := r.roleRepo.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return err
		}
		role.Permissions = permissions
	}
	return nil
}

func (r *roleService) replacePermissions(ctx context.Context, role *models.RoleEntity, dto []request.RolePermissionDto) *fiber.Error {
	permissions := make([]*models.RolePermissionEntity, 0, len(dto))
	for _, item := range dto {
		key := strings.TrimSpace(item.PermissionKey)
		if key == "" {
			continue
		}
		effect := item.Effect
		if effect == "" {
			effect = "allow"
		}
		conditions := item.Conditions
		if conditions == nil {
			conditions = map[string]any{}
		}
		data, err := jsonx.Marshal(conditions)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid permission conditions")
		}
		permissions = append(permissions, &models.RolePermissionEntity{
			RoleID:         role.ID,
			PermissionKey:  key,
			Effect:         effect,
			Conditions:     conditions,
			ConditionsJSON: string(data),
		})
	}
	if err := r.roleRepo.ReplaceRolePermissions(ctx, role.ID, permissions); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Failed to update role permissions")
	}
	return nil
}

func (r *roleService) reloadPermissionCache(ctx context.Context) *fiber.Error {
	if r.permissionCache == nil {
		return nil
	}
	if err := r.permissionCache.Reload(ctx); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to reload permission cache")
	}
	return nil
}

func parseRoleID(id string) (int64, *fiber.Error) {
	roleID, parseErr := strconv.ParseInt(id, 10, 64)
	if parseErr != nil || roleID < 1 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "Invalid role ID")
	}
	return roleID, nil
}

func normalizeRoleName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
