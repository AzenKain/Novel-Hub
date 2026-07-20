package models

import (
	"novelhub/internal/dtos/response"
	"novelhub/pkg/constants"
)

type RoleSimple struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (r *RoleSimple) ToResponse() *response.RoleSimpleResponse {
	if r == nil {
		return nil
	}
	return &response.RoleSimpleResponse{ID: r.ID, Name: r.Name}
}

func RolesToResponse(roles []*RoleSimple) []*response.RoleSimpleResponse {
	out := make([]*response.RoleSimpleResponse, 0, len(roles))
	for _, role := range roles {
		if role == nil {
			continue
		}
		out = append(out, role.ToResponse())
	}
	return out
}

type RoleEntity struct {
	ID          int64                   `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	IsSystem    bool                    `json:"is_system"`
	IsAdmin     bool                    `json:"is_admin"`
	AutoAssign  bool                    `json:"auto_assign"`
	IsDeleted   bool                    `json:"is_deleted"`
	CreatedAt   string                  `json:"created_at"`
	UpdatedAt   string                  `json:"updated_at"`
	Permissions []*RolePermissionEntity `json:"permissions,omitempty"`
}

func (r *RoleEntity) ToResponse() *response.RoleResponse {
	if r == nil {
		return nil
	}
	return &response.RoleResponse{
		ID:        r.ID,
		Name:      r.Name,
		Description: r.Description,
		IsSystem: r.IsSystem,
		IsAdmin: r.IsAdmin,
		AutoAssign: r.AutoAssign,
		IsDeleted: r.IsDeleted,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Permissions: RolePermissionsToResponse(r.Permissions),
	}
}

func (r *RoleEntity) ToRoleSimple() *RoleSimple {
	if r == nil {
		return nil
	}
	return &RoleSimple{ID: r.ID, Name: r.Name}
}

func RolesEntityToResponse(roles []*RoleEntity) []*response.RoleResponse {
	out := make([]*response.RoleResponse, 0, len(roles))
	for _, role := range roles {
		if role == nil {
			continue
		}
		out = append(out, role.ToResponse())
	}
	return out
}

func RolesEntityToRoleConstant(roles []*RoleSimple) []constants.RoleType {
	out := make([]constants.RoleType, 0, len(roles))
	for _, role := range roles {
		if role == nil {
			continue
		}
		parsed, ok := constants.ParseRole(role.Name)
		if ok {
			out = append(out, parsed)
		}
	}
	return out
}

func RolesEntityToRoleIDs(roles []*RoleSimple) []int64 {
	out := make([]int64, 0, len(roles))
	for _, role := range roles {
		if role == nil {
			continue
		}
		out = append(out, role.ID)
	}
	return out
}

type PermissionEntity struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RolePermissionEntity struct {
	ID             int64          `json:"id"`
	RoleID         int64          `json:"role_id"`
	PermissionKey  string         `json:"permission_key"`
	Effect         string         `json:"effect"`
	ConditionsJSON string         `json:"conditions_json"`
	Conditions     map[string]any `json:"conditions,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

func (p *PermissionEntity) ToResponse() *response.PermissionResponse {
	if p == nil {
		return nil
	}
	return &response.PermissionResponse{
		Key:         p.Key,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func PermissionsToResponse(items []*PermissionEntity) []*response.PermissionResponse {
	out := make([]*response.PermissionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

func (p *RolePermissionEntity) ToResponse() *response.RolePermissionResponse {
	if p == nil {
		return nil
	}
	return &response.RolePermissionResponse{
		ID:             p.ID,
		RoleID:         p.RoleID,
		PermissionKey:  p.PermissionKey,
		Effect:         p.Effect,
		ConditionsJSON: p.ConditionsJSON,
		Conditions:     p.Conditions,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func RolePermissionsToResponse(items []*RolePermissionEntity) []*response.RolePermissionResponse {
	out := make([]*response.RolePermissionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}
