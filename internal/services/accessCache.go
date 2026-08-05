package services

import (
	"context"
	"slices"
	"sort"
	"strings"
	"sync"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/constants"
)

type permissionContextKey struct{}

func WithPermissionContext(ctx context.Context, claims *response.JWTClaims) context.Context {
	return context.WithValue(ctx, permissionContextKey{}, claims)
}

func PermissionContextFrom(ctx context.Context) *response.JWTClaims {
	claims, _ := ctx.Value(permissionContextKey{}).(*response.JWTClaims)
	return claims
}

type PermissionCache interface {
	Reload(ctx context.Context) error
	Can(ctx context.Context, userID string, permission string, attrs map[string]any) bool
	CanRoles(roleIDs []string, roles []constants.RoleType, permission string, attrs map[string]any) bool
	IsAdmin(roleIDs []string, roles []constants.RoleType) bool
	GetGuestPermissions() []string
}

type cachedRolePermission struct {
	PermissionKey string
	Effect        string
	Conditions    map[string]any
}

type cachedRole struct {
	ID          string
	Name        string
	IsAdmin     bool
	Position    int64
	Permissions []cachedRolePermission
}

type permissionCache struct {
	roleRepo repositories.RoleRepository
	mu       sync.RWMutex
	roles    map[string]*cachedRole
	nameToID map[string]string
}

func NewPermissionCache(roleRepo repositories.RoleRepository) PermissionCache {
	return &permissionCache{
		roleRepo: roleRepo,
		roles:    map[string]*cachedRole{},
		nameToID: map[string]string{},
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

	nextRoles := make(map[string]*cachedRole, len(roles))
	nextNames := make(map[string]string, len(roles))
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
			Effect:        permission.Effect,
			Conditions:    permission.Conditions,
		})
	}

	p.mu.Lock()
	p.roles = nextRoles
	p.nameToID = nextNames
	p.mu.Unlock()
	return nil
}

func (p *permissionCache) Can(ctx context.Context, userID string, permission string, attrs map[string]any) bool {
	claims := PermissionContextFrom(ctx)
	if claims == nil {
		return false
	}
	return p.CanRoles(claims.RoleIDs, claims.Roles, permission, attrs)
}

func (p *permissionCache) CanRoles(roleIDs []string, roles []constants.RoleType, permission string, attrs map[string]any) bool {
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

	sortedIDs := append([]string(nil), resolvedRoleIDs...)
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

func (p *permissionCache) IsAdmin(roleIDs []string, roles []constants.RoleType) bool {
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

	guestRoleID, ok := p.nameToID[constants.RoleTypeGuest.String()]
	if !ok {
		return nil
	}
	guestRole, ok := p.roles[guestRoleID]
	if !ok || guestRole == nil {
		return nil
	}
	keys := make([]string, 0, len(guestRole.Permissions))
	for _, perm := range guestRole.Permissions {
		if perm.Effect == "allow" && len(perm.Conditions) == 0 {
			keys = append(keys, perm.PermissionKey)
		}
	}
	return keys
}

func (p *permissionCache) hasBanned(roleIDs []string) bool {
	for _, id := range roleIDs {
		if role, ok := p.roles[id]; ok && role != nil {
			if strings.EqualFold(role.Name, string(constants.RoleTypeBanned)) {
				return true
			}
		}
	}
	return false
}

func (p *permissionCache) hasAdmin(roleIDs []string) bool {
	for _, id := range roleIDs {
		if role, ok := p.roles[id]; ok && role != nil {
			if role.IsAdmin || strings.EqualFold(role.Name, string(constants.RoleTypeAdmin)) {
				return true
			}
		}
	}
	return false
}

func (p *permissionCache) resolveRoleIDs(roleIDs []string, roles []constants.RoleType) []string {
	out := make([]string, 0, len(roleIDs)+len(roles))
	for _, id := range roleIDs {
		if id != "" && !slices.Contains(out, id) {
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
	if len(conditions) != 1 {
		return false
	}
	libraryIDs, ok := strictLibraryIDs(conditions["library_ids"])
	if !ok {
		return false
	}
	libraryID, ok := attrs["library_id"].(string)
	return ok && libraryID != "" && slices.Contains(libraryIDs, libraryID)
}

func strictLibraryIDs(value any) ([]string, bool) {
	var libraryIDs []string
	switch typed := value.(type) {
	case []string:
		libraryIDs = typed
	case []any:
		libraryIDs = make([]string, len(typed))
		for i, item := range typed {
			id, ok := item.(string)
			if !ok {
				return nil, false
			}
			libraryIDs[i] = id
		}
	default:
		return nil, false
	}
	if len(libraryIDs) == 0 {
		return nil, false
	}
	for _, id := range libraryIDs {
		if strings.TrimSpace(id) == "" {
			return nil, false
		}
	}
	return libraryIDs, true
}

func validPermissionConditions(conditions map[string]any) bool {
	return len(conditions) == 0 || conditionsMatch(conditions, map[string]any{"library_id": firstLibraryID(conditions)})
}

func firstLibraryID(conditions map[string]any) string {
	ids, ok := strictLibraryIDs(conditions["library_ids"])
	if !ok {
		return ""
	}
	return ids[0]
}
