package services

import (
	"context"
	"database/sql"
	"testing"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/pkg/constants"
)

func TestBulkDeleteUsers(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	// Seed users:
	// 1. Owner: "01920000-0000-7000-8000-0000000000aa" (from seedOwner)
	// 2. Caller (normal admin): "01920000-0000-7000-8000-0000000000bb"
	// 3. Target Admin: "01920000-0000-7000-8000-0000000000cc"
	// 4. Normal user 1: "01920000-0000-7000-8000-0000000000d1"
	// 5. Normal user 2: "01920000-0000-7000-8000-0000000000d2"
	// 6. Already deleted: "01920000-0000-7000-8000-0000000000dd"
	callerID := "01920000-0000-7000-8000-0000000000bb"
	otherAdminID := "01920000-0000-7000-8000-0000000000cc"
	normalUser1 := "01920000-0000-7000-8000-0000000000d1"
	normalUser2 := "01920000-0000-7000-8000-0000000000d2"
	deletedUser := "01920000-0000-7000-8000-0000000000dd"

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'caller@test.com', 'Caller', 'LOCAL', 0)`, callerID)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, callerID, seedRoleAdmin)

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'admin2@test.com', 'Admin2', 'LOCAL', 0)`, otherAdminID)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, otherAdminID, seedRoleAdmin)

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'u1@test.com', 'User1', 'LOCAL', 0)`, normalUser1)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, normalUser1, seedRoleUser)

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'u2@test.com', 'User2', 'LOCAL', 0)`, normalUser2)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, normalUser2, seedRoleUser)

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'del@test.com', 'DelUser', 'LOCAL', 1)`, deletedUser)

	claims := &response.JWTClaims{
		UId:   callerID,
		Roles: []constants.RoleType{constants.RoleTypeAdmin},
	}

	// Try deleting: Owner, Caller (self), other Admin, already deleted, normalUser1, normalUser2
	res, err := svc.BulkDeleteUsers(ctx, claims, &request.BulkUserActionDto{
		UserIDs: []string{
			"01920000-0000-7000-8000-0000000000aa", // owner
			callerID,                                // self
			otherAdminID,                            // other admin
			deletedUser,                             // already deleted
			normalUser1,                             // valid
			normalUser2,                             // valid
		},
	})
	if err != nil {
		t.Fatalf("BulkDeleteUsers error: %v", err)
	}

	if res.AffectedCount != 2 {
		t.Errorf("Expected 2 affected, got %d", res.AffectedCount)
	}
	if res.SkippedCount != 4 {
		t.Errorf("Expected 4 skipped, got %d", res.SkippedCount)
	}

	// Verify normalUser1 and normalUser2 are soft-deleted in DB
	var isDeleted1, isDeleted2 int
	_ = db.QueryRow(`SELECT is_deleted FROM users WHERE id = ?`, normalUser1).Scan(&isDeleted1)
	_ = db.QueryRow(`SELECT is_deleted FROM users WHERE id = ?`, normalUser2).Scan(&isDeleted2)
	if isDeleted1 != 1 || isDeleted2 != 1 {
		t.Errorf("Expected users to be deleted, got %d, %d", isDeleted1, isDeleted2)
	}
}

func TestBulkRestoreUsers(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	callerID := "01920000-0000-7000-8000-0000000000bb"
	deletedAdminID := "01920000-0000-7000-8000-0000000000cc"
	activeUser := "01920000-0000-7000-8000-0000000000d1"
	deletedUser := "01920000-0000-7000-8000-0000000000d2"

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'caller@test.com', 'Caller', 'LOCAL', 0)`, callerID)
	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'deladmin@test.com', 'DelAdmin', 'LOCAL', 1)`, deletedAdminID)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, deletedAdminID, seedRoleAdmin)
	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'active@test.com', 'Active', 'LOCAL', 0)`, activeUser)
	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'deleted@test.com', 'Deleted', 'LOCAL', 1)`, deletedUser)

	claims := &response.JWTClaims{
		UId:   callerID,
		Roles: []constants.RoleType{constants.RoleTypeAdmin},
	}

	res, err := svc.BulkRestoreUsers(ctx, claims, &request.BulkUserActionDto{
		UserIDs: []string{deletedAdminID, activeUser, deletedUser},
	})
	if err != nil {
		t.Fatalf("BulkRestoreUsers error: %v", err)
	}

	// deletedAdmin cannot be restored by non-owner, activeUser is skipped because not deleted, deletedUser is restored
	if res.AffectedCount != 1 {
		t.Errorf("Expected 1 affected, got %d", res.AffectedCount)
	}
	if res.SkippedCount != 2 {
		t.Errorf("Expected 2 skipped, got %d", res.SkippedCount)
	}

	var isDel int
	_ = db.QueryRow(`SELECT is_deleted FROM users WHERE id = ?`, deletedUser).Scan(&isDel)
	if isDel != 0 {
		t.Errorf("Expected deletedUser to be restored, got is_deleted=%d", isDel)
	}
}

func TestBulkChangeUserRoles(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	callerID := "01920000-0000-7000-8000-0000000000bb"
	normalUser := "01920000-0000-7000-8000-0000000000d1"

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'caller@test.com', 'Caller', 'LOCAL', 0)`, callerID)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, callerID, seedRoleAdmin)

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, auth_provider, is_deleted) VALUES (?, 'normal@test.com', 'Normal', 'LOCAL', 0)`, normalUser)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, normalUser, seedRoleUser)

	claims := &response.JWTClaims{
		UId:   callerID,
		Roles: []constants.RoleType{constants.RoleTypeAdmin},
	}

	// Normal admin cannot grant ADMIN role
	_, err := svc.BulkChangeUserRoles(ctx, claims, &request.BulkChangeUserRolesDto{
		UserIDs: []string{normalUser},
		Action:  "assign",
		RoleIDs: []string{seedRoleAdmin},
	})
	if err == nil {
		t.Error("Expected error when non-owner tries to grant ADMIN role")
	}

	// Admin cannot self-ban
	res, err := svc.BulkChangeUserRoles(ctx, claims, &request.BulkChangeUserRolesDto{
		UserIDs: []string{callerID, normalUser},
		Action:  "assign",
		RoleIDs: []string{seedRoleBanned},
	})
	if err != nil {
		t.Fatalf("BulkChangeUserRoles error: %v", err)
	}
	if res.AffectedCount != 1 {
		t.Errorf("Expected 1 affected (normalUser), got %d", res.AffectedCount)
	}
	if res.SkippedCount != 1 {
		t.Errorf("Expected 1 skipped (caller self-ban), got %d", res.SkippedCount)
	}
}

func TestBulkUpdateUserInfo(t *testing.T) {
	svc, db := newUserSvc(t)
	seedOwner(t, db)
	ctx := context.Background()

	callerID := "01920000-0000-7000-8000-0000000000bb"
	normalUser := "01920000-0000-7000-8000-0000000000d1"

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, avatar_url, auth_provider, is_deleted, is_kids_mode, max_allowed_age_rating) VALUES (?, 'caller@test.com', 'Caller', 'http://a.com/av.jpg', 'LOCAL', 0, 0, 'R18+')`, callerID)
	_, _ = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, callerID, seedRoleAdmin)

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, avatar_url, auth_provider, is_deleted, is_kids_mode, max_allowed_age_rating) VALUES (?, 'normal@test.com', 'Normal', 'http://a.com/av2.jpg', 'LOCAL', 0, 0, 'R18+')`, normalUser)

	claims := &response.JWTClaims{
		UId:   callerID,
		Roles: []constants.RoleType{constants.RoleTypeAdmin},
	}

	kidsModeTrue := true
	ageRating := "PG-13"
	resetAvatar := true
	revokeSessions := true

	res, err := svc.BulkUpdateUserInfo(ctx, claims, &request.BulkUpdateUserInfoDto{
		UserIDs:             []string{normalUser},
		MaxAllowedAgeRating: &ageRating,
		IsKidsMode:          &kidsModeTrue,
		ResetAvatar:         &resetAvatar,
		RevokeSessions:      &revokeSessions,
	})
	if err != nil {
		t.Fatalf("BulkUpdateUserInfo error: %v", err)
	}
	if res.AffectedCount != 1 {
		t.Errorf("Expected 1 affected, got %d", res.AffectedCount)
	}

	var kidsMode int
	var rating string
	var avatar sql.NullString
	_ = db.QueryRow(`SELECT is_kids_mode, max_allowed_age_rating, avatar_url FROM users WHERE id = ?`, normalUser).Scan(&kidsMode, &rating, &avatar)
	if kidsMode != 1 || rating != "PG-13" || avatar.Valid {
		t.Errorf("Expected kids_mode=1, rating=PG-13, avatar=NULL; got kids_mode=%d, rating=%s, avatar=%v", kidsMode, rating, avatar)
	}
}
