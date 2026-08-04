package response

type UserResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	AvatarUrl    string `json:"avatar_url"`
	AuthProvider string `json:"auth_provider"`
	TokenVersion int32  `json:"token_version"`
	IsDeleted    bool   `json:"is_deleted"`
	// IsOwner marks the root admin recorded at setup. IDs are UUIDs, so the UI
	// cannot infer ownership from the id itself.
	IsOwner   bool                  `json:"is_owner"`
	CreatedAt string                `json:"created_at,omitempty"`
	UpdatedAt string                `json:"updated_at,omitempty"`
	Roles     []*RoleSimpleResponse `json:"roles"`
}
