package services

import (
	"context"
	"errors"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"testing"
)

type sessionPermissionStub struct{ allowed bool }

func (p sessionPermissionStub) Reload(context.Context) error { return nil }
func (p sessionPermissionStub) Can(context.Context, string, string, map[string]any) bool {
	return p.allowed
}
func (p sessionPermissionStub) CanRoles([]string, []constants.RoleType, string, map[string]any) bool {
	return p.allowed
}
func (p sessionPermissionStub) IsAdmin([]string, []constants.RoleType) bool { return false }
func (p sessionPermissionStub) GetGuestPermissions() []string               { return nil }

type sessionBookRepo struct {
	repositories.BookDBRepository
	book *models.BookEntity
	err  error
}

func (r sessionBookRepo) GetBook(context.Context, string) (*models.BookEntity, error) {
	return r.book, r.err
}

type sessionFeatureRepo struct {
	repositories.FeatureRepository
	err error
	got sqlc.UpsertReadingSessionParams
}

func (r *sessionFeatureRepo) UpsertReadingSession(_ context.Context, a sqlc.UpsertReadingSessionParams) (*models.ReadingSessionEntity, error) {
	r.got = a
	return nil, r.err
}
func newSessionService(r *sessionFeatureRepo, b *models.BookEntity, be error, a bool) *featureService {
	return &featureService{repo: r, bookRepo: sessionBookRepo{book: b, err: be}, permissions: sessionPermissionStub{allowed: a}}
}
func TestRecordReadingSessionGeneratesID(t *testing.T) {
	r := &sessionFeatureRepo{}
	if e := newSessionService(r, &models.BookEntity{LibraryID: "l"}, nil, true).RecordReadingSession(context.Background(), "u", "b", 3, 7, &response.JWTClaims{}); e != nil {
		t.Fatal(e)
	}
	if r.got.ID == "" {
		t.Fatal("missing ID")
	}
}
func TestRecordReadingSessionInaccessibleBook(t *testing.T) {
	r := &sessionFeatureRepo{}
	e := newSessionService(r, nil, errors.New("missing"), true).RecordReadingSession(context.Background(), "u", "b", 1, 0, &response.JWTClaims{})
	if !errors.Is(e, apperrors.ErrForbidden) {
		t.Fatal(e)
	}
	if r.got.ID != "" {
		t.Fatal("called repository")
	}
}
func TestRecordReadingSessionRepositoryFailurePreservesCause(t *testing.T) {
	c := errors.New("NOT NULL constraint failed: reading_sessions.id")
	r := &sessionFeatureRepo{err: c}
	e := newSessionService(r, &models.BookEntity{LibraryID: "l"}, nil, true).RecordReadingSession(context.Background(), "u", "b", 1, 0, &response.JWTClaims{})
	if !errors.Is(e, apperrors.ErrInternalError) || !errors.Is(e, c) {
		t.Fatalf("causes not preserved: %v", e)
	}
}

var _ repositories.BookDBRepository = sessionBookRepo{}
var _ repositories.FeatureRepository = (*sessionFeatureRepo)(nil)
