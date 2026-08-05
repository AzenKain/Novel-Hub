package request

import "time"

type CreateUserDto struct {
	Email     string   `json:"email" validate:"required,min=5,max=255,email"`
	Password  string   `json:"password" validate:"required,min=8,max=64"`
	FullName  string   `json:"full_name" validate:"required,min=2,max=100"`
	AvatarUrl string   `json:"avatar_url,omitempty" validate:"omitempty,image_url"`
	RoleIDs   []string `json:"role_ids,omitempty" validate:"omitempty,dive,uuid"`
}

type UpdateProfileDto struct {
	FullName  *string `json:"full_name,omitempty" validate:"omitempty,min=2,max=100"`
	AvatarUrl *string `json:"avatar_url,omitempty" validate:"omitempty,image_url"`
}

type ChangePasswordDto struct {
	OldPassword string `json:"old_password" validate:"omitempty"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=64,nefield=OldPassword"`
}

type ResetPasswordDto struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=64"`
}

type ChangeRoleDto struct {
	Roles []string `json:"role_ids" validate:"required,min=1,dive,uuid"`
}

type SendUserEmailDto struct {
	Subject string `json:"subject" validate:"required,min=1,max=200"`
	Body    string `json:"body" validate:"required,min=1,max=10000"`
}

type SearchUserDto struct {
	PaginationDto
	Sort         string     `json:"sort,omitempty" query:"sort" validate:"omitempty,oneof=id created_at updated_at email is_deleted auth_provider"`
	Search       string     `json:"search,omitempty" query:"search" validate:"omitempty,min=2,max=200"`
	IsDeleted    *bool      `json:"is_deleted,omitempty" query:"is_deleted" validate:"omitempty"`
	RoleIDs      []string   `json:"role_ids,omitempty" query:"role_ids" validate:"omitempty,dive,uuid"`
	AuthProvider string     `json:"auth_provider,omitempty" query:"auth_provider" validate:"omitempty,oneof=LOCAL"`
	CreatedFrom  *time.Time `json:"created_from,omitempty" query:"created_from" validate:"omitempty"`
	CreatedTo    *time.Time `json:"created_to,omitempty" query:"created_to" validate:"omitempty"`
}
