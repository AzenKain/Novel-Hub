package request

type RolePermissionDto struct {
	PermissionKey string         `json:"permission_key" validate:"required,min=3,max=100"`
	Effect        string         `json:"effect" validate:"omitempty,oneof=allow deny"`
	Conditions    map[string]any `json:"conditions,omitempty" validate:"omitempty"`
}

type CreateRoleDto struct {
	Name        string              `json:"name" validate:"required,min=2,max=80"`
	Description string              `json:"description,omitempty" validate:"omitempty,max=500"`
	AutoAssign  bool                `json:"auto_assign"`
	Permissions []RolePermissionDto `json:"permissions,omitempty" validate:"omitempty,dive"`
}

type UpdateRoleDto struct {
	Name        string              `json:"name" validate:"required,min=2,max=80"`
	Description string              `json:"description,omitempty" validate:"omitempty,max=500"`
	AutoAssign  bool                `json:"auto_assign"`
	Permissions []RolePermissionDto `json:"permissions,omitempty" validate:"omitempty,dive"`
}

type UpdateRolePermissionsDto struct {
	Permissions []RolePermissionDto `json:"permissions" validate:"required,dive"`
}

type ReorderRolesDto struct {
	RoleIDs []string `json:"role_ids" validate:"required,min=1,dive,uuid"`
}
