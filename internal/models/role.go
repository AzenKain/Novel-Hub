package models

import (
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
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
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		IsAdmin:     r.IsAdmin,
		AutoAssign:  r.AutoAssign,
		IsDeleted:   r.IsDeleted,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
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

func (r *RoleEntity) FromSqlc(row sqlc.Role) *RoleEntity {
	r.ID = row.ID
	r.Name = row.Name
	r.Description = row.Description
	r.IsSystem = row.IsSystem != 0
	r.IsAdmin = row.IsAdmin != 0
	r.AutoAssign = row.AutoAssign != 0
	r.IsDeleted = row.IsDeleted != 0
	r.CreatedAt = row.CreatedAt
	r.UpdatedAt = row.UpdatedAt
	return r
}

type RoleEntities []*RoleEntity

func (e *RoleEntities) FromSqlc(rows []sqlc.Role) []*RoleEntity {
	slice := make([]*RoleEntity, len(rows))
	flat := make([]RoleEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (p *PermissionEntity) FromSqlc(row sqlc.Permission) *PermissionEntity {
	p.Key = row.Key
	p.Description = row.Description
	p.CreatedAt = row.CreatedAt
	p.UpdatedAt = row.UpdatedAt
	return p
}

type PermissionEntities []*PermissionEntity

func (e *PermissionEntities) FromSqlc(rows []sqlc.Permission) []*PermissionEntity {
	slice := make([]*PermissionEntity, len(rows))
	flat := make([]PermissionEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (p *RolePermissionEntity) FromSqlc(row sqlc.RolePermission) *RolePermissionEntity {
	p.ID = row.ID
	p.RoleID = row.RoleID
	p.PermissionKey = row.PermissionKey
	p.Effect = row.Effect
	p.ConditionsJSON = row.ConditionsJson
	p.CreatedAt = row.CreatedAt
	p.UpdatedAt = row.UpdatedAt
	
	var conditions map[string]any
	if row.ConditionsJson != "" {
		_ = jsonx.Unmarshal([]byte(row.ConditionsJson), &conditions)
	}
	if conditions == nil {
		conditions = map[string]any{}
	}
	p.Conditions = conditions
	
	return p
}

type RolePermissionEntities []*RolePermissionEntity

func (e *RolePermissionEntities) FromSqlc(rows []sqlc.RolePermission) []*RolePermissionEntity {
	slice := make([]*RolePermissionEntity, len(rows))
	flat := make([]RolePermissionEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}
