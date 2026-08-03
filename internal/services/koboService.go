package services

import (
	"context"
	"io"
	"os"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
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
	SyncState(ctx context.Context, userID string, stateData map[string]any, claims *response.JWTClaims) error
}

type koboService struct {
	bookRepo       repositories.BookDBRepository
	diskRepo       repositories.BookFileRepository
	bookService    BookService
	featureService FeatureService
	permissions    PermissionCache
}

func NewKoboService(bookRepo repositories.BookDBRepository, diskRepo repositories.BookFileRepository, bookService BookService, featureService FeatureService, permissions PermissionCache) KoboService {
	return &koboService{
		bookRepo:       bookRepo,
		diskRepo:       diskRepo,
		bookService:    bookService,
		featureService: featureService,
		permissions:    permissions,
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
	books, err := s.bookRepo.SearchBooks(ctx, nil, nil, "", "", "", "", "", nil, "", 100)
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

	file, err := os.Open(filePath)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to read book file")
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to read book file")
	}
	if err := kepub.ConvertEPUBToKePub(file, info.Size(), out); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to convert book file")
	}

	return nil
}

func (s *koboService) SyncState(ctx context.Context, userID string, stateData map[string]any, claims *response.JWTClaims) error {
	if userID == "" {
		return apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}
	if stateData == nil {
		return nil
	}
	items, ok := stateData["Bookmark"].([]any)
	if !ok {
		if stateArr, isArr := stateData["State"].([]any); isArr {
			items = stateArr
		}
	}
	for _, itemRaw := range items {
		itemMap, isMap := itemRaw.(map[string]any)
		if !isMap {
			continue
		}
		bookID, _ := itemMap["EntitlementId"].(string)
		if bookID == "" {
			bookID, _ = itemMap["VolumeId"].(string)
		}
		if bookID == "" {
			bookID, _ = itemMap["BookId"].(string)
		}
		if bookID == "" {
			continue
		}

		book, err := s.bookRepo.GetBook(ctx, bookID)
		if err != nil || book == nil {
			continue
		}
		resolved := resolveClaims(claims)
		if !s.bookService.CanReadBook(ctx, book, resolved) || !s.permissions.CanRoles(resolved.RoleIDs, resolved.Roles, constants.PermKoboSync, map[string]any{"library_id": book.LibraryID}) {
			continue
		}

		progressPercent := float64(0)
		if pct, ok := itemMap["ProgressPercent"].(float64); ok {
			progressPercent = pct
		} else if pct, ok := itemMap["PercentRead"].(float64); ok {
			progressPercent = pct
		}
		cfi := ""
		if loc, ok := itemMap["LocationCfi"].(string); ok {
			cfi = loc
		}

		locType := "kobo"
		input := models.ReadingActivityInput{
			UserID:          userID,
			BookID:          bookID,
			ChapterID:       bookID,
			ChapterTitle:    "Kobo Sync",
			ProgressPercent: &progressPercent,
			LocationCfi:     &cfi,
			LocationType:    &locType,
			EventType:       "kobo_sync",
		}
		_, _ = s.featureService.RecordReadingActivity(ctx, input, claims)
	}
	return nil
}
