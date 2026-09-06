package services

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
)

func deduplicateUserIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func (u *userService) BulkDeleteUsers(ctx context.Context, claims *response.JWTClaims, dto *request.BulkUserActionDto) (*response.BulkActionResultResponse, error) {
	dto.UserIDs = deduplicateUserIDs(dto.UserIDs)
	if len(dto.UserIDs) == 0 {
		return &response.BulkActionResultResponse{}, nil
	}

	rootID := u.rootAdminID(ctx)
	isRoot := claims != nil && rootID != "" && claims.UId == rootID

	users, err := u.userRepo.GetByIDs(ctx, dto.UserIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch users")
	}

	userMap := make(map[string]*models.UserEntity, len(users))
	for _, user := range users {
		if user != nil {
			userMap[user.ID] = user
		}
	}

	eligibleIDs := make([]string, 0, len(dto.UserIDs))
	eligibleUsers := make([]*models.UserEntity, 0, len(dto.UserIDs))
	skippedReasons := make([]string, 0)

	for _, rawID := range dto.UserIDs {
		user, exists := userMap[rawID]
		if !exists || user == nil {
			skippedReasons = append(skippedReasons, fmt.Sprintf("User %s not found", rawID))
			continue
		}

		if user.IsDeleted {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Account is already deleted", user.FullName, user.Email))
			continue
		}

		if rootID != "" && user.ID == rootID {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): The owner account cannot be deleted", user.FullName, user.Email))
			continue
		}

		if claims != nil && user.ID == claims.UId {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): You cannot delete your own account", user.FullName, user.Email))
			continue
		}

		if user.IsAdmin() && !isRoot {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Only the owner can delete other admin accounts", user.FullName, user.Email))
			continue
		}

		eligibleIDs = append(eligibleIDs, user.ID)
		eligibleUsers = append(eligibleUsers, user)
	}

	if len(eligibleIDs) > 0 {
		tx, err := u.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		userRepoTx := u.userRepo.WithTx(tx)
		if err := userRepoTx.BulkDelete(ctx, eligibleIDs); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to delete selected users")
		}

		if err := tx.Commit(); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user deletions")
		}

		for _, user := range eligibleUsers {
			u.userRepo.InvalidateUserCache(ctx, user.ID, user.Email)
		}
	}

	return &response.BulkActionResultResponse{
		AffectedCount:  len(eligibleIDs),
		SkippedCount:   len(skippedReasons),
		SkippedReasons: skippedReasons,
	}, nil
}

func (u *userService) BulkRestoreUsers(ctx context.Context, claims *response.JWTClaims, dto *request.BulkUserActionDto) (*response.BulkActionResultResponse, error) {
	dto.UserIDs = deduplicateUserIDs(dto.UserIDs)
	if len(dto.UserIDs) == 0 {
		return &response.BulkActionResultResponse{}, nil
	}

	rootID := u.rootAdminID(ctx)
	isRoot := claims != nil && rootID != "" && claims.UId == rootID

	users, err := u.userRepo.GetByIDs(ctx, dto.UserIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch users")
	}

	userMap := make(map[string]*models.UserEntity, len(users))
	for _, user := range users {
		if user != nil {
			userMap[user.ID] = user
		}
	}

	eligibleIDs := make([]string, 0, len(dto.UserIDs))
	eligibleUsers := make([]*models.UserEntity, 0, len(dto.UserIDs))
	skippedReasons := make([]string, 0)

	for _, rawID := range dto.UserIDs {
		user, exists := userMap[rawID]
		if !exists || user == nil {
			skippedReasons = append(skippedReasons, fmt.Sprintf("User %s not found", rawID))
			continue
		}

		if !user.IsDeleted {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Account is active, restore not needed", user.FullName, user.Email))
			continue
		}

		if rootID != "" && user.ID == rootID {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): The owner account cannot be restored", user.FullName, user.Email))
			continue
		}

		if user.IsAdmin() && !isRoot {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Only the owner can restore admin accounts", user.FullName, user.Email))
			continue
		}

		eligibleIDs = append(eligibleIDs, user.ID)
		eligibleUsers = append(eligibleUsers, user)
	}

	if len(eligibleIDs) > 0 {
		tx, err := u.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		userRepoTx := u.userRepo.WithTx(tx)
		if err := userRepoTx.BulkRestore(ctx, eligibleIDs); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to restore selected users")
		}

		if err := tx.Commit(); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user restores")
		}

		for _, user := range eligibleUsers {
			u.userRepo.InvalidateUserCache(ctx, user.ID, user.Email)
		}
	}

	return &response.BulkActionResultResponse{
		AffectedCount:  len(eligibleIDs),
		SkippedCount:   len(skippedReasons),
		SkippedReasons: skippedReasons,
	}, nil
}

func (u *userService) BulkChangeUserRoles(ctx context.Context, claims *response.JWTClaims, dto *request.BulkChangeUserRolesDto) (*response.BulkActionResultResponse, error) {
	dto.UserIDs = deduplicateUserIDs(dto.UserIDs)
	if len(dto.UserIDs) == 0 {
		return &response.BulkActionResultResponse{}, nil
	}

	roles, err := u.resolveRoles(ctx, dto.RoleIDs)
	if err != nil {
		return nil, err
	}

	rootID := u.rootAdminID(ctx)
	isRoot := claims != nil && rootID != "" && claims.UId == rootID

	hasAdmin := false
	for _, role := range roles {
		if role.IsAdmin || role.Name == constants.RoleTypeAdmin.String() {
			hasAdmin = true
			break
		}
	}

	if hasAdmin && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can grant or modify the ADMIN role")
	}

	users, err := u.userRepo.GetByIDs(ctx, dto.UserIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch users")
	}

	userMap := make(map[string]*models.UserEntity, len(users))
	for _, user := range users {
		if user != nil {
			userMap[user.ID] = user
		}
	}

	eligibleIDs := make([]string, 0, len(dto.UserIDs))
	eligibleUsers := make([]*models.UserEntity, 0, len(dto.UserIDs))
	skippedReasons := make([]string, 0)

	for _, rawID := range dto.UserIDs {
		user, exists := userMap[rawID]
		if !exists || user == nil {
			skippedReasons = append(skippedReasons, fmt.Sprintf("User %s not found", rawID))
			continue
		}

		if user.IsDeleted {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Cannot change roles of deleted accounts", user.FullName, user.Email))
			continue
		}

		if rootID != "" && user.ID == rootID {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): The owner account cannot be modified in bulk operations", user.FullName, user.Email))
			continue
		}

		if claims != nil && user.ID == claims.UId {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): You cannot modify your own roles in bulk operations", user.FullName, user.Email))
			continue
		}

		if user.IsAdmin() && !isRoot {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Only the owner can modify other admin accounts", user.FullName, user.Email))
			continue
		}

		if dto.Action == "unassign" {
			hasAnyRole := false
			for _, ur := range user.Roles {
				if ur != nil && slices.Contains(dto.RoleIDs, ur.ID) {
					hasAnyRole = true
					break
				}
			}
			if !hasAnyRole {
				skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): User does not possess any of the selected roles", user.FullName, user.Email))
				continue
			}
		}

		if dto.Action == "assign" {
			allAlreadyAssigned := true
			for _, roleID := range dto.RoleIDs {
				hasThisRole := false
				for _, ur := range user.Roles {
					if ur != nil && ur.ID == roleID {
						hasThisRole = true
						break
					}
				}
				if !hasThisRole {
					allAlreadyAssigned = false
					break
				}
			}
			if allAlreadyAssigned {
				skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): User already possesses all selected roles", user.FullName, user.Email))
				continue
			}
		}

		eligibleIDs = append(eligibleIDs, user.ID)
		eligibleUsers = append(eligibleUsers, user)
	}

	if len(eligibleIDs) > 0 {
		tx, err := u.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		userRepoTx := u.userRepo.WithTx(tx)
		roleRepoTx := u.roleRepo.WithTx(tx)

		switch dto.Action {
		case "replace":
			if err := roleRepoTx.BulkDeleteRolesFromUsers(ctx, eligibleIDs); err != nil {
				return nil, apperrors.New(apperrors.ErrInternalError, "Failed to clear old roles")
			}
			for _, uid := range eligibleIDs {
				for _, role := range roles {
					if err := roleRepoTx.CreateUserRole(ctx, uid, role.ID); err != nil {
						return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign roles")
					}
				}
			}
		case "assign":
			for _, uid := range eligibleIDs {
				for _, role := range roles {
					if err := roleRepoTx.CreateUserRole(ctx, uid, role.ID); err != nil {
						return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign roles")
					}
				}
			}
		case "unassign":
			for _, role := range roles {
				if err := roleRepoTx.BulkRemoveRoleFromUsers(ctx, role.ID, eligibleIDs); err != nil {
					return nil, apperrors.New(apperrors.ErrInternalError, "Failed to remove roles")
				}
			}
		}

		revokeIDs := make([]string, 0, len(eligibleIDs))
		for _, id := range eligibleIDs {
			if claims == nil || id != claims.UId {
				revokeIDs = append(revokeIDs, id)
			}
		}
		if len(revokeIDs) > 0 {
			if err := userRepoTx.BulkRevokeSessions(ctx, revokeIDs); err != nil {
				return nil, apperrors.New(apperrors.ErrInternalError, "Failed to revoke user sessions")
			}
		}

		if err := tx.Commit(); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit role changes")
		}

		for _, user := range eligibleUsers {
			u.userRepo.InvalidateUserCache(ctx, user.ID, user.Email)
		}
		u.roleRepo.InvalidateRoleCache(ctx, dto.RoleIDs)
	}

	return &response.BulkActionResultResponse{
		AffectedCount:  len(eligibleIDs),
		SkippedCount:   len(skippedReasons),
		SkippedReasons: skippedReasons,
	}, nil
}

func (u *userService) BulkUpdateUserInfo(ctx context.Context, claims *response.JWTClaims, dto *request.BulkUpdateUserInfoDto) (*response.BulkActionResultResponse, error) {
	dto.UserIDs = deduplicateUserIDs(dto.UserIDs)
	if len(dto.UserIDs) == 0 {
		return &response.BulkActionResultResponse{}, nil
	}

	rootID := u.rootAdminID(ctx)
	isRoot := claims != nil && rootID != "" && claims.UId == rootID

	users, err := u.userRepo.GetByIDs(ctx, dto.UserIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch users")
	}

	userMap := make(map[string]*models.UserEntity, len(users))
	for _, user := range users {
		if user != nil {
			userMap[user.ID] = user
		}
	}

	eligibleIDs := make([]string, 0, len(dto.UserIDs))
	eligibleUsers := make([]*models.UserEntity, 0, len(dto.UserIDs))
	skippedReasons := make([]string, 0)

	for _, rawID := range dto.UserIDs {
		user, exists := userMap[rawID]
		if !exists || user == nil {
			skippedReasons = append(skippedReasons, fmt.Sprintf("User %s not found", rawID))
			continue
		}

		if user.IsDeleted {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Cannot update deleted accounts", user.FullName, user.Email))
			continue
		}

		if rootID != "" && user.ID == rootID {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): The owner account cannot be modified in bulk operations", user.FullName, user.Email))
			continue
		}

		if claims != nil && user.ID == claims.UId {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): You cannot modify your own profile in bulk operations", user.FullName, user.Email))
			continue
		}

		if user.IsAdmin() && !isRoot {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Only the owner can modify other admin accounts", user.FullName, user.Email))
			continue
		}

		eligibleIDs = append(eligibleIDs, user.ID)
		eligibleUsers = append(eligibleUsers, user)
	}

	if len(eligibleIDs) > 0 {
		var isKidsMode sql.NullInt64
		if dto.IsKidsMode != nil {
			val := int64(0)
			if *dto.IsKidsMode {
				val = 1
			}
			isKidsMode = sql.NullInt64{Int64: val, Valid: true}
		}

		resetAvatarInt := int64(0)
		if dto.ResetAvatar != nil && *dto.ResetAvatar {
			resetAvatarInt = 1
		}

		revokeSessionsInt := int64(0)
		if dto.RevokeSessions != nil && *dto.RevokeSessions {
			revokeSessionsInt = 1
		}

		params := sqlc.BulkUpdateUserInfoParams{
			MaxAllowedAgeRating: convert.StrPtrToNullString(dto.MaxAllowedAgeRating),
			IsKidsMode:          isKidsMode,
			ResetAvatar:         resetAvatarInt,
			RevokeSessions:      revokeSessionsInt,
			Ids:                 eligibleIDs,
		}

		tx, err := u.txManager.BeginTx(ctx, nil)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		userRepoTx := u.userRepo.WithTx(tx)
		if err := userRepoTx.BulkUpdateInfo(ctx, params); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update user information")
		}

		if err := tx.Commit(); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user updates")
		}

		for _, user := range eligibleUsers {
			u.userRepo.InvalidateUserCache(ctx, user.ID, user.Email)
		}
	}

	return &response.BulkActionResultResponse{
		AffectedCount:  len(eligibleIDs),
		SkippedCount:   len(skippedReasons),
		SkippedReasons: skippedReasons,
	}, nil
}

func (u *userService) BulkSendEmail(ctx context.Context, claims *response.JWTClaims, dto *request.BulkSendUserEmailDto) (*response.BulkActionResultResponse, error) {
	dto.UserIDs = deduplicateUserIDs(dto.UserIDs)
	if len(dto.UserIDs) == 0 {
		return &response.BulkActionResultResponse{}, nil
	}

	if u.settings == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Email delivery is not available")
	}

	users, err := u.userRepo.GetByIDs(ctx, dto.UserIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch users")
	}

	userMap := make(map[string]*models.UserEntity, len(users))
	for _, user := range users {
		if user != nil {
			userMap[user.ID] = user
		}
	}

	eligibleUsers := make([]*models.UserEntity, 0, len(dto.UserIDs))
	skippedReasons := make([]string, 0)

	for _, rawID := range dto.UserIDs {
		user, exists := userMap[rawID]
		if !exists || user == nil {
			skippedReasons = append(skippedReasons, fmt.Sprintf("User %s not found", rawID))
			continue
		}

		if user.IsDeleted {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s (%s): Cannot send email to deleted accounts", user.FullName, user.Email))
			continue
		}

		if strings.TrimSpace(user.Email) == "" {
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s: User has no valid email", user.FullName))
			continue
		}

		eligibleUsers = append(eligibleUsers, user)
	}

	queuedCount := 0
	for _, user := range eligibleUsers {
		subject := strings.ReplaceAll(dto.Subject, "{{name}}", user.FullName)
		subject = strings.ReplaceAll(subject, "{{full_name}}", user.FullName)
		subject = strings.ReplaceAll(subject, "{{email}}", user.Email)

		body := strings.ReplaceAll(dto.Body, "{{name}}", user.FullName)
		body = strings.ReplaceAll(body, "{{full_name}}", user.FullName)
		body = strings.ReplaceAll(body, "{{email}}", user.Email)

		payload, err := jsonx.MarshalString(sendUserEmailPayload{
			UserID:  user.ID,
			Subject: subject,
			Body:    body,
		})
		if err != nil {
			continue
		}

		if u.jobQueue != nil {
			jobID := uuid.Must(uuid.NewV7()).String()
			if err := u.jobQueue.Enqueue(ctx, worker.Job{
				ID:      jobID,
				Type:    "send_user_email",
				Payload: payload,
			}); err == nil {
				queuedCount++
			}
		} else {
			if err := u.ExecuteSendUserEmailJob(ctx, payload); err == nil {
				queuedCount++
			}
		}
	}

	return &response.BulkActionResultResponse{
		AffectedCount:  queuedCount,
		SkippedCount:   len(skippedReasons),
		SkippedReasons: skippedReasons,
	}, nil
}
