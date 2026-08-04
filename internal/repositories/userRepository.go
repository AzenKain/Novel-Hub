package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/config"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.UserEntity, error)
	GetByIDWithoutDeleted(ctx context.Context, id string) (*models.UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	GetAuthByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	GetAuthByID(ctx context.Context, id string) (*models.UserEntity, error)
	Search(ctx context.Context, params sqlc.SearchUserIDsParams) ([]*models.UserEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.UserEntity, error)
	Count(ctx context.Context, params sqlc.CountUsersParams) (int64, error)
	UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error)
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
	UpdateProfile(ctx context.Context, params sqlc.UpdateProfileParams) (*models.UserEntity, error)
	UpdateRefreshToken(ctx context.Context, id string, refreshToken *string) error
	RotateRefreshToken(ctx context.Context, id string, currentRefreshToken, newRefreshToken string) (bool, error)
	GetTokenVersion(ctx context.Context, id string) (int32, error)
	UpdateTokenVersion(ctx context.Context, id string, tokenVersion int64) error
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	InvalidateUserCache(ctx context.Context, id, email string)
	WithTx(tx *sql.Tx) UserRepository
}

type userRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewUserRepository(db sqlc.DBTX, c cache.Cache) UserRepository {
	return &userRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *userRepository) WithTx(tx *sql.Tx) UserRepository {
	return &userRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  r.sfg,
	}
}

func (r *userRepository) hydrateRoles(ctx context.Context, user *models.UserEntity) error {
	key := cache.BuildKey("user", "roles", user.ID)
	if r.c != nil && !r.inTx {
		var roles []*models.RoleSimple
		if err := r.c.Get(ctx, key, &roles); err == nil {
			user.Roles = roles
			return nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.GetUserRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		userRoles := make([]*models.RoleSimple, 0, len(rows))
		for _, row := range rows {
			userRoles = append(userRoles, &models.RoleSimple{ID: row.ID, Name: row.Name})
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, userRoles, constants.NormalCacheDuration)
		}
		return userRoles, nil
	})
	if err != nil {
		return err
	}
	user.Roles = v.([]*models.RoleSimple)
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*models.UserEntity, error) {
	key := cache.BuildKey("user", "id", id)
	if r.c != nil && !r.inTx {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetUserByID(ctx, id)
		if err != nil {
			return nil, err
		}
		userPtr := (&models.UserEntity{}).FromSqlc(row)
		if err := r.hydrateRoles(ctx, userPtr); err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, userPtr, constants.NormalCacheDuration)
		}
		return userPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserEntity), nil
}

func (r *userRepository) GetByIDWithoutDeleted(ctx context.Context, id string) (*models.UserEntity, error) {
	row, err := r.q.GetUserByIDWithoutDeleted(ctx, id)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	key := cache.BuildKey("user", "email", email)
	if r.c != nil && !r.inTx {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		userPtr := (&models.UserEntity{}).FromSqlc(row)
		if err := r.hydrateRoles(ctx, userPtr); err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, userPtr, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("user", "id", userPtr.ID), userPtr, constants.NormalCacheDuration)
		}
		return userPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserEntity), nil
}

// GetAuthByEmail and GetAuthByID deliberately bypass the cache used by GetByEmail
// and GetByID. The RAM cache serialises entities as JSON, and UserEntity tags
// PasswordHash and RefreshToken as `json:"-"` so credentials never leak into a
// response — which means a cached entity comes back with both fields empty. Any
// caller that compares a password or a refresh token must read the row itself.
func (r *userRepository) GetAuthByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetAuthByID(ctx context.Context, id string) (*models.UserEntity, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error) {
	row, err := r.q.UpsertUser(ctx, params)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if r.c != nil {
		// ON CONFLICT(email) updates full_name/avatar_url of an existing row, so the
		// entity keys are stale too — not just the list keys. row.ID is the surviving
		// row's id, which is not params.ID on the conflict path.
		_ = r.c.Del(
			ctx,
			cache.BuildKey("user", "id", user.ID),
			cache.BuildKey("user", "email", user.Email),
			cache.BuildKey("user", "roles", user.ID),
		)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	}
	return user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, params sqlc.UpdateProfileParams) (*models.UserEntity, error) {
	row, err := r.q.UpdateProfile(ctx, params)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "email", user.Email), cache.BuildKey("user", "id", user.ID))
		// SearchUserIDs and CountUsers both match search_text against full_name.
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	}
	return user, nil
}

func ftsMatchQuery(searchText any) (string, bool) {
	term, ok := searchText.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(term)
	if len([]rune(trimmed)) < 3 {
		return "", false
	}
	return `"` + strings.ReplaceAll(trimmed, `"`, `""`) + `"`, true
}

func (r *userRepository) Search(ctx context.Context, params sqlc.SearchUserIDsParams) ([]*models.UserEntity, error) {
	key := cache.QueryKey("user:search", params)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		dbIds, err := r.searchIDs(ctx, params)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
		}
		return dbIds, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByIDs(ctx, v.([]string))
}

const ftsListCutoff = 2000

func (r *userRepository) ftsSelectivity(ctx context.Context, match string) (bool, error) {
	matches, err := r.q.ProbeUserSearchMatches(ctx, sqlc.ProbeUserSearchMatchesParams{
		MatchQuery: match,
		Cap:        ftsListCutoff,
	})
	if err != nil {
		return false, err
	}
	return matches < ftsListCutoff, nil
}

func (r *userRepository) searchIDs(ctx context.Context, params sqlc.SearchUserIDsParams) ([]string, error) {
	match, ok := ftsMatchQuery(params.SearchText)
	if !ok {
		return r.q.SearchUserIDs(ctx, params)
	}
	if selective, err := r.ftsSelectivity(ctx, match); err != nil || !selective {
		return r.q.SearchUserIDs(ctx, params)
	}
	return r.q.SearchUserIDsFTS(ctx, sqlc.SearchUserIDsFTSParams{
		MatchQuery:      match,
		IsDeleted:       params.IsDeleted,
		RoleID:          params.RoleID,
		AuthProvider:    params.AuthProvider,
		CreatedFrom:     params.CreatedFrom,
		CreatedTo:       params.CreatedTo,
		CursorCreatedAt: params.CursorCreatedAt,
		CursorID:        params.CursorID,
		Limit:           params.Limit,
	})
}

func (r *userRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.UserEntity, error) {
	if len(ids) == 0 {
		return []*models.UserEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("user", "id", id)
	}

	users := make([]*models.UserEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
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
		sort.Strings(missingIds)
		sfgKey := "users:ids:" + strings.Join(missingIds, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := queryInChunks(missingIds, func(chunk []string) ([]sqlc.User, error) {
				return r.q.GetUsersByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			missingMap := make(map[string]*models.UserEntity)
			for _, row := range rows {
				u := (&models.UserEntity{}).FromSqlc(row)
				_ = r.hydrateRoles(ctx, u)
				missingMap[u.ID] = u
			}
			return missingMap, nil
		})
		if err != nil {
			return nil, err
		}
		missingMap := v.(map[string]*models.UserEntity)

		for _, u := range missingMap {
			users = append(users, u)
		}

		if r.c != nil && !r.inTx {
			missingToCache := make(map[string]any)
			for _, missingId := range missingIds {
				// GetUsersByIDs has no is_deleted filter (admin listings need soft-deleted
				// rows), but user:id:<id> is the key GetByID reads — caching a deleted row
				// there resurrects the user for a full TTL. Return it, don't cache it.
				if u, ok := missingMap[missingId]; ok && !u.IsDeleted {
					missingToCache[cache.BuildKey("user", "id", missingId)] = u
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	userMap := make(map[string]*models.UserEntity)
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

func (r *userRepository) Count(ctx context.Context, params sqlc.CountUsersParams) (int64, error) {
	key := cache.QueryKey("user:count", params)
	if r.c != nil && !r.inTx {
		var count int64
		if err := r.c.Get(ctx, key, &count); err == nil {
			return count, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		count, err := r.countUsers(ctx, params)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, count, constants.ListCacheDuration)
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (r *userRepository) countUsers(ctx context.Context, params sqlc.CountUsersParams) (int64, error) {
	match, ok := ftsMatchQuery(params.SearchText)
	if !ok {
		return r.q.CountUsers(ctx, params)
	}
	if selective, err := r.ftsSelectivity(ctx, match); err != nil || !selective {
		return r.q.CountUsers(ctx, params)
	}
	return r.q.CountUsersFTS(ctx, sqlc.CountUsersFTSParams{
		MatchQuery:   match,
		IsDeleted:    params.IsDeleted,
		RoleID:       params.RoleID,
		AuthProvider: params.AuthProvider,
		CreatedFrom:  params.CreatedFrom,
		CreatedTo:    params.CreatedTo,
	})
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.DeleteUser(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "id", user.ID), cache.BuildKey("user", "email", user.Email), cache.BuildKey("user", "token", user.ID), constants.CacheKeyRoleCountActiveAdminUsers, constants.CacheKeySettingsAdminCount)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	}
	return nil
}

func (r *userRepository) Restore(ctx context.Context, id string) error {
	if err := r.q.RestoreUser(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "id", id), cache.BuildKey("user", "token", id), constants.CacheKeyRoleCountActiveAdminUsers, constants.CacheKeySettingsAdminCount)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	}
	return nil
}

func (r *userRepository) RotateRefreshToken(ctx context.Context, id string, currentRefreshToken, newRefreshToken string) (bool, error) {
	updated, err := r.q.RotateUserRefreshToken(ctx, sqlc.RotateUserRefreshTokenParams{
		ID:                  id,
		CurrentRefreshToken: sql.NullString{String: currentRefreshToken, Valid: true},
		NewRefreshToken:     sql.NullString{String: newRefreshToken, Valid: true},
	})
	if err != nil || updated == 0 {
		return false, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "id", id), cache.BuildKey("user", "token", id))
	}
	return true, nil
}

func (r *userRepository) GetTokenVersion(ctx context.Context, id string) (int32, error) {
	if !config.GetBoolConfigWithDefault("TOKEN_VERSION_CACHE", true) || config.GetBoolConfigWithDefault("ENABLE_PREFORK", false) {
		raw, err := r.q.GetUserTokenVersion(ctx, id)
		return int32(raw), err // #nosec G115 -- token version is bounded by application updates
	}
	key := cache.BuildKey("user", "token", id)
	if r.c != nil && !r.inTx {
		var version int32
		if err := r.c.Get(ctx, key, &version); err == nil {
			return version, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		raw, err := r.q.GetUserTokenVersion(ctx, id)
		if err != nil {
			return int32(0), err
		}
		version := int32(raw)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, version, constants.NormalCacheDuration)
		}
		return version, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int32), nil
}

func (r *userRepository) UpdateTokenVersion(ctx context.Context, id string, tokenVersion int64) error {
	user, _ := r.GetByID(ctx, id)
	if err := r.q.UpdateUserTokenVersion(ctx, sqlc.UpdateUserTokenVersionParams{ID: id, TokenVersion: tokenVersion}); err != nil {
		return err
	}
	// Inside a transaction the row is not committed yet: a concurrent reader that misses
	// would re-cache the OLD version and keep the revoked JWT valid for the full TTL.
	// Callers must invalidate after Commit via InvalidateUserCache.
	if r.c != nil && !r.inTx {
		keys := []string{cache.BuildKey("user", "token", id), cache.BuildKey("user", "id", id)}
		if user != nil && user.Email != "" {
			keys = append(keys, cache.BuildKey("user", "email", user.Email))
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}

// InvalidateUserCache drops every cached view of a user. Call it after tx.Commit() for
// mutations made through WithTx, whose own invalidation is deferred to avoid re-caching
// uncommitted state.
//
// email is passed in rather than looked up: every caller already holds the entity, a
// read through GetByID would re-populate the very keys being dropped, and a post-commit
// DB read would return the NEW email when the key that needs clearing is the old one.
// Pass "" when unknown — the id-keyed entries still go.
func (r *userRepository) InvalidateUserCache(ctx context.Context, id, email string) {
	if r.c == nil {
		return
	}
	keys := []string{
		cache.BuildKey("user", "token", id),
		cache.BuildKey("user", "id", id),
		cache.BuildKey("user", "roles", id),
	}
	if email != "" {
		keys = append(keys, cache.BuildKey("user", "email", email))
	}
	_ = r.c.Del(ctx, keys...)
}

func (r *userRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
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
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("user", "email", user.Email), cache.BuildKey("user", "id", user.ID), cache.BuildKey("user", "token", user.ID))
	}
	return nil
}

func (r *userRepository) UpdateRefreshToken(ctx context.Context, id string, refreshToken *string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.UpdateUserRefreshToken(ctx, sqlc.UpdateUserRefreshTokenParams{
		ID:           id,
		RefreshToken: convert.StrPtrToNullString(refreshToken),
	}); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("user", "email", user.Email), cache.BuildKey("user", "id", user.ID), cache.BuildKey("user", "token", user.ID))
	}
	return nil
}
