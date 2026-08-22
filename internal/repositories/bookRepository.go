package repositories

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/dtos/request"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
)

type BookCatalogRepository interface {
	CreateBook(ctx context.Context, book *models.BookEntity) error
	GetBook(ctx context.Context, id string) (*models.BookEntity, error)
	SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, sort string, cursor string, limit int64, userID string) ([]*models.BookEntity, error)
	SearchSmartFilterBooks(ctx context.Context, libraryID *string, rules []request.SmartFilterRuleItemDto, cursor *time.Time, cursorID string, limit int64, userID string) ([]*models.BookEntity, error)
	UpdateBook(ctx context.Context, book *models.BookEntity) error
	DeleteBook(ctx context.Context, id string) error
	GetBooksByIDs(ctx context.Context, ids []string) ([]*models.BookEntity, error)
	CreateBookWithFile(ctx context.Context, book *models.BookEntity, file *sqlc.CreateBookFileParams) error
	ListBookIDs(ctx context.Context, cursor *time.Time, cursorID string, limit int64) ([]string, error)
	ListBookIDsByLibrary(ctx context.Context, libraryID string, limit int64) ([]string, error)
	BulkUpdateBookLibrary(ctx context.Context, bookIDs []string, libraryID string) error
	BulkDeleteBooks(ctx context.Context, bookIDs []string) error
	WithTx(tx *sql.Tx) BookDBRepository
}

type BookFileRecordRepository interface {
	CreateBookFile(ctx context.Context, params sqlc.CreateBookFileParams) error
	UpsertBookFile(ctx context.Context, params sqlc.UpsertBookFileParams) error
	GetFilesByBookId(ctx context.Context, bookID string) ([]*models.BookFileEntity, error)
	GetFilesByBookIDs(ctx context.Context, bookIDs []string) ([]*models.BookFileEntity, error)
	GetBookFileByPath(ctx context.Context, path string) (*models.BookFileEntity, error)
	GetBookFileById(ctx context.Context, id string) (*models.BookFileEntity, error)
	UpdateBookFileHash(ctx context.Context, id string, hash string) error
	GetDuplicateFiles(ctx context.Context, limit, offset int64) ([]*models.DuplicateFileEntity, error)
	GetDuplicateFileDetails(ctx context.Context, limit int64) ([]*models.DuplicateFileDetailEntity, error)
	ListAllFiles(ctx context.Context, limit, offset int64) ([]*models.FileRefEntity, error)
	DeleteFile(ctx context.Context, id string) error
	RepointFileUserData(ctx context.Context, oldFileID string, newFileID string) error
	CountFilesForBook(ctx context.Context, bookID string) (int64, error)
	WithTx(tx *sql.Tx) BookDBRepository
}

type ChapterRepository interface {
	CreateChapter(ctx context.Context, chapter *models.ChapterEntity) error
	GetChapter(ctx context.Context, id string) (*models.ChapterEntity, error)
	ListChaptersByBook(ctx context.Context, bookID string) ([]*models.ChapterEntity, error)
	GetChaptersByIDs(ctx context.Context, ids []string) ([]*models.ChapterEntity, error)
	DeleteChapter(ctx context.Context, id string) error
	DeleteChaptersByBook(ctx context.Context, bookID string) error
	WithTx(tx *sql.Tx) BookDBRepository
}

type BookMetadataRepository interface {
	CreateAuthor(ctx context.Context, author *models.AuthorEntity) error
	GetAuthorByName(ctx context.Context, name string) (*models.AuthorEntity, error)
	GetAuthorByID(ctx context.Context, id string) (*models.AuthorEntity, error)
	GetAuthorsByIDs(ctx context.Context, ids []string) ([]*models.AuthorEntity, error)
	CreateTag(ctx context.Context, tag *models.TagEntity) error
	GetTagByName(ctx context.Context, name string) (*models.TagEntity, error)
	AddBookTag(ctx context.Context, bookID, tagID string) error
	GetSeriesByName(ctx context.Context, name string) (*models.SeriesEntity, error)
	CreateSeries(ctx context.Context, series *models.SeriesEntity) error
	LinkBookSeries(ctx context.Context, bookID, seriesID string, seriesIndex *string) error
	ClearBookSeries(ctx context.Context, bookID string) error
	GetBookSeries(ctx context.Context, bookID string) ([]*models.BookSeriesEntity, error)
	GetNextBookInSeries(ctx context.Context, seriesID, currentBookID string) (*models.NextInSeriesEntity, error)
	GetPublisherByName(ctx context.Context, name string) (*models.PublisherEntity, error)
	CreatePublisher(ctx context.Context, publisher *models.PublisherEntity) error
	LinkBookPublisher(ctx context.Context, bookID, publisherID string) error
	ClearBookPublishers(ctx context.Context, bookID string) error
	GetLanguageByName(ctx context.Context, name string) (*models.LanguageEntity, error)
	CreateLanguage(ctx context.Context, language *models.LanguageEntity) error
	LinkBookLanguage(ctx context.Context, bookID, languageID string) error
	ClearBookLanguages(ctx context.Context, bookID string) error
	ClearBookTags(ctx context.Context, bookID string) error
	ListAuthorsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	ListSeriesWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	ListPublishersWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	ListLanguagesWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	ListTagsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	ListFormatsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	ListRatingsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error)
	WithTx(tx *sql.Tx) BookDBRepository
}

type BookFTSRepository interface {
	SearchFTS(ctx context.Context, query string, limit, offset int64) ([]*models.FTSResultEntity, error)
	SearchFTSInBook(ctx context.Context, bookID, query string) ([]*models.BookSearchSnippet, error)
	DeleteFTSBook(ctx context.Context, bookID string) error
	InsertFTSChapter(ctx context.Context, bookID, chapterID, title, content string) error
	WithTx(tx *sql.Tx) BookDBRepository
}

type BookMergeRepository interface {
	ListAllTitleAuthor(ctx context.Context) ([]*models.BookTitleAuthorEntity, error)
	MergeBookData(ctx context.Context, sourceID, targetID string) error
}

type BookDBRepository interface {
	BookCatalogRepository
	BookFileRecordRepository
	ChapterRepository
	BookMetadataRepository
	BookFTSRepository
	BookMergeRepository
	WithTx(tx *sql.Tx) BookDBRepository
	FlushCache(ctx context.Context)
}

type bookDBRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewBookDBRepository(db *sql.DB, c cache.Cache) BookDBRepository {
	return &bookDBRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

// The returned repository buffers its invalidations; FlushCache after Commit publishes them.
// Inline sweeps would run a full-cache scan per mutated row while the write lock is held.
func (r *bookDBRepository) WithTx(tx *sql.Tx) BookDBRepository {
	var c cache.Cache
	if r.c != nil {
		c = cache.NewDeferred(r.c)
	}
	return &bookDBRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
		c:       c,
		inTx:    true,
		sfg:     &singleflight.Group{},
	}
}

func (r *bookDBRepository) FlushCache(ctx context.Context) {
	if d, ok := r.c.(*cache.DeferredCache); ok {
		d.Flush(ctx)
	}
}
