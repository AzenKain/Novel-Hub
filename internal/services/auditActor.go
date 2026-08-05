package services

import (
	"context"
	"strings"
)

type auditActorKey struct{}

// On the context, not a parameter: eleven audited methods never receive JWT claims, and the
// IP is only known to the fiber handler. Mirrors WithPermissionContext in accessCache.go.
type AuditActor struct {
	UserID string
	Email  string
	IP     string
}

func WithAuditActor(ctx context.Context, actor AuditActor) context.Context {
	return context.WithValue(ctx, auditActorKey{}, actor)
}

func AuditActorFrom(ctx context.Context) AuditActor {
	actor, _ := ctx.Value(auditActorKey{}).(AuditActor)
	return actor
}

func (a AuditActor) isEmpty() bool {
	return strings.TrimSpace(a.UserID) == "" && strings.TrimSpace(a.Email) == ""
}
