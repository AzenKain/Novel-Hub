package response

type UserResponse struct {
	ID           string                `json:"id"`
	Email        string                `json:"email"`
	FullName     string                `json:"full_name"`
	AvatarUrl    string                `json:"avatar_url"`
	AuthProvider string                `json:"auth_provider"`
	Oauth2ID     string                `json:"oauth2_id,omitempty"`
	TokenVersion        int32                 `json:"token_version"`
	IsDeleted           bool                  `json:"is_deleted"`
	IsKidsMode          bool                  `json:"is_kids_mode"`
	MaxAllowedAgeRating string                `json:"max_allowed_age_rating"`
	IsOwner             bool                  `json:"is_owner"`
	CreatedAt           string                `json:"created_at,omitempty"`
	UpdatedAt           string                `json:"updated_at,omitempty"`
	Roles               []*RoleSimpleResponse `json:"roles"`
}

type BulkActionResultResponse struct {
	AffectedCount  int      `json:"affected_count"`
	SkippedCount   int      `json:"skipped_count"`
	SkippedReasons []string `json:"skipped_reasons,omitempty"`
}
