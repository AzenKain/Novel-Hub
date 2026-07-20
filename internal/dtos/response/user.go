package response

type UserResponse struct {
	ID           int64                 `json:"id"`
	Email        string                `json:"email"`
	FullName     string                `json:"full_name"`
	AvatarUrl    string                `json:"avatar_url"`
	AuthProvider string                `json:"auth_provider"`
	TokenVersion int32                 `json:"token_version"`
	IsDeleted    bool                  `json:"is_deleted"`
	CreatedAt    string                `json:"created_at,omitempty"`
	UpdatedAt    string                `json:"updated_at,omitempty"`
	Roles        []*RoleSimpleResponse `json:"roles"`
}
