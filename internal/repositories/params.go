package repositories

import "time"

type UserSearchFilter struct {
	IsDeleted    *bool
	RoleID       *int64
	AuthProvider *string
	CreatedFrom  *string
	CreatedTo    *string
	SearchText   *string
}

type UserSearchParams struct {
	UserSearchFilter
	Offset int64
	Limit  int64
}

type UpsertUserParams struct {
	Email        string
	PasswordHash *string
	AuthProvider string
	FullName     *string
	AvatarURL    *string
}

type UpdateProfileParams struct {
	ID        int64
	FullName  *string
	AvatarURL *string
}

type BookFileRecordParams struct {
	ID        string
	BookID    string
	Path      string
	Format    string
	SizeBytes int64
	ModTime   time.Time
	Hash      *string
	State     *string
}
