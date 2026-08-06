package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

const (
	AuditActionUserCreate     = "user.create"
	AuditActionUserUpdate     = "user.update"
	AuditActionUserDelete     = "user.delete"
	AuditActionUserRestore    = "user.restore"
	AuditActionUserRoleChange = "user.role_change"
	AuditActionUserResetPass  = "user.reset_password"
	AuditActionUserSendEmail  = "user.send_email"
	AuditActionRoleCreate     = "role.create"
	AuditActionRoleUpdate     = "role.update"
	AuditActionRoleDelete     = "role.delete"
	AuditActionRolePermUpdate = "role.permission_update"
	AuditActionSettingsUpdate = "settings.update"
	AuditActionBackupCreate   = "backup.create"
	AuditActionBackupRestore  = "backup.restore"
	AuditActionBackupDelete   = "backup.delete"
	AuditActionBookBulkDelete = "book.bulk_delete"
	AuditActionBookBulkMove   = "book.bulk_move"
	AuditActionTOTPEnable     = "totp.enable"
	AuditActionTOTPDisable    = "totp.disable"

	auditRetentionDays = 90
)

type AuditService interface {
	Record(ctx context.Context, action string, targetType string, targetID string, targetLabel string)
	List(ctx context.Context, dto *request.ListAuditLogsDto) (*response.PaginatedResponse, error)
	ListActions(ctx context.Context) ([]string, error)
	Prune(ctx context.Context) (int64, error)
}

type auditService struct {
	repo     repositories.AuditRepository
	userRepo repositories.UserRepository
}

func NewAuditService(repo repositories.AuditRepository, userRepo repositories.UserRepository) AuditService {
	return &auditService{repo: repo, userRepo: userRepo}
}

func (s *auditService) resolveActorEmail(ctx context.Context, actor AuditActor) string {
	if actor.Email != "" || actor.UserID == "" || s.userRepo == nil {
		return actor.Email
	}
	user, err := s.userRepo.GetByID(ctx, actor.UserID)
	if err != nil || user == nil {
		return ""
	}
	return user.Email
}

func (s *auditService) Record(ctx context.Context, action string, targetType string, targetID string, targetLabel string) {
	actor := AuditActorFrom(ctx)
	if actor.isEmpty() {
		log.Warn().Str("action", action).Msg("audit record has no actor; controller did not attach one")
	}

	entity := &models.AuditLogEntity{
		ID:          uuid.Must(uuid.NewV7()).String(),
		ActorEmail:  s.resolveActorEmail(ctx, actor),
		Action:      action,
		TargetType:  targetType,
		TargetLabel: targetLabel,
		IP:          actor.IP,
	}
	if actor.UserID != "" {
		entity.ActorID = &actor.UserID
	}
	if targetID != "" {
		entity.TargetID = &targetID
	}

	if _, err := s.repo.Create(ctx, entity); err != nil {
		log.Error().Err(err).Str("action", action).Msg("failed to write audit log")
	}
}

func (s *auditService) List(ctx context.Context, dto *request.ListAuditLogsDto) (*response.PaginatedResponse, error) {
	limit := dto.Limit
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		limit = 20
	}

	filter := repositories.AuditFilter{
		Action:  dto.Action,
		ActorID: dto.ActorID,
		Limit:   int64(limit),
	}
	if parts := convert.DecodeCursor(dto.Cursor); len(parts) == 2 {
		filter.CursorCreatedAt = convert.CursorTimeString(parts[0])
		filter.CursorID = parts[1]
	}

	entries, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to list audit logs")
	}
	total, err := s.repo.Count(ctx, repositories.AuditFilter{Action: dto.Action, ActorID: dto.ActorID})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to count audit logs")
	}

	var nextCursor string
	if len(entries) == limit {
		last := entries[len(entries)-1]
		nextCursor = convert.EncodeCursor(last.CreatedAt, last.ID)
	}
	return response.BuildCursorPaginatedResponse(models.AuditLogEntitiesToResponse(entries), total, limit, nextCursor), nil
}

func (s *auditService) ListActions(ctx context.Context) ([]string, error) {
	actions, err := s.repo.ListActions(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to list audit actions")
	}
	return actions, nil
}

func (s *auditService) Prune(ctx context.Context) (int64, error) {
	return s.repo.Prune(ctx, auditRetentionDays)
}
