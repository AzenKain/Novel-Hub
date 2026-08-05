package services

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/kepub"
	"novelhub/pkg/kobo"

	"github.com/google/uuid"
)

type KoboService interface {
	GetInitialization(endpointURL string) *response.KoboInitResponse
	GetUserProfile(claims *response.JWTClaims) *response.KoboUserProfileResponse
	AuthDevice(ctx context.Context, userKey string) (*response.KoboAuthDeviceResponse, error)
	GetSyncList(ctx context.Context, dto request.KoboSyncDto, claims *response.JWTClaims) (*response.KoboSyncResponse, error)
	GetBookMetadata(ctx context.Context, bookUUID, endpointURL string, claims *response.JWTClaims) ([]kobo.BookMetadata, error)
	GetReadingState(ctx context.Context, userID, bookUUID string, claims *response.JWTClaims) ([]kobo.ReadingState, error)
	PutReadingState(ctx context.Context, userID, bookUUID string, dto request.PutKoboStateDto, claims *response.JWTClaims) (*kobo.PutStateResponse, error)
	GetCoverPath(ctx context.Context, bookUUID string, claims *response.JWTClaims) (string, error)
	GetBookKePubStream(ctx context.Context, bookID string, claims *response.JWTClaims, out io.Writer) error
}

type koboService struct {
	bookRepo       repositories.BookDBRepository
	diskRepo       repositories.BookFileRepository
	koboRepo       repositories.KoboRepository
	bookService    BookService
	featureService FeatureService
	permissions    PermissionCache
}

func NewKoboService(
	bookRepo repositories.BookDBRepository,
	diskRepo repositories.BookFileRepository,
	koboRepo repositories.KoboRepository,
	bookService BookService,
	featureService FeatureService,
	permissions PermissionCache,
) KoboService {
	return &koboService{
		bookRepo:       bookRepo,
		diskRepo:       diskRepo,
		koboRepo:       koboRepo,
		bookService:    bookService,
		featureService: featureService,
		permissions:    permissions,
	}
}

func (s *koboService) GetInitialization(endpointURL string) *response.KoboInitResponse {
	return &response.KoboInitResponse{Resources: kobo.Resources(endpointURL)}
}

func (s *koboService) GetUserProfile(claims *response.JWTClaims) *response.KoboUserProfileResponse {
	resolved := resolveClaims(claims)
	return &response.KoboUserProfileResponse{
		User: response.KoboProfileUserResponse{
			UserKey: resolved.UId,
			UserID:  resolved.UId,
			IsGuest: false,
		},
	}
}

func (s *koboService) AuthDevice(ctx context.Context, userKey string) (*response.KoboAuthDeviceResponse, error) {
	accessToken, err := kobo.RandomToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := kobo.RandomToken()
	if err != nil {
		return nil, err
	}
	return &response.KoboAuthDeviceResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		TrackingID:   uuid.NewString(),
		UserKey:      userKey,
	}, nil
}

func (s *koboService) GetSyncList(ctx context.Context, dto request.KoboSyncDto, claims *response.JWTClaims) (*response.KoboSyncResponse, error) {
	if strings.TrimSpace(dto.UserID) == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User authentication required")
	}

	token := kobo.ParseSyncToken(dto.SyncToken)

	syncedCount, err := s.koboRepo.CountSyncedBooks(ctx, dto.UserID)
	if err != nil {
		return nil, err
	}
	if syncedCount == 0 {
		token.BooksLastModified = time.Time{}
		token.BooksLastCreated = time.Time{}
		token.ReadingStateLastModified = time.Time{}
	}

	synced, err := s.koboRepo.SyncedBookIDs(ctx, dto.UserID)
	if err != nil {
		return nil, err
	}
	syncedSet := make(map[string]struct{}, len(synced))
	for _, id := range synced {
		syncedSet[id] = struct{}{}
	}

	resolved := resolveClaims(claims)
	newToken := token
	items := make([]response.KoboSyncItemResponse, 0, kobo.SyncItemLimit)
	remaining := false

	var cursor *time.Time
	cursorID := ""

	for !remaining {
		books, err := s.bookRepo.SearchBooks(ctx, nil, nil, "", "", "", "", "", cursor, cursorID, constants.MaxPaginationLimit)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "failed to query books for kobo sync")
		}
		if len(books) == 0 {
			break
		}

		for _, book := range books {
			if len(items) >= kobo.SyncItemLimit {
				remaining = true
				break
			}

			created := book.CreatedAt
			cursor, cursorID = &created, book.ID

			if _, already := syncedSet[book.ID]; already {
				continue
			}
			if !s.canSync(ctx, book, resolved) {
				continue
			}

			info, err := s.bookInfo(ctx, book, dto.EndpointURL, resolved)
			if err != nil {
				return nil, err
			}
			if len(info.Downloads) == 0 {
				continue
			}

			entitlement := &response.KoboEntitlementResponse{
				BookEntitlement: kobo.NewBookEntitlement(info),
				BookMetadata:    kobo.NewBookMetadata(info),
			}

			state, hasState, err := s.readingState(ctx, dto.UserID, book)
			if err != nil {
				return nil, err
			}
			if hasState {
				entitlement.ReadingState = &state
				if book.UpdatedAt.After(newToken.ReadingStateLastModified) {
					newToken.ReadingStateLastModified = book.UpdatedAt
				}
			}

			item := response.KoboSyncItemResponse{ChangedEntitlement: entitlement}
			if book.CreatedAt.After(token.BooksLastCreated) {
				item = response.KoboSyncItemResponse{NewEntitlement: entitlement}
			}
			items = append(items, item)

			if book.UpdatedAt.After(newToken.BooksLastModified) {
				newToken.BooksLastModified = book.UpdatedAt
			}
			if book.CreatedAt.After(newToken.BooksLastCreated) {
				newToken.BooksLastCreated = book.CreatedAt
			}

			if err := s.koboRepo.MarkBookSynced(ctx, dto.UserID, book.ID); err != nil {
				return nil, err
			}
		}

		if int64(len(books)) < constants.MaxPaginationLimit {
			break
		}
	}

	encoded, err := newToken.Encode()
	if err != nil {
		return nil, err
	}
	return &response.KoboSyncResponse{Items: items, SyncToken: encoded, Continue: remaining}, nil
}

func (s *koboService) GetBookMetadata(ctx context.Context, bookUUID, endpointURL string, claims *response.JWTClaims) ([]kobo.BookMetadata, error) {
	book, resolved, err := s.accessibleBook(ctx, bookUUID, claims)
	if err != nil {
		return nil, err
	}
	info, err := s.bookInfo(ctx, book, endpointURL, resolved)
	if err != nil {
		return nil, err
	}
	return []kobo.BookMetadata{kobo.NewBookMetadata(info)}, nil
}

func (s *koboService) GetReadingState(ctx context.Context, userID, bookUUID string, claims *response.JWTClaims) ([]kobo.ReadingState, error) {
	book, _, err := s.accessibleBook(ctx, bookUUID, claims)
	if err != nil {
		return nil, err
	}
	state, err := s.readingStateOrEmpty(ctx, userID, book)
	if err != nil {
		return nil, err
	}
	return []kobo.ReadingState{state}, nil
}

func (s *koboService) PutReadingState(ctx context.Context, userID, bookUUID string, dto request.PutKoboStateDto, claims *response.JWTClaims) (*kobo.PutStateResponse, error) {
	book, resolved, err := s.accessibleBook(ctx, bookUUID, claims)
	if err != nil {
		return nil, err
	}
	if len(dto.ReadingStates) == 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "ReadingStates is required")
	}

	incoming := dto.ReadingStates[0]
	result := kobo.PutStateResult{EntitlementID: bookUUID}
	success := &kobo.PutStateSubResult{Result: "Success"}

	if bookmark := incoming.CurrentBookmark; bookmark != nil {
		progress := float64(0)
		if bookmark.ProgressPercent != nil {
			progress = *bookmark.ProgressPercent
		} else if bookmark.ContentSourceProgressPercent != nil {
			progress = *bookmark.ContentSourceProgressPercent
		}

		locationValue, locationType := "", "KoboSpan"
		if loc := bookmark.Location; loc != nil {
			locationValue = loc.Value
			if loc.Type != "" {
				locationType = loc.Type
			}
		}

		if _, err := s.featureService.RecordReadingActivity(ctx, models.ReadingActivityInput{
			UserID:          userID,
			BookID:          book.ID,
			ChapterID:       book.ID,
			ChapterTitle:    "Kobo",
			ProgressPercent: &progress,
			LocationCfi:     &locationValue,
			LocationType:    &locationType,
			EventType:       "kobo_sync",
		}, resolved); err != nil {
			return nil, err
		}
		result.CurrentBookmarkResult = success
	}

	if incoming.Statistics != nil {
		result.StatisticsResult = success
	}
	if incoming.StatusInfo != nil {
		result.StatusInfoResult = success
	}

	state, err := s.readingStateOrEmpty(ctx, userID, book)
	if err != nil {
		return nil, err
	}
	result.LastModified = state.LastModified
	result.PriorityTimestamp = state.PriorityTimestamp

	return &kobo.PutStateResponse{RequestResult: "Success", UpdateResults: []kobo.PutStateResult{result}}, nil
}

func (s *koboService) GetCoverPath(ctx context.Context, bookUUID string, claims *response.JWTClaims) (string, error) {
	book, _, err := s.accessibleBook(ctx, bookUUID, claims)
	if err != nil {
		return "", err
	}
	if book.CoverURL == nil || strings.TrimSpace(*book.CoverURL) == "" {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}
	path, err := s.diskRepo.ResolveCoverPath(ctx, book.ID, *book.CoverURL)
	if err != nil {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}
	return path, nil
}

func (s *koboService) GetBookKePubStream(ctx context.Context, bookID string, claims *response.JWTClaims, out io.Writer) error {
	book, resolved, err := s.accessibleBook(ctx, bookID, claims)
	if err != nil {
		return err
	}
	if !s.bookService.CanDownloadBook(ctx, book, resolved) {
		return apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}
	files, err := s.bookRepo.GetFilesByBookId(ctx, bookID)
	if err != nil || len(files) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}

	target := pickKoboFile(files)
	if target == nil {
		return apperrors.New(apperrors.ErrNotFound, "Book file not found")
	}

	file, err := os.Open(target.Path)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to read book file")
	}
	defer file.Close()

	if strings.EqualFold(target.Format, "KEPUB") {
		if _, err := io.Copy(out, file); err != nil {
			return apperrors.New(apperrors.ErrInternalError, "Failed to read book file")
		}
		return nil
	}

	info, err := file.Stat()
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to read book file")
	}
	if err := kepub.ConvertEPUBToKePub(file, info.Size(), out); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to convert book file")
	}
	return nil
}

func (s *koboService) accessibleBook(ctx context.Context, bookID string, claims *response.JWTClaims) (*models.BookEntity, *response.JWTClaims, error) {
	resolved := resolveClaims(claims)
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil {
		return nil, nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	if !s.canSync(ctx, book, resolved) {
		return nil, nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	return book, resolved, nil
}

func (s *koboService) canSync(ctx context.Context, book *models.BookEntity, claims *response.JWTClaims) bool {
	if !s.bookService.CanReadBook(ctx, book, claims) {
		return false
	}
	return s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermKoboSync, map[string]any{"library_id": book.LibraryID})
}

func (s *koboService) bookInfo(ctx context.Context, book *models.BookEntity, endpointURL string, claims *response.JWTClaims) (kobo.BookInfo, error) {
	info := kobo.BookInfo{
		UUID:         book.ID,
		Title:        book.Title,
		Description:  book.Description,
		Created:      book.CreatedAt,
		LastModified: book.UpdatedAt,
		PublishedAt:  book.CreatedAt,
	}
	if book.AuthorName != nil && strings.TrimSpace(*book.AuthorName) != "" {
		info.Authors = []string{*book.AuthorName}
	}

	meta := parseBookMetadataMap(book.MetadataJSON)
	if publisher, ok := meta["publisher"].(string); ok {
		info.Publisher = publisher
	}
	if language, ok := meta["language"].(string); ok {
		info.Language = language
	}
	if series, ok := meta["series"].(string); ok {
		info.SeriesName = series
	}
	if index, ok := meta["series_index"].(float64); ok {
		info.SeriesIndex = index
	}
	if published, ok := meta["published_at"].(string); ok {
		if ts, err := time.Parse(time.RFC3339, published); err == nil {
			info.PublishedAt = ts
		}
	}
	if len(info.Authors) == 0 {
		if creator, ok := meta["creator"].(string); ok && strings.TrimSpace(creator) != "" {
			info.Authors = []string{creator}
		}
	}

	if !s.bookService.CanDownloadBook(ctx, book, claims) {
		return info, nil
	}

	files, err := s.bookRepo.GetFilesByBookId(ctx, book.ID)
	if err != nil {
		return info, nil
	}
	for _, file := range files {
		formats, ok := kobo.KoboFormats[strings.ToUpper(file.Format)]
		if !ok {
			continue
		}
		for _, koboFormat := range formats {
			info.Downloads = append(info.Downloads, kobo.BookDownloadURL(endpointURL, book.ID, file.Format, koboFormat, file.SizeBytes))
		}
	}
	return info, nil
}

func (s *koboService) readingState(ctx context.Context, userID string, book *models.BookEntity) (kobo.ReadingState, bool, error) {
	progress, err := s.featureService.GetReadingProgress(ctx, userID, book.ID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return kobo.ReadingState{}, false, nil
		}
		return kobo.ReadingState{}, false, err
	}
	return buildReadingState(book, progress), true, nil
}

func (s *koboService) readingStateOrEmpty(ctx context.Context, userID string, book *models.BookEntity) (kobo.ReadingState, error) {
	progress, err := s.featureService.GetReadingProgress(ctx, userID, book.ID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return kobo.NewReadingState(kobo.ReadingStateInput{
				BookUUID:     book.ID,
				BookCreated:  book.CreatedAt,
				LastModified: book.UpdatedAt,
			}), nil
		}
		return kobo.ReadingState{}, err
	}
	return buildReadingState(book, progress), nil
}

func buildReadingState(book *models.BookEntity, progress *response.ReadingProgressResponse) kobo.ReadingState {
	in := kobo.ReadingStateInput{
		BookUUID:     book.ID,
		BookCreated:  book.CreatedAt,
		LastModified: book.UpdatedAt,
	}
	if progress == nil {
		return kobo.NewReadingState(in)
	}
	if progress.ProgressPercent != nil {
		in.ProgressPercent = *progress.ProgressPercent
	}
	if progress.LocationCfi != nil {
		in.LocationValue = *progress.LocationCfi
	}
	if progress.LocationType != nil {
		in.LocationType = *progress.LocationType
	}
	in.OpenedCount = progress.OpenedCount
	if progress.LastOpenedAt != nil {
		in.LastOpenedAt = *progress.LastOpenedAt
	}
	if progress.UpdatedAt != nil {
		in.LastModified = *progress.UpdatedAt
	}
	return kobo.NewReadingState(in)
}

func pickKoboFile(files []*models.BookFileEntity) *models.BookFileEntity {
	ranked := make([]*models.BookFileEntity, 0, len(files))
	for _, file := range files {
		if _, ok := kobo.KoboFormats[strings.ToUpper(file.Format)]; ok {
			ranked = append(ranked, file)
		}
	}
	if len(ranked) == 0 {
		return nil
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return strings.EqualFold(ranked[i].Format, "KEPUB") && !strings.EqualFold(ranked[j].Format, "KEPUB")
	})
	return ranked[0]
}
