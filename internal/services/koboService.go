package services

import (
	"bytes"
	"context"
	"io"
	"os"

	"novelhub/internal/dtos/response"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/kepub"
)

type KoboService interface {
	GetInitialization(ctx context.Context) (map[string]any, error)
	GetUserProfile(ctx context.Context) (map[string]any, error)
	GetSyncList(ctx context.Context, syncToken string, claims *response.JWTClaims) (map[string]any, error)
	GetBookKePubStream(ctx context.Context, bookID string, claims *response.JWTClaims, out io.Writer) error
	SyncState(ctx context.Context, userID int64, stateData map[string]any) error
}

type koboService struct {
	bookRepo    repositories.BookDBRepository
	diskRepo    repositories.BookFileRepository
	bookService BookService
	permissions PermissionCache
}

func NewKoboService(bookRepo repositories.BookDBRepository, diskRepo repositories.BookFileRepository, bookService BookService, permissions PermissionCache) KoboService {
	return &koboService{
		bookRepo:    bookRepo,
		diskRepo:    diskRepo,
		bookService: bookService,
		permissions: permissions,
	}
}

func (s *koboService) GetInitialization(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"Resources": map[string]string{
			"user_profile":     "/kobo/v1/user/profile",
			"sync":             "/kobo/v1/library/sync",
			"analytics":        "/kobo/v1/analytics",
			"reading_services": "/kobo/v1/user/services",
		},
	}, nil
}

func (s *koboService) GetUserProfile(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"User": map[string]any{
			"UserKey":   "novelhub-kobo-user",
			"UserToken": "novelhub-token",
			"UserEmail": "user@novelhub.local",
			"IsGuest":   false,
		},
	}, nil
}

func (s *koboService) GetSyncList(ctx context.Context, syncToken string, claims *response.JWTClaims) (map[string]any, error) {
	books, err := s.bookRepo.SearchBooks(ctx, nil, nil, "", "", "", "", "", nil, 100)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "failed to query books for kobo sync")
	}

	claims = resolveClaims(claims)
	var syncItems []map[string]any
	for _, b := range books {
		attrs := map[string]any{"library_id": b.LibraryID}
		if !s.bookService.CanReadBook(ctx, b, claims) || !s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermKoboSync, attrs) {
			continue
		}
		syncItems = append(syncItems, map[string]any{
			"Id": b.ID,
			"NewEntitlement": map[string]any{
				"Id": b.ID,
				"BookEntitlement": map[string]any{
					"Id": b.ID,
					"Metadata": map[string]any{
						"Title":       b.Title,
						"Description": b.Description,
					},
				},
			},
		})
	}

	return map[string]any{
		"SyncToken": "sync-token-1",
		"Items":     syncItems,
	}, nil
}

func (s *koboService) GetBookKePubStream(ctx context.Context, bookID string, claims *response.JWTClaims, out io.Writer) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil || !s.bookService.CanReadBook(ctx, book, claims) || !s.bookService.CanDownloadBook(ctx, book, claims) {
		return apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}
	resolved := resolveClaims(claims)
	if !s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermKoboSync, map[string]any{"library_id": book.LibraryID}) {
		return apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}
	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}

	targetFile := files[0]
	filePath := targetFile.Path

	data, err := os.ReadFile(filePath)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to read book file")
	}

	reader := bytes.NewReader(data)
	if err := kepub.ConvertEPUBToKePub(reader, int64(len(data)), out); err != nil {
		_, _ = out.Write(data)
	}

	return nil
}

func (s *koboService) SyncState(ctx context.Context, userID int64, stateData map[string]any) error {
	return nil
}
