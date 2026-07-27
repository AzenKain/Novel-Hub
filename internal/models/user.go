package models

import (
	"bytes"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

var jsonNull = []byte("null")

type UserEntity struct {
	ID           string        `json:"id"`
	Email        string        `json:"email"`
	PasswordHash string        `json:"-"`
	FullName     string        `json:"full_name"`
	AvatarUrl    string        `json:"avatar_url"`
	TokenVersion int32         `json:"token_version"`
	AuthProvider string        `json:"auth_provider"`
	RefreshToken string        `json:"-"`
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
func (u *UserEntity) IsAdmin() bool {
	if u == nil {
		return false
	}
	for _, r := range u.Roles {
		if r != nil && (r.IsAdmin || r.Name == "ADMIN") {
			return true
		}
	}
	return false
}

func (u *UserEntity) FromSqlc(row sqlc.User) *UserEntity {
	u.ID = row.ID
	u.Email = row.Email
	u.PasswordHash = convert.NullStringToString(row.PasswordHash)
	u.FullName = convert.NullStringToString(row.FullName)
	u.AvatarUrl = convert.NullStringToString(row.AvatarUrl)
	u.AuthProvider = row.AuthProvider
	u.TokenVersion = int32(row.TokenVersion) // #nosec G115
	u.RefreshToken = convert.NullStringToString(row.RefreshToken)
	u.IsDeleted = row.IsDeleted != 0
	u.CreatedAt = row.CreatedAt
	u.UpdatedAt = row.UpdatedAt
	u.Roles = []*RoleSimple{}
	return u
}

type UserEntities []*UserEntity

func (e *UserEntities) FromSqlc(rows []sqlc.User) []*UserEntity {
	slice := make([]*UserEntity, len(rows))
	flat := make([]UserEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}
