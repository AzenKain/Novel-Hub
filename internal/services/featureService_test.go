package services

import (
	"context"
	"errors"
	"testing"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/constants"

	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
)

type sessionPermissionStub struct{ allowed bool }

func (p sessionPermissionStub) Reload(_ context.Context) error { return nil }
func (p sessionPermissionStub) Can(context.Context, string, string, map[string]any) bool {
	return p.allowed
}
func (p sessionPermissionStub) CanRoles([]string, []constants.RoleType, string, map[string]any) bool {
	return p.allowed
}
func (p sessionPermissionStub) IsAdmin([]string, []constants.RoleType) bool { return false }
func (p sessionPermissionStub) GetGuestPermissions() []string               { return nil }

func TestReadingSessionBookAccessibleForbidden(t *testing.T) {
	book := &models.BookEntity{LibraryID: "library"}
	claims := &response.JWTClaims{}
	if readingSessionBookAccessible(nil, nil, sessionPermissionStub{allowed: true}, claims) {
		t.Fatal("nil book must be forbidden")
	}
	if readingSessionBookAccessible(book, errors.New("missing"), sessionPermissionStub{allowed: true}, claims) {
		t.Fatal("book lookup errors must be forbidden")
	}
	if readingSessionBookAccessible(book, nil, sessionPermissionStub{}, claims) {
		t.Fatal("denied permission must be forbidden")
	}
}

func TestReadingSessionRepositoryErrorPreservesInternalCause(t *testing.T) {
	cause := errors.New("NOT NULL constraint failed: reading_sessions.id")
	err := apperrors.New(errors.Join(apperrors.ErrInternalError, cause), "Failed to record reading session")
	if !errors.Is(err, apperrors.ErrInternalError) || !errors.Is(err, cause) {
		t.Fatalf("error does not preserve causes: %v", err)
	}
}
