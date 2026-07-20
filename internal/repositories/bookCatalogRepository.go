package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

func (r *bookDBRepository) CreateBook(ctx context.Context, book *models.BookEntity) error {
	params := sqlc.CreateBookParams{
		ID:           book.ID,
		LibraryID:    book.LibraryID,
		Title:        book.Title,
		AuthorID:     convert.StrPtrToNullString(book.AuthorID),
		Description:  convert.StrPtrToNullString(book.Description),
		CoverUrl:     convert.StrPtrToNullString(book.CoverURL),
		Status:       convert.StrPtrToNullString(&book.Status),
		MetadataJson: convert.StrPtrToNullString(book.MetadataJSON),
	}
	res, err := r.queries.CreateBook(ctx, params)
	if err != nil {
		return err
	}
	book.CreatedAt = res.CreatedAt.Time
	book.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", book.ID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) CreateBookWithFile(ctx context.Context, book *models.BookEntity, file *BookFileRecordParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	txRepo := r.WithTx(tx)
	if err := txRepo.CreateBook(ctx, book); err != nil {
		return err
	}
	if err := txRepo.CreateBookFile(ctx, *file); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if r.c != nil {
		_ = r.c.Del(
			ctx,
			fmt.Sprintf("book:id:%s", book.ID),
			fmt.Sprintf("book_file:book:%s", book.ID),
			"feature:library_stats",
			"book_file:all",
			"book_file:duplicates",
			fmt.Sprintf("book_file:count:%s", book.ID),
		)
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetBook(ctx context.Context, id string) (*models.BookEntity, error) {
	key := fmt.Sprintf("book:id:%s", id)
	if r.c != nil && !r.inTx {
		var book models.BookEntity
		if err := r.c.Get(ctx, key, &book); err == nil {
			return &book, nil
		}
	}

	res, err := r.queries.GetBook(ctx, id)
	if err != nil {
		return nil, err
	}
	book := (&models.BookEntity{}).FromSqlc(res)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, book, constants.NormalCacheDuration)
	}
	return book, nil
}

func (r *bookDBRepository) SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, limit, offset int64) ([]*models.BookEntity, error) {
	if nav == "random" {
		var libID interface{}
		libStr := ""
		if libraryID != nil && *libraryID != "" {
			libID = *libraryID
			libStr = *libraryID
		}
		cacheKey := fmt.Sprintf("book:search:random:%s:%d", libStr, limit)
		if r.c != nil && !r.inTx {
			var ids []string
			if err := r.c.Get(ctx, cacheKey, &ids); err == nil {
				return r.GetBooksByIDs(ctx, ids)
			}
		}

		ids, err := r.queries.GetRandomBookIDs(ctx, sqlc.GetRandomBookIDsParams{
			LibraryID: libID,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, ids, 30*time.Second)
		}
		return r.GetBooksByIDs(ctx, ids)
	}

	if collection != "" && collection != "Missing metadata" {
		ids, err := r.queries.GetBookIDsInCollection(ctx, collection)
		if err != nil {
			return nil, err
		}
		total := int64(len(ids))
		if offset >= total {
			return []*models.BookEntity{}, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		paginatedIDs := ids[offset:end]
		return r.GetBooksByIDs(ctx, paginatedIDs)
	}
	if chip != "" && chip != "All" && chip != "No cover" && chip != "Duplicates" && chip != "Reading" && chip != "Unread" {
		return []*models.BookEntity{}, nil
	}

	filters := buildBookSearchFilters(nav, collection, chip, facet, facetID)
	if !filters.Valid {
		return []*models.BookEntity{}, nil
	}

	var libID, searchStr interface{}
	if libraryID != nil && *libraryID != "" {
		libID = *libraryID
	}
	if search != nil && *search != "" {
		searchStr = *search
	}

	params := sqlc.SearchBookIDsParams{
		LibraryID:             libID,
		Search:                searchStr,
		FilterMissingMetadata: filters.MissingMetadata,
		FilterNoCover:         filters.NoCover,
		FilterHasFiles:        filters.HasFiles,
		FilterHasAuthor:       filters.HasAuthor,
		FilterHasSeries:       filters.HasSeries,
		FilterHasTags:         filters.HasTags,
		FilterHasPublishers:   filters.HasPublishers,
		FilterHasLanguages:    filters.HasLanguages,
		FilterHasFormats:      filters.HasFormats,
		FilterReading:         filters.Reading,
		FilterRead:            filters.Read,
		FilterUnread:          filters.Unread,
		FilterHot:             filters.Hot,
		FilterTopDownloaded:   filters.TopDownloaded,
		FilterTopRated:        filters.TopRated,
		FilterArchived:        filters.Archived,
		FilterBookmarked:      filters.Bookmarked,
		UserID:                filters.UserID,
		AuthorID:              filters.AuthorID,
		SeriesID:              filters.SeriesID,
		TagID:                 filters.TagID,
		PublisherID:           filters.PublisherID,
		LanguageID:            filters.LanguageID,
		FileFormat:            filters.FileFormat,
		Limit:                 limit,
		Offset:                offset,
	}
	queryKey := cache.QueryKey("book:search", params)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, queryKey, &ids); err == nil {
			return r.GetBooksByIDs(ctx, ids)
		}
	}

	ids, err := r.queries.SearchBookIDs(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}
	return r.GetBooksByIDs(ctx, ids)
}

func (r *bookDBRepository) SearchBooksCursor(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, cursor time.Time, limit int64) ([]*models.BookEntity, error) {
	if nav == "random" {
		var libID interface{}
		libStr := ""
		if libraryID != nil && *libraryID != "" {
			libID = *libraryID
			libStr = *libraryID
		}
		cacheKey := fmt.Sprintf("book:search:random:%s:%d", libStr, limit)
		if r.c != nil && !r.inTx {
			var ids []string
			if err := r.c.Get(ctx, cacheKey, &ids); err == nil {
				return r.GetBooksByIDs(ctx, ids)
			}
		}

		ids, err := r.queries.GetRandomBookIDs(ctx, sqlc.GetRandomBookIDsParams{
			LibraryID: libID,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, ids, 30*time.Second)
		}
		return r.GetBooksByIDs(ctx, ids)
	}

	if collection != "" && collection != "Missing metadata" {
		ids, err := r.queries.GetBookIDsInCollection(ctx, collection)
		if err != nil {
			return nil, err
		}
		books, err := r.GetBooksByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		var filtered []*models.BookEntity
		for _, b := range books {
			if b.CreatedAt.Before(cursor) {
				filtered = append(filtered, b)
			}
		}
		if int64(len(filtered)) > limit {
			return filtered[:limit], nil
		}
		return filtered, nil
	}
	if chip != "" && chip != "All" && chip != "No cover" && chip != "Duplicates" && chip != "Reading" && chip != "Unread" {
		return []*models.BookEntity{}, nil
	}

	filters := buildBookSearchFilters(nav, collection, chip, facet, facetID)
	if !filters.Valid {
		return []*models.BookEntity{}, nil
	}

	var libID, searchStr interface{}
	if libraryID != nil && *libraryID != "" {
		libID = *libraryID
	}
	if search != nil && *search != "" {
		searchStr = *search
	}

	params := sqlc.SearchBookIDsCursorParams{
		CreatedAt:             sql.NullTime{Time: cursor, Valid: true},
		LibraryID:             libID,
		Search:                searchStr,
		FilterMissingMetadata: filters.MissingMetadata,
		FilterNoCover:         filters.NoCover,
		FilterHasFiles:        filters.HasFiles,
		FilterHasAuthor:       filters.HasAuthor,
		FilterHasSeries:       filters.HasSeries,
		FilterHasTags:         filters.HasTags,
		FilterHasPublishers:   filters.HasPublishers,
		FilterHasLanguages:    filters.HasLanguages,
		FilterHasFormats:      filters.HasFormats,
		FilterReading:         filters.Reading,
		FilterRead:            filters.Read,
		FilterUnread:          filters.Unread,
		FilterHot:             filters.Hot,
		FilterTopDownloaded:   filters.TopDownloaded,
		FilterTopRated:        filters.TopRated,
		FilterArchived:        filters.Archived,
		FilterBookmarked:      filters.Bookmarked,
		UserID:                filters.UserID,
		AuthorID:              filters.AuthorID,
		SeriesID:              filters.SeriesID,
		TagID:                 filters.TagID,
		PublisherID:           filters.PublisherID,
		LanguageID:            filters.LanguageID,
		FileFormat:            filters.FileFormat,
		Limit:                 limit,
	}
	queryKey := cache.QueryKey("book:search:cursor", params)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, queryKey, &ids); err == nil {
			return r.GetBooksByIDs(ctx, ids)
		}
	}

	ids, err := r.queries.SearchBookIDsCursor(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}
	return r.GetBooksByIDs(ctx, ids)
}

type bookSearchFilters struct {
	Valid           bool
	MissingMetadata interface{}
	NoCover         interface{}
	HasFiles        interface{}
	HasAuthor       interface{}
	HasSeries       interface{}
	HasTags         interface{}
	HasPublishers   interface{}
	HasLanguages    interface{}
	HasFormats      interface{}
	Reading         interface{}
	Read            interface{}
	Unread          interface{}
	Hot             interface{}
	TopDownloaded   interface{}
	TopRated        interface{}
	Archived        interface{}
	Bookmarked      interface{}
	UserID          sql.NullInt64
	AuthorID        interface{}
	SeriesID        interface{}
	TagID           interface{}
	PublisherID     interface{}
	LanguageID      interface{}
	FileFormat      interface{}
}

func buildBookSearchFilters(nav, collection, chip, facet, facetID string) bookSearchFilters {
	filters := bookSearchFilters{Valid: true}
	set := func(target *interface{}) {
		*target = 1
	}

	switch strings.ToLower(strings.TrimSpace(nav)) {
	case "", "all novels", "books", "explore", "book_lists":
	case "hot":
		set(&filters.Hot)
	case "downloaded":
		set(&filters.TopDownloaded)
	case "top_rated":
		set(&filters.TopRated)
	case "reading":
		set(&filters.Reading)
	case "read":
		set(&filters.Read)
	case "unread":
		set(&filters.Unread)
	case "series":
		set(&filters.HasSeries)
	case "authors":
		set(&filters.HasAuthor)
	case "tags":
		set(&filters.HasTags)
	case "publishers":
		set(&filters.HasPublishers)
	case "languages":
		set(&filters.HasLanguages)
	case "ratings":
		set(&filters.TopRated)
	case "formats":
		set(&filters.HasFormats)
	case "archived":
		set(&filters.Archived)
	default:
		filters.Valid = false
	}

	if collection == "Missing metadata" {
		set(&filters.MissingMetadata)
	}
	switch chip {
	case "No cover":
		set(&filters.NoCover)
	case "Reading":
		set(&filters.Reading)
	case "Unread":
		set(&filters.Unread)
	}

	if facetID == "" {
		return filters
	}
	switch strings.ToLower(strings.TrimSpace(facet)) {
	case "author":
		filters.AuthorID = facetID
	case "series":
		filters.SeriesID = facetID
	case "tag":
		filters.TagID = facetID
	case "publisher":
		filters.PublisherID = facetID
	case "language":
		filters.LanguageID = facetID
	case "format":
		filters.FileFormat = facetID
	default:
		filters.Valid = false
	}
	return filters
}

func (r *bookDBRepository) UpdateBook(ctx context.Context, book *models.BookEntity) error {
	params := sqlc.UpdateBookParams{
		ID:           book.ID,
		Title:        book.Title,
		AuthorID:     convert.StrPtrToNullString(book.AuthorID),
		Description:  convert.StrPtrToNullString(book.Description),
		CoverUrl:     convert.StrPtrToNullString(book.CoverURL),
		Status:       convert.StrPtrToNullString(&book.Status),
		MetadataJson: convert.StrPtrToNullString(book.MetadataJSON),
	}
	res, err := r.queries.UpdateBook(ctx, params)
	if err != nil {
		return err
	}
	book.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", book.ID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
	}
	return nil
}

func (r *bookDBRepository) DeleteBook(ctx context.Context, id string) error {
	files, _ := r.queries.GetFilesByBookId(ctx, id)
	chapterIDs, _ := r.queries.ListChapterIDsByBook(ctx, id)
	if err := r.queries.DeleteBook(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			fmt.Sprintf("book:id:%s", id),
			fmt.Sprintf("chapter:book:%s", id),
			fmt.Sprintf("book_file:book:%s", id),
			fmt.Sprintf("book_file:count:%s", id),
			"feature:library_stats",
		)
		for _, chapterID := range chapterIDs {
			_ = r.c.Del(ctx, fmt.Sprintf("chapter:id:%s", chapterID))
		}
		for _, file := range files {
			_ = r.c.Del(
				ctx,
				fmt.Sprintf("book_file:id:%s", file.ID),
				fmt.Sprintf("book_file:path:%s", file.Path),
				fmt.Sprintf("book_file:book:%s", file.BookID),
				fmt.Sprintf("book_file:count:%s", file.BookID),
				"book_file:all",
				"book_file:duplicates",
			)
		}
		_ = r.c.DelByPattern(context.Background(), "book:search*")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
	}
	return nil
}

func (r *bookDBRepository) GetBooksByIDs(ctx context.Context, ids []string) ([]*models.BookEntity, error) {
	if len(ids) == 0 {
		return []*models.BookEntity{}, nil
	}
	if r.c == nil || r.inTx {
		return r.fetchBooksByIDs(ctx, ids)
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("book:id:%s", id)
	}

	booksByID := make(map[string]*models.BookEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))
	cachedBytes := r.c.MGet(ctx, keys...)
	for i, bytes := range cachedBytes {
		if len(bytes) > 0 {
			var book models.BookEntity
			if err := jsonx.Unmarshal(bytes, &book); err == nil {
				booksByID[book.ID] = &book
				continue
			}
		}
		missingIDs = append(missingIDs, ids[i])
		missingKeys = append(missingKeys, keys[i])
	}

	if len(missingIDs) > 0 {
		rows, err := r.queries.GetBooksByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		missingMap := make(map[string]*models.BookEntity, len(rows))
		for _, row := range rows {
			book := (&models.BookEntity{}).FromSqlc(row)
			booksByID[book.ID] = book
			missingMap[book.ID] = book
		}
		missingToCache := make(map[string]any, len(missingMap))
		for i, missingID := range missingIDs {
			if book, ok := missingMap[missingID]; ok {
				missingToCache[missingKeys[i]] = book
			}
		}
		if len(missingToCache) > 0 {
			_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
		}
	}

	return orderBooks(ids, booksByID), nil
}

func (r *bookDBRepository) fetchBooksByIDs(ctx context.Context, ids []string) ([]*models.BookEntity, error) {
	rows, err := r.queries.GetBooksByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	booksByID := make(map[string]*models.BookEntity, len(rows))
	for _, row := range rows {
		book := (&models.BookEntity{}).FromSqlc(row)
		booksByID[book.ID] = book
	}
	return orderBooks(ids, booksByID), nil
}

func orderBooks(ids []string, booksByID map[string]*models.BookEntity) []*models.BookEntity {
	ordered := make([]*models.BookEntity, 0, len(ids))
	for _, id := range ids {
		if book, ok := booksByID[id]; ok {
			ordered = append(ordered, book)
		}
	}
	return ordered
}
