package repositories

import (
	"context"
	"database/sql"
	"strconv"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*models.UserEntity, error)
	GetByIDWithoutDeleted(ctx context.Context, id int64) (*models.UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	Search(ctx context.Context, params UserSearchParams) ([]*models.UserEntity, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*models.UserEntity, error)
	Count(ctx context.Context, params UserSearchFilter) (int64, error)
	UpsertUser(ctx context.Context, params UpsertUserParams) (*models.UserEntity, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*models.UserEntity, error)
	UpdateRefreshToken(ctx context.Context, id int64, refreshToken *string) error
	GetTokenVersion(ctx context.Context, id int64) (int32, error)
	UpdateTokenVersion(ctx context.Context, id int64, tokenVersion int64) error
	Delete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	WithTx(tx *sql.Tx) UserRepository
}

type userRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewUserRepository(db sqlc.DBTX, c cache.Cache) UserRepository {
	return &userRepository{q: sqlc.New(db), c: c}
}

func (r *userRepository) WithTx(tx *sql.Tx) UserRepository {
	return &userRepository{q: r.q.WithTx(tx), c: r.c}
}

func nullString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func strPtrToNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func userFilterValue(params UserSearchFilter) sqlc.CountUsersParams {
	var isDeleted interface{}
	if params.IsDeleted != nil {
		if *params.IsDeleted {
			isDeleted = int64(1)
		} else {
			isDeleted = int64(0)
		}
	}
	var roleID interface{}
	if params.RoleID != nil {
		roleID = *params.RoleID
	}
	var authProvider interface{}
	if params.AuthProvider != nil && *params.AuthProvider != "" {
		authProvider = *params.AuthProvider
	}
	var createdFrom interface{}
	if params.CreatedFrom != nil && *params.CreatedFrom != "" {
		createdFrom = *params.CreatedFrom
	}
	var createdTo interface{}
	if params.CreatedTo != nil && *params.CreatedTo != "" {
		createdTo = *params.CreatedTo
	}
	var searchText interface{}
	if params.SearchText != nil && *params.SearchText != "" {
		searchText = *params.SearchText
	}
	return sqlc.CountUsersParams{
		IsDeleted:    isDeleted,
		RoleID:       roleID,
		AuthProvider: authProvider,
		CreatedFrom:  createdFrom,
		CreatedTo:    createdTo,
		SearchText:   searchText,
	}
}

func userSearchValue(params UserSearchParams) sqlc.SearchUserIDsParams {
	filter := userFilterValue(params.UserSearchFilter)
	return sqlc.SearchUserIDsParams{
		IsDeleted:    filter.IsDeleted,
		RoleID:       filter.RoleID,
		AuthProvider: filter.AuthProvider,
		CreatedFrom:  filter.CreatedFrom,
		CreatedTo:    filter.CreatedTo,
		SearchText:   filter.SearchText,
		Offset:       params.Offset,
		Limit:        params.Limit,
	}
}

func mapUser(row sqlc.User) *models.UserEntity {
	return &models.UserEntity{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: nullString(row.PasswordHash),
		FullName:     nullString(row.FullName),
		AvatarUrl:    nullString(row.AvatarUrl),
		AuthProvider: row.AuthProvider,
		TokenVersion: int32(row.TokenVersion), // #nosec G115
		RefreshToken: nullString(row.RefreshToken),
		IsDeleted:    row.IsDeleted != 0,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Roles:        []*models.RoleSimple{},
	}
}

func (r *userRepository) hydrateRoles(ctx context.Context, user *models.UserEntity) error {
	rows, err := r.q.GetUserRoles(ctx, user.ID)
	if err != nil {
		return err
	}
	user.Roles = make([]*models.RoleSimple, 0, len(rows))
	for _, row := range rows {
		user.Roles = append(user.Roles, &models.RoleSimple{ID: row.ID, Name: row.Name})
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*models.UserEntity, error) {
	key := "user:id:" + strconv.FormatInt(id, 10)
	if r.c != nil {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	userPtr := mapUser(row)
	if err := r.hydrateRoles(ctx, userPtr); err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Set(ctx, key, userPtr, constants.NormalCacheDuration)
	}
	return userPtr, nil
}

func (r *userRepository) GetByIDWithoutDeleted(ctx context.Context, id int64) (*models.UserEntity, error) {
	row, err := r.q.GetUserByIDWithoutDeleted(ctx, id)
	if err != nil {
		return nil, err
	}
	user := mapUser(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	key := "user:email:" + email
	if r.c != nil {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	userPtr := mapUser(row)
	if err := r.hydrateRoles(ctx, userPtr); err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Set(ctx, key, userPtr, constants.NormalCacheDuration)
		_ = r.c.Set(ctx, "user:id:"+strconv.FormatInt(userPtr.ID, 10), userPtr, constants.NormalCacheDuration)
	}
	return userPtr, nil
}

func (r *userRepository) UpsertUser(ctx context.Context, params UpsertUserParams) (*models.UserEntity, error) {
	row, err := r.q.UpsertUser(ctx, sqlc.UpsertUserParams{
		Email:        params.Email,
		PasswordHash: strPtrToNullString(params.PasswordHash),
		AuthProvider: params.AuthProvider,
		FullName:     strPtrToNullString(params.FullName),
		AvatarUrl:    strPtrToNullString(params.AvatarURL),
	})
	if err != nil {
		return nil, err
	}
	user := mapUser(row)
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), "user:search*")
		_ = r.c.DelByPattern(context.Background(), "user:count*")
	}
	return user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*models.UserEntity, error) {
	row, err := r.q.UpdateProfile(ctx, sqlc.UpdateProfileParams{
		ID:        params.ID,
		FullName:  strPtrToNullString(params.FullName),
		AvatarUrl: strPtrToNullString(params.AvatarURL),
	})
	if err != nil {
		return nil, err
	}
	user := mapUser(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:email:"+user.Email, "user:id:"+strconv.FormatInt(user.ID, 10))
	}
	return user, nil
}

func (r *userRepository) Search(ctx context.Context, params UserSearchParams) ([]*models.UserEntity, error) {
	sqlcParams := userSearchValue(params)
	key := cache.QueryKey("user:search", sqlcParams)
	if r.c != nil {
		var ids []int64
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetByIDs(ctx, ids)
		}
	}

	dbIds, err := r.q.SearchUserIDs(ctx, sqlcParams)
	if err != nil {
		return nil, err
	}

	if r.c != nil {
		_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
	}
	return r.GetByIDs(ctx, dbIds)
}

func (r *userRepository) GetByIDs(ctx context.Context, ids []int64) ([]*models.UserEntity, error) {
	if len(ids) == 0 {
		return []*models.UserEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = "user:id:" + strconv.FormatInt(id, 10)
	}

	users := make([]*models.UserEntity, 0, len(ids))
	missingIds := []int64{}
	missingKeys := []string{}

	if r.c != nil {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var user models.UserEntity
				if err := jsonx.Unmarshal(bytes, &user); err == nil {
					users = append(users, &user)
					continue
				}
			}
			missingIds = append(missingIds, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIds = ids
		missingKeys = keys
	}

	if len(missingIds) > 0 {
		rows, err := r.q.GetUsersByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[int64]*models.UserEntity)
		for _, row := range rows {
			u := mapUser(row)
			_ = r.hydrateRoles(ctx, u)
			missingMap[u.ID] = u
			users = append(users, u)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if u, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = u
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	userMap := make(map[int64]*models.UserEntity)
	for _, u := range users {
		userMap[u.ID] = u
	}
	ordered := make([]*models.UserEntity, 0, len(ids))
	for _, id := range ids {
		if u, ok := userMap[id]; ok {
			ordered = append(ordered, u)
		}
	}

	return ordered, nil
}

func (r *userRepository) Count(ctx context.Context, params UserSearchFilter) (int64, error) {
	sqlcParams := userFilterValue(params)
	key := cache.QueryKey("user:count", sqlcParams)
	if r.c != nil {
		var count int64
		if err := r.c.Get(ctx, key, &count); err == nil {
			return count, nil
		}
	}
	count, err := r.q.CountUsers(ctx, sqlcParams)
	if err != nil {
		return 0, err
	}
	if r.c != nil {
		_ = r.c.Set(ctx, key, count, constants.ListCacheDuration)
	}
	return count, nil
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.DeleteUser(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:id:"+strconv.FormatInt(user.ID, 10), "user:email:"+user.Email, "user:token:"+strconv.FormatInt(user.ID, 10))
		_ = r.c.DelByPattern(context.Background(), "user:search*")
		_ = r.c.DelByPattern(context.Background(), "user:count*")
	}
	return nil
}

func (r *userRepository) Restore(ctx context.Context, id int64) error {
	if err := r.q.RestoreUser(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:id:"+strconv.FormatInt(id, 10), "user:token:"+strconv.FormatInt(id, 10))
		_ = r.c.DelByPattern(context.Background(), "user:search*")
		_ = r.c.DelByPattern(context.Background(), "user:count*")
	}
	return nil
}

func (r *userRepository) GetTokenVersion(ctx context.Context, id int64) (int32, error) {
	key := "user:token:" + strconv.FormatInt(id, 10)
	if r.c != nil {
		var version int32
		if err := r.c.Get(ctx, key, &version); err == nil {
			return version, nil
		}
	}
	raw, err := r.q.GetUserTokenVersion(ctx, id)
	if err != nil {
		return 0, err
	}
	version := int32(raw) // #nosec G115
	if r.c != nil {
		_ = r.c.Set(ctx, key, version, constants.NormalCacheDuration)
	}
	return version, nil
}

func (r *userRepository) UpdateTokenVersion(ctx context.Context, id int64, tokenVersion int64) error {
	user, _ := r.GetByID(ctx, id)
	if err := r.q.UpdateUserTokenVersion(ctx, sqlc.UpdateUserTokenVersionParams{ID: id, TokenVersion: tokenVersion}); err != nil {
		return err
	}
	if r.c != nil {
		keys := []string{"user:token:" + strconv.FormatInt(id, 10), "user:id:" + strconv.FormatInt(id, 10)}
		if user != nil && user.Email != "" {
			keys = append(keys, "user:email:"+user.Email)
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: sql.NullString{String: passwordHash, Valid: passwordHash != ""},
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:email:"+user.Email, "user:id:"+strconv.FormatInt(user.ID, 10), "user:token:"+strconv.FormatInt(user.ID, 10))
	}
	return nil
}

func (r *userRepository) UpdateRefreshToken(ctx context.Context, id int64, refreshToken *string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.UpdateUserRefreshToken(ctx, sqlc.UpdateUserRefreshTokenParams{
		ID:           id,
		RefreshToken: strPtrToNullString(refreshToken),
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:email:"+user.Email, "user:id:"+strconv.FormatInt(user.ID, 10), "user:token:"+strconv.FormatInt(user.ID, 10))
	}
	return nil
}
