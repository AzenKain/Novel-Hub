package services

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
)

type permissionContextKey struct{}

type PermissionContext struct {
	RoleIDs []int64
	Roles   []constants.RoleType
}

func WithPermissionContext(ctx context.Context, permissionCtx PermissionContext) context.Context {
	return context.WithValue(ctx, permissionContextKey{}, permissionCtx)
}

func PermissionContextFrom(ctx context.Context) PermissionContext {
	value, _ := ctx.Value(permissionContextKey{}).(PermissionContext)
	return value
}

type PermissionCache interface {
	Reload(ctx context.Context) error
	Can(ctx context.Context, userID string, permission string, attrs map[string]any) bool
	CanRoles(roleIDs []int64, roles []constants.RoleType, permission string, attrs map[string]any) bool
	IsAdmin(roleIDs []int64, roles []constants.RoleType) bool
	GetGuestPermissions() []string
}

type cachedRolePermission struct {
	PermissionKey string
	Effect        string
	Conditions    map[string]any
}

type cachedRole struct {
	ID          int64
	Name        string
	IsAdmin     bool
	Position    int64
	Permissions []cachedRolePermission
}

type permissionCache struct {
	roleRepo repositories.RoleRepository
	mu       sync.RWMutex
	roles    map[int64]*cachedRole
	nameToID map[string]int64
}

func NewPermissionCache(roleRepo repositories.RoleRepository) PermissionCache {
	return &permissionCache{
		roleRepo: roleRepo,
		roles:    map[int64]*cachedRole{},
		nameToID: map[string]int64{},
	}
}

func (p *permissionCache) Reload(ctx context.Context) error {
	roles, err := p.roleRepo.All(ctx)
	if err != nil {
		return err
	}
	rolePermissions, err := p.roleRepo.ListRolePermissions(ctx)
	if err != nil {
		return err
	}

	nextRoles := make(map[int64]*cachedRole, len(roles))
	nextNames := make(map[string]int64, len(roles))
	for _, role := range roles {
		if role == nil || role.IsDeleted {
			continue
		}
		nextRoles[role.ID] = &cachedRole{
			ID:       role.ID,
			Name:     role.Name,
			IsAdmin:  role.IsAdmin,
			Position: role.Position,
		}
		nextNames[role.Name] = role.ID
	}

	for _, permission := range rolePermissions {
		if permission == nil {
			continue
		}
		role, ok := nextRoles[permission.RoleID]
		if !ok {
			continue
		}
		role.Permissions = append(role.Permissions, cachedRolePermission{
			PermissionKey: permission.PermissionKey,
			Effect:         permission.Effect,
			Conditions:     permission.Conditions,
		})
	}

	p.mu.Lock()
	p.roles = nextRoles
	p.nameToID = nextNames
	p.mu.Unlock()
	return nil
}

func (p *permissionCache) Can(ctx context.Context, userID string, permission string, attrs map[string]any) bool {
	permissionCtx := PermissionContextFrom(ctx)
	return p.CanRoles(permissionCtx.RoleIDs, permissionCtx.Roles, permission, attrs)
}

func (p *permissionCache) CanRoles(roleIDs []int64, roles []constants.RoleType, permission string, attrs map[string]any) bool {
	if permission == "" {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	resolvedRoleIDs := p.resolveRoleIDs(roleIDs, roles)

	if p.hasBanned(resolvedRoleIDs) {
		return false
	}

	if p.hasAdmin(resolvedRoleIDs) {
		return true
	}

	sortedIDs := append([]int64(nil), resolvedRoleIDs...)
	sort.Slice(sortedIDs, func(i, j int) bool {
		rI := p.roles[sortedIDs[i]]
		rJ := p.roles[sortedIDs[j]]
		posI, posJ := int64(0), int64(0)
		if rI != nil {
			posI = rI.Position
		}
		if rJ != nil {
			posJ = rJ.Position
		}
		return posI > posJ
	})

	allowed := false
	for _, roleID := range sortedIDs {
		role, ok := p.roles[roleID]
		if !ok || role == nil {
			continue
		}
		for _, rolePermission := range role.Permissions {
			if rolePermission.PermissionKey != permission {
				continue
			}
			if !conditionsMatch(rolePermission.Conditions, attrs) {
				continue
			}
			if rolePermission.Effect == "deny" {
				return false
			}
			if rolePermission.Effect == "allow" {
				allowed = true
			}
		}
	}
	return allowed
}

func (p *permissionCache) IsAdmin(roleIDs []int64, roles []constants.RoleType) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	resolved := p.resolveRoleIDs(roleIDs, roles)
	if p.hasBanned(resolved) {
		return false
	}
	return p.hasAdmin(resolved)
}

func (p *permissionCache) GetGuestPermissions() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	guestRole, ok := p.roles[constants.SystemRoleIDGuest]
	if !ok || guestRole == nil {
		return nil
	}
	keys := make([]string, 0, len(guestRole.Permissions))
	for _, perm := range guestRole.Permissions {
		if perm.Effect != "deny" {
			keys = append(keys, perm.PermissionKey)
		}
	}
	return keys
}

func (p *permissionCache) hasBanned(roleIDs []int64) bool {
	for _, id := range roleIDs {
		if id == constants.SystemRoleIDBanned {
			return true
		}
		if role, ok := p.roles[id]; ok && role != nil {
			if strings.EqualFold(role.Name, string(constants.RoleTypeBanned)) {
				return true
			}
		}
	}
	return false
}

func (p *permissionCache) hasAdmin(roleIDs []int64) bool {
	for _, id := range roleIDs {
		if id == constants.SystemRoleIDAdmin {
			return true
		}
		if role, ok := p.roles[id]; ok && role != nil {
			if role.IsAdmin || strings.EqualFold(role.Name, string(constants.RoleTypeAdmin)) {
				return true
			}
		}
	}
	return false
}

func (p *permissionCache) resolveRoleIDs(roleIDs []int64, roles []constants.RoleType) []int64 {
	out := make([]int64, 0, len(roleIDs)+len(roles))
	for _, id := range roleIDs {
		if id > 0 && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, role := range roles {
		if id, ok := p.nameToID[role.String()]; ok && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func conditionsMatch(conditions map[string]any, attrs map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	if libraryIDs, ok := stringSliceCondition(conditions["library_ids"]); ok {
		if len(libraryIDs) == 0 {
			return true
		}
		libraryID := attrString(attrs, "library_id")
		if libraryID == "" {
			return false
		}
		return slices.Contains(libraryIDs, libraryID)
	}
	return true
}

func stringSliceCondition(value any) ([]string, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case []string:
		return typed, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out, true
	default:
		return []string{fmt.Sprint(typed)}, true
	}
}

func attrString(attrs map[string]any, key string) string {
	if len(attrs) == 0 {
		return ""
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}
