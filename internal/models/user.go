package models

import (
	"bytes"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/jsonx"
)

var jsonNull = []byte("null")

type UserEntity struct {
	ID           int64         `json:"id"`
	Email        string        `json:"email"`
	PasswordHash string        `json:"password_hash"`
	FullName     string        `json:"full_name"`
	AvatarUrl    string        `json:"avatar_url"`
	TokenVersion int32         `json:"token_version"`
	AuthProvider string        `json:"auth_provider"`
	RefreshToken string        `json:"refresh_token"`
	IsDeleted    bool          `json:"is_deleted"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	Roles        []*RoleSimple `json:"roles"`
}

func (u *UserEntity) ParseRoles(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, jsonNull) {
		u.Roles = []*RoleSimple{}
		return nil
	}
	return jsonx.Unmarshal(data, &u.Roles)
}

func (u *UserEntity) ToResponse() *response.UserResponse {
	if u == nil {
		return nil
	}
	return &response.UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		TokenVersion: u.TokenVersion,
		FullName:     u.FullName,
		AvatarUrl:    u.AvatarUrl,
		AuthProvider: u.AuthProvider,
		IsDeleted:    u.IsDeleted,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Roles:        RolesToResponse(u.Roles),
	}
}

func UsersEntityToResponse(users []*UserEntity) []*response.UserResponse {
	out := make([]*response.UserResponse, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		out = append(out, user.ToResponse())
	}
	return out
}
