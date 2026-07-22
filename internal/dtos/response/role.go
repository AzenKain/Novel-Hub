package response

type RoleSimpleResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"is_admin"`
	IsBanned bool   `json:"is_banned"`
}

type RoleResponse struct {
	ID          int64                     `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	IsSystem    bool                      `json:"is_system"`
	IsAdmin     bool                      `json:"is_admin"`
	IsBanned    bool                      `json:"is_banned"`
	AutoAssign  bool                      `json:"auto_assign"`
	Position    int64                     `json:"position"`
	IsDeleted   bool                      `json:"is_deleted"`
	CreatedAt   string                    `json:"created_at,omitempty"`
	UpdatedAt   string                    `json:"updated_at,omitempty"`
	Permissions []*RolePermissionResponse `json:"permissions,omitempty"`
}

type PermissionResponse struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type RolePermissionResponse struct {
	ID             int64          `json:"id"`
	RoleID         int64          `json:"role_id"`
	PermissionKey  string         `json:"permission_key"`
	Effect         string         `json:"effect"`
	ConditionsJSON string         `json:"conditions_json"`
	Conditions     map[string]any `json:"conditions,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
	UpdatedAt      string         `json:"updated_at,omitempty"`
}
