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

type RoleRepository interface {
	GetByID(ctx context.Context, id int64) (*models.RoleEntity, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*models.RoleEntity, error)
	GetByName(ctx context.Context, name string) (*models.RoleEntity, error)
	All(ctx context.Context) ([]*models.RoleEntity, error)
	ListPermissions(ctx context.Context) ([]*models.PermissionEntity, error)
	ListRolePermissions(ctx context.Context) ([]*models.RolePermissionEntity, error)
	GetRolePermissions(ctx context.Context, roleID int64) ([]*models.RolePermissionEntity, error)
	Create(ctx context.Context, params RoleCreateParams) (*models.RoleEntity, error)
	Update(ctx context.Context, params RoleUpdateParams) (*models.RoleEntity, error)
	Delete(ctx context.Context, id int64) error
	ReplaceRolePermissions(ctx context.Context, roleID int64, permissions []*models.RolePermissionEntity) error
	CreateUserRole(ctx context.Context, userID, roleID int64) error
	BulkDeleteRolesFromUser(ctx context.Context, userID int64) error
	GetAutoAssignRoleIDs(ctx context.Context) ([]int64, error)
	WithTx(tx *sql.Tx) RoleRepository
}

type RoleCreateParams struct {
	Name        string
	Description string
	IsSystem    bool
	IsAdmin     bool
	AutoAssign  bool
}

type RoleUpdateParams struct {
	ID          int64
	Name        string
	Description string
	AutoAssign  bool
	SystemOnly  bool
}

type roleRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewRoleRepository(db sqlc.DBTX, c cache.Cache) RoleRepository {
	return &roleRepository{q: sqlc.New(db), c: c}
}

func (r *roleRepository) WithTx(tx *sql.Tx) RoleRepository {
	return &roleRepository{q: r.q.WithTx(tx), c: r.c}
}

func mapRole(row sqlc.Role) *models.RoleEntity {
	return &models.RoleEntity{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem != 0,
		IsAdmin:     row.IsAdmin != 0,
		AutoAssign:  row.AutoAssign != 0,
		IsDeleted:   row.IsDeleted != 0,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapPermission(row sqlc.Permission) *models.PermissionEntity {
	return &models.PermissionEntity{
		Key:         row.Key,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapRolePermission(row sqlc.RolePermission) *models.RolePermissionEntity {
	var conditions map[string]any
	if row.ConditionsJson != "" {
		_ = jsonx.Unmarshal([]byte(row.ConditionsJson), &conditions)
	}
	if conditions == nil {
		conditions = map[string]any{}
	}
	return &models.RolePermissionEntity{
		ID:             row.ID,
		RoleID:         row.RoleID,
		PermissionKey:  row.PermissionKey,
		Effect:         row.Effect,
		ConditionsJSON: row.ConditionsJson,
		Conditions:     conditions,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func (r *roleRepository) GetByIDs(ctx context.Context, ids []int64) ([]*models.RoleEntity, error) {
	if len(ids) == 0 {
		return []*models.RoleEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = "role:id:" + strconv.FormatInt(id, 10)
	}

	roles := make([]*models.RoleEntity, 0, len(ids))
	missingIds := []int64{}
	missingKeys := []string{}

	if r.c != nil {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var role models.RoleEntity
				if err := jsonx.Unmarshal(bytes, &role); err == nil {
					roles = append(roles, &role)
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
		rows, err := r.q.GetRolesByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[int64]*models.RoleEntity)
		for _, row := range rows {
			role := mapRole(row)
			missingMap[role.ID] = role
			roles = append(roles, role)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if r, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = r
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	roleMap := make(map[int64]*models.RoleEntity)
	for _, role := range roles {
		roleMap[role.ID] = role
	}
	ordered := make([]*models.RoleEntity, 0, len(ids))
	for _, id := range ids {
		if r, ok := roleMap[id]; ok {
			ordered = append(ordered, r)
		}
	}

	return ordered, nil
}

func (r *roleRepository) GetByID(ctx context.Context, id int64) (*models.RoleEntity, error) {
	key := "role:id:" + strconv.FormatInt(id, 10)
	if r.c != nil {
		var role models.RoleEntity
		if err := r.c.Get(ctx, key, &role); err == nil {
			return &role, nil
		}
	}

	row, err := r.q.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rolePtr := mapRole(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, rolePtr, constants.NormalCacheDuration)
	}
	return rolePtr, nil
}

func (r *roleRepository) ListPermissions(ctx context.Context) ([]*models.PermissionEntity, error) {
	rows, err := r.q.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*models.PermissionEntity, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPermission(row))
	}
	return out, nil
}

func (r *roleRepository) ListRolePermissions(ctx context.Context) ([]*models.RolePermissionEntity, error) {
	rows, err := r.q.ListRolePermissions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*models.RolePermissionEntity, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapRolePermission(row))
	}
	return out, nil
}

func (r *roleRepository) GetRolePermissions(ctx context.Context, roleID int64) ([]*models.RolePermissionEntity, error) {
	rows, err := r.q.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	out := make([]*models.RolePermissionEntity, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapRolePermission(row))
	}
	return out, nil
}

func (r *roleRepository) Create(ctx context.Context, params RoleCreateParams) (*models.RoleEntity, error) {
	row, err := r.q.CreateRole(ctx, sqlc.CreateRoleParams{
		Name:        params.Name,
		Description: params.Description,
		IsSystem:    boolToInt64(params.IsSystem),
		IsAdmin:     boolToInt64(params.IsAdmin),
		AutoAssign:  boolToInt64(params.AutoAssign),
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "role:all")
	}
	return mapRole(row), nil
}

func (r *roleRepository) Update(ctx context.Context, params RoleUpdateParams) (*models.RoleEntity, error) {
	var row sqlc.Role
	var err error
	if params.SystemOnly {
		row, err = r.q.UpdateSystemRoleDescription(ctx, sqlc.UpdateSystemRoleDescriptionParams{
			ID:          params.ID,
			Description: params.Description,
		})
	} else {
		row, err = r.q.UpdateRole(ctx, sqlc.UpdateRoleParams{
			ID:          params.ID,
			Name:        params.Name,
			Description: params.Description,
			AutoAssign:  boolToInt64(params.AutoAssign),
		})
	}
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "role:all", "role:id:"+strconv.FormatInt(params.ID, 10), "role:name:"+row.Name)
	}
	return mapRole(row), nil
}

func (r *roleRepository) Delete(ctx context.Context, id int64) error {
	if err := r.q.DeleteRole(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "role:all", "role:id:"+strconv.FormatInt(id, 10))
		_ = r.c.DelByPattern(context.Background(), "user:*")
	}
	return nil
}

func (r *roleRepository) ReplaceRolePermissions(ctx context.Context, roleID int64, permissions []*models.RolePermissionEntity) error {
	if err := r.q.DeleteRolePermissions(ctx, roleID); err != nil {
		return err
	}
	for _, permission := range permissions {
		if permission == nil {
			continue
		}
		conditions := permission.ConditionsJSON
		if conditions == "" {
			data, err := jsonx.Marshal(permission.Conditions)
			if err != nil {
				return err
			}
			conditions = string(data)
		}
		effect := permission.Effect
		if effect == "" {
			effect = "allow"
		}
		if err := r.q.UpsertRolePermission(ctx, sqlc.UpsertRolePermissionParams{
			RoleID:         roleID,
			PermissionKey:  permission.PermissionKey,
			Effect:         effect,
			ConditionsJson: conditions,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*models.RoleEntity, error) {
	key := "role:name:" + name
	if r.c != nil {
		var role models.RoleEntity
		if err := r.c.Get(ctx, key, &role); err == nil {
			return &role, nil
		}
	}

	row, err := r.q.GetRoleByName(ctx, name)
	if err != nil {
		return nil, err
	}
	rolePtr := mapRole(row)
	if r.c != nil {
		_ = r.c.Set(ctx, key, rolePtr, constants.NormalCacheDuration)
	}
	return rolePtr, nil
}

func (r *roleRepository) All(ctx context.Context) ([]*models.RoleEntity, error) {
	key := "role:all"
	if r.c != nil {
		var ids []int64
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetByIDs(ctx, ids)
		}
	}

	dbIds, err := r.q.GetRoleIDs(ctx)
	if err != nil {
		return nil, err
	}

	if r.c != nil {
		_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
	}
	return r.GetByIDs(ctx, dbIds)
}

func (r *roleRepository) CreateUserRole(ctx context.Context, userID, roleID int64) error {
	if err := r.q.CreateUserRole(ctx, sqlc.CreateUserRoleParams{UserID: userID, RoleID: roleID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:id:"+strconv.FormatInt(userID, 10), "user:token:"+strconv.FormatInt(userID, 10))
		_ = r.c.DelByPattern(context.Background(), "user:search*")
		_ = r.c.DelByPattern(context.Background(), "user:count*")
	}
	return nil
}

func (r *roleRepository) BulkDeleteRolesFromUser(ctx context.Context, userID int64) error {
	if err := r.q.BulkDeleteRolesFromUser(ctx, userID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "user:id:"+strconv.FormatInt(userID, 10), "user:token:"+strconv.FormatInt(userID, 10))
		_ = r.c.DelByPattern(context.Background(), "user:search*")
		_ = r.c.DelByPattern(context.Background(), "user:count*")
	}
	return nil
}

func (r *roleRepository) GetAutoAssignRoleIDs(ctx context.Context) ([]int64, error) {
	return r.q.GetAutoAssignRoleIDs(ctx)
}

func DecodeRole(raw []byte) (*models.RoleEntity, error) {
	var role models.RoleEntity
	if err := jsonx.Unmarshal(raw, &role); err != nil {
		return nil, err
	}
	return &role, nil
}
