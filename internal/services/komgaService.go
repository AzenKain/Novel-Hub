package services

import (
	"context"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
)

type KomgaService interface {
	ListSeries(ctx context.Context, search string, page, size int64, claims *response.JWTClaims) (*response.KomgaPageWrapper[response.KomgaSeries], error)
	GetSeries(ctx context.Context, seriesID string, claims *response.JWTClaims) (*response.KomgaSeries, error)
	ListSeriesBooks(ctx context.Context, seriesID string, claims *response.JWTClaims) ([]response.KomgaBook, error)
	GetBook(ctx context.Context, bookID string, claims *response.JWTClaims) (*response.KomgaBook, error)
	ListBookPages(ctx context.Context, bookID string, zeroBased bool, claims *response.JWTClaims) ([]response.KomgaPage, error)
	GetBookPage(ctx context.Context, bookID string, pageNumber int, zeroBased bool, claims *response.JWTClaims) (*models.ReaderAssetEntity, error)
	ListLibraries(ctx context.Context, claims *response.JWTClaims) ([]response.KomgaLibrary, error)
	SeriesProgress(ctx context.Context, seriesID string, claims *response.JWTClaims) (*response.KomgaReadProgressV2, error)
	SeriesProgressV1(ctx context.Context, seriesID string, claims *response.JWTClaims) (*response.KomgaTachiyomiReadProgressV1, error)
	MarkSeriesReadUpTo(ctx context.Context, seriesID string, lastNumberSort float64, claims *response.JWTClaims) error
	GetBookReadProgress(ctx context.Context, bookID string, claims *response.JWTClaims) (*response.KomgaBookReadProgress, error)
	UpdateBookReadProgress(ctx context.Context, bookID string, page int, completed bool, claims *response.JWTClaims) error
	DeleteBookReadProgress(ctx context.Context, bookID string, claims *response.JWTClaims) error
	ListReadLists(ctx context.Context, page, size int64, claims *response.JWTClaims) (*response.KomgaPageWrapper[response.KomgaReadList], error)
	GetReadList(ctx context.Context, readListID string, claims *response.JWTClaims) (*response.KomgaReadList, error)
	GetReadListBooks(ctx context.Context, readListID string, claims *response.JWTClaims) ([]response.KomgaBook, error)
	GetUserMe(ctx context.Context, claims *response.JWTClaims) (*response.KomgaUserResponse, error)
	BookCoverPath(ctx context.Context, bookID string, claims *response.JWTClaims) (string, error)
	SeriesCoverPath(ctx context.Context, seriesID string, claims *response.JWTClaims) (string, error)
}

type komgaService struct {
	repo        repositories.KomgaRepository
	bookRepo    repositories.BookDBRepository
	diskRepo    repositories.BookFileRepository
	readLists   repositories.ReadListRepository
	userRepo    repositories.UserRepository
	bookService BookService
	libraries   LibraryService
	features    FeatureService
	permissions PermissionCache
	cache       cache.Cache
}

func NewKomgaService(
	repo repositories.KomgaRepository,
	bookRepo repositories.BookDBRepository,
	diskRepo repositories.BookFileRepository,
	readLists repositories.ReadListRepository,
	userRepo repositories.UserRepository,
	bookService BookService,
	libraries LibraryService,
	features FeatureService,
	permissions PermissionCache,
	ramCache cache.Cache,
) KomgaService {
	return &komgaService{
		repo:        repo,
		bookRepo:    bookRepo,
		diskRepo:    diskRepo,
		readLists:   readLists,
		userRepo:    userRepo,
		bookService: bookService,
		libraries:   libraries,
		features:    features,
		permissions: permissions,
		cache:       ramCache,
	}
}

const komgaTimeFormat = "2006-01-02T15:04:05"

func komgaTime(t *time.Time) string {
	if t == nil {
		return time.Unix(0, 0).UTC().Format(komgaTimeFormat)
	}
	return t.UTC().Format(komgaTimeFormat)
}

func komgaTimePtr(t *time.Time) *string {
	formatted := komgaTime(t)
	return &formatted
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGT"[exp])
}

func (s *komgaService) syncableLibraryIDs(ctx context.Context, claims *response.JWTClaims) ([]string, error) {
	readable, err := s.libraries.ReadableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}
	allowed := make([]string, 0, len(readable))
	for _, id := range readable {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermKomgaSync, map[string]any{"library_id": id}) {
			allowed = append(allowed, id)
		}
	}
	return allowed, nil
}

func (s *komgaService) seriesDTO(entity *models.KomgaSeriesEntity, progress *models.KomgaSeriesProgressEntity) response.KomgaSeries {
	dto := response.KomgaSeries{
		ID:               entity.ID,
		LibraryID:        entity.LibraryID,
		Name:             entity.Name,
		Created:          &entity.LastModified,
		LastModified:     &entity.LastModified,
		FileLastModified: entity.LastModified,
		BooksCount:       int(entity.BookCount),
		Metadata: response.KomgaSeriesMetadata{
			Status:           "ONGOING",
			Created:          &entity.LastModified,
			LastModified:     &entity.LastModified,
			Title:            entity.Name,
			TitleSort:        entity.Name,
			Summary:          "",
			ReadingDirection: "LEFT_TO_RIGHT",
			Publisher:        "",
			Language:         "",
			Genres:           []string{},
			Tags:             []string{},
		},
		BooksMetadata: response.KomgaBookMetadataAggregation{
			Authors:       []response.KomgaAuthor{},
			Tags:          []string{},
			Summary:       "",
			SummaryNumber: "",
			Created:       entity.LastModified,
			LastModified:  entity.LastModified,
		},
	}
	if progress != nil {
		dto.BooksReadCount = int(progress.BooksReadCount)
		dto.BooksInProgress = int(progress.BooksInProgressCount)
		dto.BooksUnreadCount = int(progress.BooksUnreadCount())
	} else {
		dto.BooksUnreadCount = int(entity.BookCount)
	}
	return dto
}

func (s *komgaService) ListSeries(ctx context.Context, search string, page, size int64, claims *response.JWTClaims) (*response.KomgaPageWrapper[response.KomgaSeries], error) {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}
	if size <= 0 || size > constants.MaxPaginationLimit {
		size = 20
	}
	if page < 0 {
		page = 0
	}

	search = strings.TrimSpace(search)

	empty := &response.KomgaPageWrapper[response.KomgaSeries]{
		Content: []response.KomgaSeries{}, Empty: true, First: page == 0, Last: true,
		Number: page, Size: size,
	}
	if len(libraryIDs) == 0 {
		return empty, nil
	}

	total, err := s.repo.CountSeries(ctx, libraryIDs, search)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListSeries(ctx, libraryIDs, search, size, page*size)
	if err != nil {
		return nil, err
	}

	seriesIDs := make([]string, len(rows))
	for i, row := range rows {
		seriesIDs[i] = row.ID
	}
	progress, err := s.repo.SeriesProgressByIDs(ctx, claims.UId, seriesIDs, libraryIDs)
	if err != nil {
		return nil, err
	}

	content := make([]response.KomgaSeries, 0, len(rows))
	for _, row := range rows {
		content = append(content, s.seriesDTO(row, progress[row.ID]))
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = int64(math.Ceil(float64(total) / float64(size)))
	}
	return &response.KomgaPageWrapper[response.KomgaSeries]{
		Content:          content,
		Empty:            len(content) == 0,
		First:            page == 0,
		Last:             page >= totalPages-1,
		Number:           page,
		NumberOfElements: int64(len(content)),
		Size:             size,
		TotalElements:    total,
		TotalPages:       totalPages,
	}, nil
}

func (s *komgaService) GetSeries(ctx context.Context, seriesID string, claims *response.JWTClaims) (*response.KomgaSeries, error) {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}
	if len(libraryIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "Series not found")
	}
	entity, err := s.oneSeries(ctx, seriesID, libraryIDs)
	if err != nil {
		return nil, err
	}
	progress, _ := s.repo.SeriesProgress(ctx, claims.UId, seriesID, libraryIDs)
	dto := s.seriesDTO(entity, progress)
	return &dto, nil
}

func (s *komgaService) oneSeries(ctx context.Context, seriesID string, libraryIDs []string) (*models.KomgaSeriesEntity, error) {
	rows, err := s.repo.GetSeriesByIDs(ctx, []string{seriesID}, libraryIDs)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "Series not found")
	}
	return rows[0], nil
}

func (s *komgaService) bookDTO(ctx context.Context, book *models.KomgaSeriesBookEntity, seriesID, seriesTitle string) response.KomgaBook {
	pageCount := 0
	if images, err := s.pageNames(ctx, book.ID); err == nil {
		pageCount = len(images)
	}

	var sizeBytes int64
	if files, err := s.bookRepo.GetFilesByBookId(ctx, book.ID); err == nil && len(files) > 0 {
		sizeBytes = files[0].SizeBytes
	}

	number := strconv.FormatFloat(book.NumberSort, 'f', -1, 64)
	if book.SeriesIndex != nil && strings.TrimSpace(*book.SeriesIndex) != "" {
		number = *book.SeriesIndex
	}
	summary := ""
	if book.Description != nil {
		summary = *book.Description
	}

	return response.KomgaBook{
		ID:               book.ID,
		SeriesID:         seriesID,
		SeriesTitle:      seriesTitle,
		Name:             book.Title,
		Number:           book.NumberSort,
		Created:          komgaTimePtr(book.CreatedAt),
		LastModified:     komgaTimePtr(book.UpdatedAt),
		FileLastModified: komgaTime(book.UpdatedAt),
		SizeBytes:        sizeBytes,
		Size:             humanSize(sizeBytes),
		Media: response.KomgaMedia{
			Status:       "READY",
			MediaType:    "application/zip",
			PagesCount:   pageCount,
			MediaProfile: "DIVINA",
		},
		Metadata: response.KomgaBookMetadata{
			Title:      book.Title,
			Summary:    summary,
			Number:     number,
			NumberSort: book.NumberSort,
			Authors:    []response.KomgaAuthor{},
			Tags:       []string{},
		},
	}
}

func (s *komgaService) ListSeriesBooks(ctx context.Context, seriesID string, claims *response.JWTClaims) ([]response.KomgaBook, error) {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}
	if len(libraryIDs) == 0 {
		return []response.KomgaBook{}, nil
	}
	series, err := s.oneSeries(ctx, seriesID, libraryIDs)
	if err != nil {
		return nil, err
	}
	books, err := s.repo.ListSeriesBooks(ctx, seriesID, libraryIDs)
	if err != nil {
		return nil, err
	}
	result := make([]response.KomgaBook, 0, len(books))
	for _, book := range books {
		result = append(result, s.bookDTO(ctx, book, seriesID, series.Name))
	}
	return result, nil
}

func (s *komgaService) accessibleBook(ctx context.Context, bookID string, claims *response.JWTClaims) (*models.KomgaSeriesBookEntity, *models.KomgaBookSeriesRefEntity, error) {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, nil, err
	}
	ref, err := s.repo.GetBookSeries(ctx, bookID)
	if err != nil {
		return nil, nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
	}
	books, err := s.repo.ListSeriesBooks(ctx, ref.SeriesID, libraryIDs)
	if err != nil {
		return nil, nil, err
	}
	for _, book := range books {
		if book.ID == bookID {
			return book, ref, nil
		}
	}
	return nil, nil, apperrors.New(apperrors.ErrNotFound, "Book not found")
}

func (s *komgaService) GetBook(ctx context.Context, bookID string, claims *response.JWTClaims) (*response.KomgaBook, error) {
	book, ref, err := s.accessibleBook(ctx, bookID, claims)
	if err != nil {
		return nil, err
	}
	dto := s.bookDTO(ctx, book, ref.SeriesID, ref.SeriesName)
	return &dto, nil
}

func (s *komgaService) ListBookPages(ctx context.Context, bookID string, zeroBased bool, claims *response.JWTClaims) ([]response.KomgaPage, error) {
	if _, _, err := s.accessibleBook(ctx, bookID, claims); err != nil {
		return nil, err
	}
	images, err := s.pageNames(ctx, bookID)
	if err != nil {
		return nil, err
	}
	pages := make([]response.KomgaPage, 0, len(images))
	for i, name := range images {
		num := i + 1
		if zeroBased {
			num = i
		}
		pages = append(pages, response.KomgaPage{
			Number:    num,
			FileName:  path.Base(name),
			MediaType: komgaImageMediaType(name),
		})
	}
	return pages, nil
}

func komgaImageMediaType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}

// Page numbers are 1-based by default, or 0-based if requested via zeroBased=true.
func (s *komgaService) GetBookPage(ctx context.Context, bookID string, pageNumber int, zeroBased bool, claims *response.JWTClaims) (*models.ReaderAssetEntity, error) {
	if _, _, err := s.accessibleBook(ctx, bookID, claims); err != nil {
		return nil, err
	}
	images, err := s.pageNames(ctx, bookID)
	if err != nil {
		return nil, err
	}
	idx := pageNumber - 1
	if zeroBased {
		idx = pageNumber
	}
	if idx < 0 || idx >= len(images) {
		return nil, apperrors.New(apperrors.ErrNotFound, "Page not found")
	}
	return s.bookService.GetAsset(ctx, bookID, images[idx], "")
}

func (s *komgaService) ListLibraries(ctx context.Context, claims *response.JWTClaims) ([]response.KomgaLibrary, error) {
	libs, err := s.libraries.ListLibraries(ctx, claims)
	if err != nil {
		return nil, err
	}
	result := make([]response.KomgaLibrary, 0, len(libs))
	for _, lib := range libs {
		if s.permissions.CanRoles(claims.RoleIDs, claims.Roles, constants.PermKomgaSync, map[string]any{"library_id": lib.ID}) {
			result = append(result, response.KomgaLibrary{ID: lib.ID, Name: lib.Name})
		}
	}
	return result, nil
}

func (s *komgaService) SeriesProgress(ctx context.Context, seriesID string, claims *response.JWTClaims) (*response.KomgaReadProgressV2, error) {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}

	if len(libraryIDs) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "Series not found")
	}

	progress, err := s.repo.SeriesProgress(ctx, claims.UId, seriesID, libraryIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Series not found")
	}

	if progress.BooksCount == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "Series not found")
	}
	return &response.KomgaReadProgressV2{
		BooksCount:                  int(progress.BooksCount),
		BooksReadCount:              int(progress.BooksReadCount),
		BooksUnreadCount:            int(progress.BooksUnreadCount()),
		BooksInProgressCount:        int(progress.BooksInProgressCount),
		LastReadContinuousNumberSrt: progress.LastReadNumberSort,
		MaxNumberSort:               progress.MaxNumberSort,
	}, nil
}

func (s *komgaService) MarkSeriesReadUpTo(ctx context.Context, seriesID string, lastNumberSort float64, claims *response.JWTClaims) error {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return err
	}
	if len(libraryIDs) == 0 {
		return apperrors.New(apperrors.ErrNotFound, "Series not found")
	}
	books, err := s.repo.ListSeriesBooks(ctx, seriesID, libraryIDs)
	if err != nil {
		return err
	}
	full := 100.0
	for _, book := range books {
		if book.NumberSort > lastNumberSort {
			continue
		}
		chapterID, chapterTitle := s.firstChapter(ctx, book.ID)
		if chapterID == "" {
			continue
		}
		if _, err := s.features.RecordReadingActivity(ctx, models.ReadingActivityInput{
			UserID:          claims.UId,
			BookID:          book.ID,
			ChapterID:       chapterID,
			ChapterTitle:    chapterTitle,
			ProgressPercent: &full,
			EventType:       "progress",
		}, claims); err != nil {
			return err
		}
	}
	_ = s.repo.InvalidateSeriesProgress(ctx, claims.UId, seriesID)
	return nil
}

func (s *komgaService) firstChapter(ctx context.Context, bookID string) (string, string) {
	bootstrap, err := s.bookService.GetReaderBootstrap(ctx, bookID, "")
	if err != nil || bootstrap == nil || len(bootstrap.Chapters) == 0 {
		return "", ""
	}
	last := bootstrap.Chapters[len(bootstrap.Chapters)-1]
	return last.ID, last.Title
}

func (s *komgaService) BookCoverPath(ctx context.Context, bookID string, claims *response.JWTClaims) (string, error) {
	if _, _, err := s.accessibleBook(ctx, bookID, claims); err != nil {
		return "", err
	}
	return s.coverPath(ctx, bookID)
}

func (s *komgaService) SeriesCoverPath(ctx context.Context, seriesID string, claims *response.JWTClaims) (string, error) {
	libraryIDs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return "", err
	}
	books, err := s.repo.ListSeriesBooks(ctx, seriesID, libraryIDs)
	if err != nil || len(books) == 0 {
		return "", apperrors.New(apperrors.ErrNotFound, "Series cover not found")
	}
	for _, book := range books {
		if book.CoverURL != nil && strings.TrimSpace(*book.CoverURL) != "" {
			return s.coverPath(ctx, book.ID)
		}
	}
	return "", apperrors.New(apperrors.ErrNotFound, "Series cover not found")
}

func (s *komgaService) coverPath(ctx context.Context, bookID string) (string, error) {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil || book == nil || book.CoverURL == nil || strings.TrimSpace(*book.CoverURL) == "" {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}
	resolved, err := s.diskRepo.ResolveCoverPath(ctx, book.ID, *book.CoverURL)
	if err != nil {
		return "", apperrors.New(apperrors.ErrNotFound, "Cover not found")
	}
	return resolved, nil
}

// Reading a page reopens the archive to map its 1-based number onto a name. That mapping never
// changes for a file, but re-listing costs 38ms on a 248MB CBR — 56% of the request, since RAR
// has no usable central directory (pkg/bookparser/comic/realcbr_bench_test.go). Cache the names,
// never the bytes.
func (s *komgaService) pageNames(ctx context.Context, bookID string) ([]string, error) {
	file, err := s.bookService.GetBookFile(ctx, bookID, "")
	if err != nil {
		return nil, err
	}
	key := cache.BuildKey("komga", "pages", file.ID, file.ModTime)

	if s.cache != nil {
		var cached []string
		if err := s.cache.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}
	images, err := s.bookService.ListImages(ctx, bookID, "")
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, key, images, constants.NormalCacheDuration)
	}
	return images, nil
}

func (s *komgaService) GetUserMe(ctx context.Context, claims *response.JWTClaims) (*response.KomgaUserResponse, error) {
	roles := []string{"FILE_DOWNLOAD", "PAGE_STREAMING"}
	for _, r := range claims.Roles {
		if strings.EqualFold(string(r), "ADMIN") || strings.EqualFold(string(r), "SUPERADMIN") {
			roles = append(roles, "ADMIN")
			break
		}
	}
	libs, err := s.syncableLibraryIDs(ctx, claims)
	if err != nil {
		return nil, err
	}
	email := claims.Subject
	if s.userRepo != nil {
		if u, err := s.userRepo.GetByID(ctx, claims.UId); err == nil && u != nil && u.Email != "" {
			email = u.Email
		}
	}
	if email == "" {
		email = claims.UId + "@novelhub.local"
	}
	return &response.KomgaUserResponse{
		ID:    claims.UId,
		Email: email,
		Roles: roles,
		SharedLibraries: response.KomgaSharedLibraries{
			All:        len(libs) > 0,
			LibraryIDs: libs,
		},
		LabelsAllow:    []string{},
		LabelsExclude:  []string{},
		AgeRestriction: nil,
	}, nil
}

func (s *komgaService) SeriesProgressV1(ctx context.Context, seriesID string, claims *response.JWTClaims) (*response.KomgaTachiyomiReadProgressV1, error) {
	v2, err := s.SeriesProgress(ctx, seriesID, claims)
	if err != nil {
		return nil, err
	}
	return &response.KomgaTachiyomiReadProgressV1{
		BooksCount:             v2.BooksCount,
		Logged:                 true,
		LastBookNumberSortRead: v2.LastReadContinuousNumberSrt,
	}, nil
}

func (s *komgaService) GetBookReadProgress(ctx context.Context, bookID string, claims *response.JWTClaims) (*response.KomgaBookReadProgress, error) {
	if _, _, err := s.accessibleBook(ctx, bookID, claims); err != nil {
		return nil, err
	}
	images, err := s.pageNames(ctx, bookID)
	if err != nil {
		return nil, err
	}
	totalPages := len(images)
	if totalPages == 0 {
		totalPages = 1
	}

	progress, err := s.features.GetReadingProgress(ctx, claims.UId, bookID)
	if err != nil || progress == nil {
		now := time.Now().UTC().Format(komgaTimeFormat)
		return &response.KomgaBookReadProgress{
			Page:         1,
			Completed:    false,
			ReadDate:     nil,
			Created:      now,
			LastModified: now,
		}, nil
	}

	percent := 0.0
	if progress.ProgressPercent != nil {
		percent = *progress.ProgressPercent
	}
	page := int(math.Round(percent * float64(totalPages) / 100.0))
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	completed := percent >= 99.0
	var readDate *string
	if progress.UpdatedAt != nil {
		rd := progress.UpdatedAt.UTC().Format(komgaTimeFormat)
		readDate = &rd
	}
	now := time.Now().UTC().Format(komgaTimeFormat)
	return &response.KomgaBookReadProgress{
		Page:         page,
		Completed:    completed,
		ReadDate:     readDate,
		Created:      now,
		LastModified: now,
	}, nil
}

func (s *komgaService) UpdateBookReadProgress(ctx context.Context, bookID string, page int, completed bool, claims *response.JWTClaims) error {
	book, ref, err := s.accessibleBook(ctx, bookID, claims)
	if err != nil {
		return err
	}
	images, err := s.pageNames(ctx, bookID)
	if err != nil {
		return err
	}
	totalPages := len(images)
	if totalPages == 0 {
		totalPages = 1
	}

	percent := 100.0
	if !completed {
		if page < 1 {
			page = 1
		}
		percent = (float64(page) / float64(totalPages)) * 100.0
		if percent > 100.0 {
			percent = 100.0
		}
	}

	chapterID, chapterTitle := s.firstChapter(ctx, book.ID)
	if chapterID == "" {
		chapterID = book.ID
		chapterTitle = book.Title
	}

	_, err = s.features.RecordReadingActivity(ctx, models.ReadingActivityInput{
		UserID:          claims.UId,
		BookID:          book.ID,
		ChapterID:       chapterID,
		ChapterTitle:    chapterTitle,
		ProgressPercent: &percent,
		EventType:       "progress",
	}, claims)
	if err != nil {
		return err
	}
	if ref != nil && ref.SeriesID != "" {
		_ = s.repo.InvalidateSeriesProgress(ctx, claims.UId, ref.SeriesID)
	}
	return nil
}

func (s *komgaService) DeleteBookReadProgress(ctx context.Context, bookID string, claims *response.JWTClaims) error {
	book, ref, err := s.accessibleBook(ctx, bookID, claims)
	if err != nil {
		return err
	}
	zero := 0.0
	chapterID, chapterTitle := s.firstChapter(ctx, book.ID)
	if chapterID == "" {
		chapterID = book.ID
		chapterTitle = book.Title
	}
	_, err = s.features.RecordReadingActivity(ctx, models.ReadingActivityInput{
		UserID:          claims.UId,
		BookID:          book.ID,
		ChapterID:       chapterID,
		ChapterTitle:    chapterTitle,
		ProgressPercent: &zero,
		EventType:       "progress",
	}, claims)
	if err != nil {
		return err
	}
	if ref != nil && ref.SeriesID != "" {
		_ = s.repo.InvalidateSeriesProgress(ctx, claims.UId, ref.SeriesID)
	}
	return nil
}

func (s *komgaService) ListReadLists(ctx context.Context, page, size int64, claims *response.JWTClaims) (*response.KomgaPageWrapper[response.KomgaReadList], error) {
	if s.readLists == nil {
		return &response.KomgaPageWrapper[response.KomgaReadList]{Empty: true, First: true, Last: true, TotalPages: 1}, nil
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	lists, err := s.readLists.GetUserReadLists(ctx, claims.UId, nil, "", size)
	if err != nil {
		return nil, err
	}
	content := make([]response.KomgaReadList, 0, len(lists))
	for _, l := range lists {
		bookIDs, _ := s.readLists.GetReadListBookIDs(ctx, l.ID)
		created := l.CreatedAt.UTC().Format(komgaTimeFormat)
		updated := l.UpdatedAt.UTC().Format(komgaTimeFormat)
		content = append(content, response.KomgaReadList{
			ID:               l.ID,
			Name:             l.Name,
			Summary:          l.Description,
			Ordered:          true,
			BookIDs:          bookIDs,
			CreatedDate:      created,
			LastModifiedDate: updated,
			Filtered:         false,
		})
	}
	total := int64(len(content))
	return &response.KomgaPageWrapper[response.KomgaReadList]{
		Content:          content,
		Empty:            total == 0,
		First:            true,
		Last:             true,
		Number:           page,
		NumberOfElements: total,
		Size:             size,
		TotalElements:    total,
		TotalPages:       1,
	}, nil
}

func (s *komgaService) GetReadList(ctx context.Context, readListID string, claims *response.JWTClaims) (*response.KomgaReadList, error) {
	if s.readLists == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Read list not found")
	}
	lists, err := s.readLists.GetReadListsByIDs(ctx, []string{readListID})
	if err != nil || len(lists) == 0 || lists[0] == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Read list not found")
	}
	l := lists[0]
	bookIDs, _ := s.readLists.GetReadListBookIDs(ctx, l.ID)
	created := l.CreatedAt.UTC().Format(komgaTimeFormat)
	updated := l.UpdatedAt.UTC().Format(komgaTimeFormat)
	return &response.KomgaReadList{
		ID:               l.ID,
		Name:             l.Name,
		Summary:          l.Description,
		Ordered:          true,
		BookIDs:          bookIDs,
		CreatedDate:      created,
		LastModifiedDate: updated,
		Filtered:         false,
	}, nil
}

func (s *komgaService) GetReadListBooks(ctx context.Context, readListID string, claims *response.JWTClaims) ([]response.KomgaBook, error) {
	if s.readLists == nil {
		return []response.KomgaBook{}, nil
	}
	bookIDs, err := s.readLists.GetReadListBookIDs(ctx, readListID)
	if err != nil || len(bookIDs) == 0 {
		return []response.KomgaBook{}, nil
	}
	books := make([]response.KomgaBook, 0, len(bookIDs))
	for _, id := range bookIDs {
		b, err := s.GetBook(ctx, id, claims)
		if err == nil && b != nil {
			books = append(books, *b)
		}
	}
	return books, nil
}
