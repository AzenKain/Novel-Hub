package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"novelhub/pkg/apperrors"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/convert"
	"novelhub/pkg/database"
	"novelhub/pkg/jsonx"
)

type RoleService interface {
	GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, error)
	GetAllRole(ctx context.Context) ([]*response.RoleResponse, error)
	GetPermissions(ctx context.Context) ([]*response.PermissionResponse, error)
	CreateRole(ctx context.Context, dto *request.CreateRoleDto) (*response.RoleResponse, error)
	UpdateRole(ctx context.Context, id string, dto *request.UpdateRoleDto) (*response.RoleResponse, error)
	UpdateRolePermissions(ctx context.Context, id string, dto *request.UpdateRolePermissionsDto) (*response.RoleResponse, error)
	DeleteRole(ctx context.Context, id string) error
}

type roleService struct {
	roleRepo        repositories.RoleRepository
	permissionCache PermissionCache
	txManager       database.TxManager
}

func NewRoleService(roleRepo repositories.RoleRepository, permissionCache PermissionCache, txManager database.TxManager) RoleService {
	return &roleService{roleRepo: roleRepo, permissionCache: permissionCache, txManager: txManager}
}

func (r *roleService) GetAllRole(ctx context.Context) ([]*response.RoleResponse, error) {
	roles, err := r.roleRepo.All(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get roles")
	}
	if err := r.attachPermissions(ctx, roles...); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get role permissions")
	}
	return models.RolesEntityToResponse(roles), nil
}

func (r *roleService) GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, error) {
	roleID, parseErr := convert.ParseID(id)
	if parseErr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid role ID")
	}
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if role == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Role not found")
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) GetPermissions(ctx context.Context) ([]*response.PermissionResponse, error) {
	permissions, err := r.roleRepo.ListPermissions(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get permissions")
	}
	return models.PermissionsToResponse(permissions), nil
}

func (r *roleService) CreateRole(ctx context.Context, dto *request.CreateRoleDto) (*response.RoleResponse, error) {
	name := strings.ToUpper(strings.TrimSpace(dto.Name))
	if name == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Role name is required")
	}

	tx, err := r.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := r.roleRepo.WithTx(tx)

	role, err := txRepo.Create(ctx, sqlc.CreateRoleParams{
		Name:        name,
		Description: strings.TrimSpace(dto.Description),
		IsSystem:    0,
		IsAdmin:     0,
		AutoAssign:  convert.BoolToInt64(dto.AutoAssign),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Failed to create role")
	}
	if err := r.replacePermissions(ctx, txRepo, role, dto.Permissions); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit role creation")
	}

	if err := r.reloadPermissionCache(ctx); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) UpdateRole(ctx context.Context, id string, dto *request.UpdateRoleDto) (*response.RoleResponse, error) {
	roleID, ferr := convert.ParseID(id)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid role ID")
	}
	existing, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil || existing == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Role not found")
	}
	if existing.IsAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Admin role cannot be modified")
	}

	tx, err := r.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := r.roleRepo.WithTx(tx)

	var role *models.RoleEntity
	if existing.IsSystem {
		role, err = txRepo.UpdateSystemRoleDescription(ctx, sqlc.UpdateSystemRoleDescriptionParams{
			ID:          roleID,
			Description: strings.TrimSpace(dto.Description),
		})
	} else {
		role, err = txRepo.Update(ctx, sqlc.UpdateRoleParams{
			ID:          roleID,
			Name:        strings.ToUpper(strings.TrimSpace(dto.Name)),
			Description: strings.TrimSpace(dto.Description),
			AutoAssign:  convert.BoolToInt64(dto.AutoAssign),
		})
	}
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Failed to update role")
	}
	if !existing.IsSystem {
		if err := r.replacePermissions(ctx, txRepo, role, dto.Permissions); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit role update")
	}

	if err := r.reloadPermissionCache(ctx); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) UpdateRolePermissions(ctx context.Context, id string, dto *request.UpdateRolePermissionsDto) (*response.RoleResponse, error) {
	roleID, ferr := convert.ParseID(id)
	if ferr != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Invalid role ID")
	}
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Role not found")
	}
	if role.IsAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Admin role permissions cannot be modified")
	}

	tx, err := r.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := r.roleRepo.WithTx(tx)

	if err := r.replacePermissions(ctx, txRepo, role, dto.Permissions); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit permission update")
	}

	if err := r.reloadPermissionCache(ctx); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, role); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get role permissions")
	}
	return role.ToResponse(), nil
}

func (r *roleService) DeleteRole(ctx context.Context, id string) error {
	roleID, ferr := convert.ParseID(id)
	if ferr != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid role ID")
	}
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return apperrors.New(apperrors.ErrNotFound, "Role not found")
	}
	if role.IsSystem || role.IsAdmin {
		return apperrors.New(apperrors.ErrForbidden, "System roles cannot be deleted")
	}
	if err := r.roleRepo.Delete(ctx, roleID); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete role")
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

func (r *roleService) replacePermissions(ctx context.Context, txRepo repositories.RoleRepository, role *models.RoleEntity, dto []request.RolePermissionDto) error {
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
			return apperrors.New(apperrors.ErrBadRequest, "Invalid permission conditions")
		}
		permissions = append(permissions, &models.RolePermissionEntity{
			RoleID:         role.ID,
			PermissionKey:  key,
			Effect:         effect,
			Conditions:     conditions,
			ConditionsJSON: string(data),
		})
	}
	if err := txRepo.ReplaceRolePermissions(ctx, role.ID, permissions); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Failed to update role permissions")
	}
	return nil
}

func (r *roleService) reloadPermissionCache(ctx context.Context) error {
	if r.permissionCache == nil {
		return nil
	}
	if err := r.permissionCache.Reload(ctx); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to reload permission cache")
	}
	return nil
}


