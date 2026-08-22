package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

func (r *bookDBRepository) CreateBook(ctx context.Context, book *models.BookEntity) error {
	params := sqlc.CreateBookParams{
		ID:            book.ID,
		LibraryID:     book.LibraryID,
		Title:         book.Title,
		AuthorID:      convert.StrPtrToNullString(book.AuthorID),
		Description:   convert.StrPtrToNullString(book.Description),
		CoverUrl:      convert.StrPtrToNullString(book.CoverURL),
		Status:        convert.StrPtrToNullString(&book.Status),
		MetadataJson:  convert.StrPtrToNullString(book.MetadataJSON),
		GoogleBooksID: convert.StrPtrToNullString(book.GoogleBooksID),
		AnilistID:     convert.StrPtrToNullString(book.AnilistID),
		OpenlibraryID: convert.StrPtrToNullString(book.OpenLibraryID),
	}
	res, err := r.queries.CreateBook(ctx, params)
	if err != nil {
		return err
	}
	book.CreatedAt = res.CreatedAt
	book.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", book.ID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
	}
	return nil
}

func (r *bookDBRepository) CreateBookWithFile(ctx context.Context, book *models.BookEntity, file *sqlc.CreateBookFileParams) error {
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
			cache.BuildKey("book", "id", book.ID),
			cache.BuildKey("book_file", "book", book.ID),
			constants.CacheKeyLibraryStats,
			cache.BuildKey("book_file", "count", book.ID),
		)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookFileAllPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookFileDupesPattern)
	}
	return nil
}

func (r *bookDBRepository) ListBookIDs(ctx context.Context, cursor *time.Time, cursorID string, limit int64) ([]string, error) {
	cacheKey := cache.BuildKey("book_ids", cursor, cursorID, limit)
	if r.c != nil && !r.inTx {
		var cachedIDs []string
		if err := r.c.Get(ctx, cacheKey, &cachedIDs); err == nil {
			return cachedIDs, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		ids, err := r.queries.ListBookIDs(ctx, sqlc.ListBookIDsParams{
			CursorCreatedAt: cursorTimeArg(cursor),
			CursorID:        convert.StrPtrToNullString(&cursorID),
			Limit:           limit,
		})
		if err != nil {
			return nil, err
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *bookDBRepository) GetBook(ctx context.Context, id string) (*models.BookEntity, error) {
	key := cache.BuildKey("book", "id", id)
	if r.c != nil && !r.inTx {
		if v, ok := r.c.GetObject(key); ok {
			if book, ok := v.(*models.BookEntity); ok && book != nil {
				// Shared cache object: hand out a shallow copy so callers may
				// assign fields (metadata edits) without touching the cache.
				out := *book
				return &out, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetBook(ctx, id)
		if err != nil {
			return nil, err
		}
		book := (&models.BookEntity{}).FromSqlc(res)
		r.enrichBookEntitiesForCache(ctx, []*models.BookEntity{book})
		if r.c != nil && !r.inTx {
			_ = r.c.SetObject(ctx, key, book, constants.NormalCacheDuration)
		}
		return book, nil
	})
	if err != nil {
		return nil, err
	}
	out := *v.(*models.BookEntity)
	return &out, nil
}

func (r *bookDBRepository) SearchBooks(ctx context.Context, libraryID *string, search *string, nav, collection, chip, facet, facetID string, sort string, cursor string, limit int64, userID string) ([]*models.BookEntity, error) {
	if nav == "random" {
		var libID any
		libStr := ""
		if libraryID != nil && *libraryID != "" {
			libID = *libraryID
			libStr = *libraryID
		}
		cacheKey := cache.BuildKey("book", "search", "random", libStr, limit)
		if r.c != nil && !r.inTx {
			var ids []string
			if err := r.c.Get(ctx, cacheKey, &ids); err == nil {
				return r.GetBooksByIDs(ctx, ids)
			}
		}

		value, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
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
			return ids, nil
		})
		if err != nil {
			return nil, err
		}
		return r.GetBooksByIDs(ctx, value.([]string))
	}

	if collection != "" && collection != "Missing metadata" {
		colKey := cache.BuildKey("book_ids", "collection", collection, cursor, limit)
		if r.c != nil && !r.inTx {
			var cachedIDs []string
			if err := r.c.Get(ctx, colKey, &cachedIDs); err == nil {
				return r.GetBooksByIDs(ctx, cachedIDs)
			}
		}

		value, err, _ := r.sfg.Do(colKey, func() (any, error) {
			var cursorTime *time.Time
			var cursorID string
			if cursor != "" {
				if parts := convert.DecodeCursor(cursor); len(parts) == 2 {
					if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
						cursorTime = &t
					} else if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
						cursorTime = &t
					}
					cursorID = parts[1]
				} else if parts := strings.SplitN(cursor, "|", 2); len(parts) == 2 {
					if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
						cursorTime = &t
					} else if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
						cursorTime = &t
					}
					cursorID = parts[1]
				}
			}

			ids, err := r.queries.GetBookIDsInCollection(ctx, sqlc.GetBookIDsInCollectionParams{
				CollectionID:    collection,
				CursorCreatedAt: cursorTimeArg(cursorTime),
				CursorID:        convert.StrPtrToNullString(&cursorID),
				Limit:           limit,
			})
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, colKey, ids, constants.ListCacheDuration)
			}
			return ids, nil
		})
		if err != nil {
			return nil, err
		}
		return r.GetBooksByIDs(ctx, value.([]string))
	}

	if chip != "" && chip != "All" && chip != "No cover" && chip != "Duplicates" && chip != "Reading" && chip != "Unread" && chip != "ExcludeAudiobooks" {
		return []*models.BookEntity{}, nil
	}

	filters := buildBookSearchFilters(nav, collection, chip, facet, facetID, userID)
	if !filters.Valid {
		return []*models.BookEntity{}, nil
	}

	var libID, searchStr any
	if libraryID != nil && *libraryID != "" {
		libID = *libraryID
	}
	if search != nil && *search != "" {
		if match, ok := ftsBookMatchQuery(*search); ok {
			searchStr = match
		}
	}

	if sort == "title_az" {
		var cursorTitle *string
		var cursorID string
		if cursor != "" {
			if parts := convert.DecodeCursor(cursor); len(parts) == 2 {
				cursorTitle = &parts[0]
				cursorID = parts[1]
			} else if parts := strings.SplitN(cursor, "|", 2); len(parts) == 2 {
				cursorTitle = &parts[0]
				cursorID = parts[1]
			}
		}

		params := sqlc.SearchBookIDsOrderByTitleParams{
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
			FilterRatingStar:      filters.RatingStar,
			FilterArchived:        filters.Archived,
			FilterBookmarked:      filters.Bookmarked,
			UserID:                filters.UserID,
			AuthorID:              filters.AuthorID,
			SeriesID:              filters.SeriesID,
			TagID:                 filters.TagID,
			PublisherID:           filters.PublisherID,
			LanguageID:            filters.LanguageID,
			FileFormat:            filters.FileFormat,
			ExcludeAudiobooks:     filters.ExcludeAudiobooks,
			CursorTitle:           convert.StrPtrToNullString(cursorTitle),
			CursorID:              convert.StrPtrToNullString(&cursorID),
			Limit:                 limit,
		}

		queryKey := cache.QueryKey("book:search:cursor:title", params)
		if r.c != nil && !r.inTx {
			var ids []string
			if err := r.c.Get(ctx, queryKey, &ids); err == nil {
				return r.GetBooksByIDs(ctx, ids)
			}
		}

		value, err, _ := r.sfg.Do(queryKey, func() (any, error) {
			ids, err := r.queries.SearchBookIDsOrderByTitle(ctx, params)
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
			}
			return ids, nil
		})
		if err != nil {
			return nil, err
		}
		return r.GetBooksByIDs(ctx, value.([]string))
	}

	if sort == "series_order" {
		var cursorSeriesName *string
		var cursorSeriesIndex *string
		var cursorID string
		if cursor != "" {
			if parts := convert.DecodeCursor(cursor); len(parts) == 2 {
				subParts := strings.SplitN(parts[0], "|", 2)
				if len(subParts) == 2 {
					cursorSeriesName = &subParts[0]
					cursorSeriesIndex = &subParts[1]
				} else {
					cursorSeriesName = &parts[0]
				}
				cursorID = parts[1]
			} else if parts := strings.SplitN(cursor, "|", 2); len(parts) == 2 {
				subParts := strings.SplitN(parts[0], "|", 2)
				if len(subParts) == 2 {
					cursorSeriesName = &subParts[0]
					cursorSeriesIndex = &subParts[1]
				} else {
					cursorSeriesName = &parts[0]
				}
				cursorID = parts[1]
			}
		}

		var cursorSeriesIndexFloat sql.NullFloat64
		if cursorSeriesIndex != nil && *cursorSeriesIndex != "" {
			if f, err := strconv.ParseFloat(*cursorSeriesIndex, 64); err == nil {
				cursorSeriesIndexFloat.Float64 = f
				cursorSeriesIndexFloat.Valid = true
			}
		}

		params := sqlc.SearchBookIDsOrderBySeriesParams{
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
			FilterRatingStar:      filters.RatingStar,
			FilterArchived:        filters.Archived,
			FilterBookmarked:      filters.Bookmarked,
			UserID:                filters.UserID,
			AuthorID:              filters.AuthorID,
			SeriesID:              filters.SeriesID,
			TagID:                 filters.TagID,
			PublisherID:           filters.PublisherID,
			LanguageID:            filters.LanguageID,
			FileFormat:            filters.FileFormat,
			ExcludeAudiobooks:     filters.ExcludeAudiobooks,
			CursorSeriesName:      convert.StrPtrToNullString(cursorSeriesName),
			CursorSeriesIndex:     cursorSeriesIndexFloat,
			CursorID:              convert.StrPtrToNullString(&cursorID),
			Limit:                 limit,
		}

		queryKey := cache.QueryKey("book:search:cursor:series", params)
		if r.c != nil && !r.inTx {
			var ids []string
			if err := r.c.Get(ctx, queryKey, &ids); err == nil {
				return r.GetBooksByIDs(ctx, ids)
			}
		}

		value, err, _ := r.sfg.Do(queryKey, func() (any, error) {
			rows, err := r.queries.SearchBookIDsOrderBySeries(ctx, params)
			if err != nil {
				return nil, err
			}
			ids := make([]string, len(rows))
			for i, row := range rows {
				ids[i] = row.ID
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
			}
			return ids, nil
		})
		if err != nil {
			return nil, err
		}
		return r.GetBooksByIDs(ctx, value.([]string))
	}

	var cursorTime *time.Time
	var cursorID string
	if cursor != "" {
		if parts := convert.DecodeCursor(cursor); len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
			} else if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				cursorTime = &t
			}
			cursorID = parts[1]
		} else if parts := strings.SplitN(cursor, "|", 2); len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = &t
			} else if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				cursorTime = &t
			}
			cursorID = parts[1]
		}
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
		FilterRatingStar:      filters.RatingStar,
		FilterArchived:        filters.Archived,
		FilterBookmarked:      filters.Bookmarked,
		UserID:                filters.UserID,
		AuthorID:              filters.AuthorID,
		SeriesID:              filters.SeriesID,
		TagID:                 filters.TagID,
		PublisherID:           filters.PublisherID,
		LanguageID:            filters.LanguageID,
		FileFormat:            filters.FileFormat,
		ExcludeAudiobooks:     filters.ExcludeAudiobooks,
		CursorCreatedAt:       cursorTimeArg(cursorTime),
		CursorID:              convert.StrPtrToNullString(&cursorID),
		Limit:                 limit,
	}

	queryKey := cache.QueryKey("book:search:cursor", params)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, queryKey, &ids); err == nil {
			return r.GetBooksByIDs(ctx, ids)
		}
	}

	value, err, _ := r.sfg.Do(queryKey, func() (any, error) {
		ids, err := r.queries.SearchBookIDs(ctx, params)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetBooksByIDs(ctx, value.([]string))
}

func (r *bookDBRepository) SearchSmartFilterBooks(ctx context.Context, libraryID *string, rules []request.SmartFilterRuleItemDto, cursor *time.Time, cursorID string, limit int64, userID string) ([]*models.BookEntity, error) {
	var libID any
	if libraryID != nil && *libraryID != "" {
		libID = *libraryID
	}

	var userNullID sql.NullString
	if strings.TrimSpace(userID) != "" && userID != "0" {
		userNullID = convert.StrPtrToNullString(&userID)
	}

	params := sqlc.SearchSmartFilterBookIDsParams{
		LibraryID:       libID,
		UserID:          userNullID,
		CursorCreatedAt: cursorTimeArg(cursor),
		CursorID:        convert.StrPtrToNullString(&cursorID),
		Limit:           limit,
	}

	for _, rule := range rules {
		val := strings.TrimSpace(rule.Value)
		switch rule.Field {
		case "format":
			params.FileFormat = val
		case "status":
			switch val {
			case "unread":
				params.StatusUnread = 1
			case "read":
				params.StatusRead = 1
			case "reading":
				params.StatusReading = 1
			}
		case "rating_gte":
			if rVal, err := strconv.ParseFloat(val, 64); err == nil {
				params.RatingGte = rVal
			}
		case "author_id":
			params.AuthorID = val
		case "series_id":
			params.SeriesID = val
		case "tag_id":
			params.TagID = val
		}
	}

	queryKey := cache.QueryKey("book:search:smartfilter", params)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, queryKey, &ids); err == nil {
			return r.GetBooksByIDs(ctx, ids)
		}
	}

	value, err, _ := r.sfg.Do(queryKey, func() (any, error) {
		ids, err := r.queries.SearchSmartFilterBookIDs(ctx, params)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetBooksByIDs(ctx, value.([]string))
}

type bookSearchFilters struct {
	Valid           bool
	MissingMetadata any
	NoCover         any
	HasFiles        any
	HasAuthor       any
	HasSeries       any
	HasTags         any
	HasPublishers   any
	HasLanguages    any
	HasFormats      any
	Reading         any
	Read            any
	Unread          any
	Hot             any
	TopDownloaded   any
	TopRated        any
	RatingStar      any
	Archived        any
	Bookmarked      any
	UserID          sql.NullString
	AuthorID        any
	SeriesID        any
	TagID           any
	PublisherID     any
	LanguageID      any
	FileFormat      any
	ExcludeAudiobooks any
}

func buildBookSearchFilters(nav, collection, chip, facet, facetID string, userID string) bookSearchFilters {
	filters := bookSearchFilters{Valid: true}
	if strings.TrimSpace(userID) != "" && userID != "0" {
		filters.UserID = convert.StrPtrToNullString(&userID)
	}
	set := func(target *any) {
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
	case "ExcludeAudiobooks":
		set(&filters.ExcludeAudiobooks)
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
	case "rating", "ratings":
		if star, err := strconv.ParseInt(strings.TrimSpace(facetID), 10, 64); err == nil && star >= 1 && star <= 5 {
			filters.RatingStar = star
		} else {
			set(&filters.TopRated)
		}
	default:
		filters.Valid = false
	}
	return filters
}

func (r *bookDBRepository) UpdateBook(ctx context.Context, book *models.BookEntity) error {
	params := sqlc.UpdateBookParams{
		ID:            book.ID,
		Title:         book.Title,
		AuthorID:      convert.StrPtrToNullString(book.AuthorID),
		Description:   convert.StrPtrToNullString(book.Description),
		CoverUrl:      convert.StrPtrToNullString(book.CoverURL),
		Status:        convert.StrPtrToNullString(&book.Status),
		MetadataJson:  convert.StrPtrToNullString(book.MetadataJSON),
		GoogleBooksID: convert.StrPtrToNullString(book.GoogleBooksID),
		AnilistID:     convert.StrPtrToNullString(book.AnilistID),
		OpenlibraryID: convert.StrPtrToNullString(book.OpenLibraryID),
	}
	res, err := r.queries.UpdateBook(ctx, params)
	if err != nil {
		return err
	}
	book.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", book.ID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
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
			cache.BuildKey("book", "id", id),
			cache.BuildKey("chapter", "book", id),
			cache.BuildKey("book_file", "book", id),
			cache.BuildKey("book_file", "count", id),
			constants.CacheKeyLibraryStats,
		)
		for _, chapterID := range chapterIDs {
			_ = r.c.Del(ctx, cache.BuildKey("chapter", "id", chapterID))
		}
		for _, file := range files {
			_ = r.c.Del(
				ctx,
				cache.BuildKey("book_file", "id", file.ID),
				cache.BuildKey("book_file", "book", file.BookID),
				cache.BuildKey("book_file", "count", file.BookID),
			)
		}
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookFileAllPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookFileDupesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookTrackerMapPattern)
		_ = r.c.DelByPattern(context.Background(), "podcasts:*")
	}
	return nil
}

func (r *bookDBRepository) GetBooksByIDs(ctx context.Context, ids []string) ([]*models.BookEntity, error) {
	if len(ids) == 0 {
		return []*models.BookEntity{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("book", "id", id)
	}

	booksByID := make(map[string]*models.BookEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))
	missingKeys := make([]string, 0, len(ids))

	if r.c != nil && !r.inTx {
		cachedObjects := r.c.MGetObjects(keys...)
		for i, obj := range cachedObjects {
			if book, ok := obj.(*models.BookEntity); ok && book != nil {
				booksByID[book.ID] = book
				continue
			}
			missingIDs = append(missingIDs, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIDs = ids
		missingKeys = keys
	}

	if len(missingIDs) > 0 {
		sortedIDs := append([]string(nil), missingIDs...)
		sort.Strings(sortedIDs)
		sfgKey := "books:ids:" + strings.Join(sortedIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.Book, error) {
				return r.queries.GetBooksByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			missingMap := make(map[string]*models.BookEntity, len(rows))
			missingEntities := make([]*models.BookEntity, len(rows))
			for i, row := range rows {
				book := (&models.BookEntity{}).FromSqlc(row)
				missingMap[book.ID] = book
				missingEntities[i] = book
			}
			r.enrichBookEntitiesForCache(ctx, missingEntities)
			return missingMap, nil
		})
		if err != nil {
			return nil, err
		}
		missingMap := v.(map[string]*models.BookEntity)

		for _, book := range missingMap {
			booksByID[book.ID] = book
		}

		if r.c != nil && !r.inTx {
			missingToCache := make(map[string]any, len(missingMap))
			for _, id := range missingIDs {
				if book, ok := missingMap[id]; ok {
					missingToCache[cache.BuildKey("book", "id", id)] = book
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSetObjects(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	ordered := orderBooks(ids, booksByID)
	// Shared cache objects: shallow-copy each entity so callers can assign
	// fields (enrichment, metadata edits) without mutating the cache entries.
	for i, book := range ordered {
		if book != nil {
			out := *book
			ordered[i] = &out
		}
	}
	return ordered, nil
}

func (r *bookDBRepository) enrichBookEntitiesForCache(ctx context.Context, books []*models.BookEntity) {
	if len(books) == 0 {
		return
	}

	bookIDs := make([]string, 0, len(books))
	authorIDMap := make(map[string]bool)
	authorIDs := make([]string, 0, len(books))

	for _, book := range books {
		if book == nil {
			continue
		}
		bookIDs = append(bookIDs, book.ID)
		if book.AuthorID != nil && *book.AuthorID != "" {
			if !authorIDMap[*book.AuthorID] {
				authorIDMap[*book.AuthorID] = true
				authorIDs = append(authorIDs, *book.AuthorID)
			}
		}
	}

	filesByBookID := make(map[string][]*models.BookFileEntity)
	if len(bookIDs) > 0 {
		if files, err := r.GetFilesByBookIDs(ctx, bookIDs); err == nil {
			for _, f := range files {
				if f != nil {
					filesByBookID[f.BookID] = append(filesByBookID[f.BookID], f)
				}
			}
		}
	}

	authorNameByID := make(map[string]string)
	if len(authorIDs) > 0 {
		if authors, err := r.GetAuthorsByIDs(ctx, authorIDs); err == nil {
			for _, a := range authors {
				if a != nil && a.Name != "" {
					authorNameByID[a.ID] = a.Name
				}
			}
		}
	}

	for _, book := range books {
		if book == nil {
			continue
		}
		if files, ok := filesByBookID[book.ID]; ok {
			book.Files = files
		} else if book.Files == nil {
			book.Files = []*models.BookFileEntity{}
		}
		if book.AuthorID != nil && *book.AuthorID != "" {
			if name, ok := authorNameByID[*book.AuthorID]; ok {
				authorName := name
				book.AuthorName = &authorName
				continue
			}
		}
		if book.AuthorName == nil && book.MetadataJSON != nil && *book.MetadataJSON != "" {
			var meta struct {
				Creator  string   `json:"creator"`
				Creators []string `json:"creators"`
			}
			if err := jsonx.UnmarshalString(*book.MetadataJSON, &meta); err == nil {
				authorName := strings.TrimSpace(meta.Creator)
				if authorName == "" && len(meta.Creators) > 0 {
					authorName = strings.Join(meta.Creators, ", ")
				}
				if authorName != "" {
					book.AuthorName = &authorName
				}
			}
		}
	}
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

func (r *bookDBRepository) BulkUpdateBookLibrary(ctx context.Context, bookIDs []string, libraryID string) error {
	if len(bookIDs) == 0 {
		return nil
	}
	err := r.queries.BulkUpdateBookLibrary(ctx, sqlc.BulkUpdateBookLibraryParams{
		LibraryID: libraryID,
		BookIds:   bookIDs,
	})
	if err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		delKeys := make([]string, 0, len(bookIDs)+1)
		delKeys = append(delKeys, constants.CacheKeyLibraryStats)
		for _, id := range bookIDs {
			delKeys = append(delKeys, cache.BuildKey("book", "id", id))
		}
		_ = r.c.Del(ctx, delKeys...)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
	}
	return nil
}

func (r *bookDBRepository) BulkDeleteBooks(ctx context.Context, bookIDs []string) error {
	if len(bookIDs) == 0 {
		return nil
	}
	_ = r.queries.BulkDeleteBookFiles(ctx, bookIDs)
	_ = r.queries.BulkDeleteBookChapters(ctx, bookIDs)
	_ = r.queries.BulkDeleteBookTags(ctx, bookIDs)
	err := r.queries.BulkDeleteBooks(ctx, bookIDs)
	if err != nil {
		return err
	}
	if r.c != nil {
		delKeys := make([]string, 0, len(bookIDs)*4+1)
		delKeys = append(delKeys, constants.CacheKeyLibraryStats)
		for _, id := range bookIDs {
			delKeys = append(delKeys,
				cache.BuildKey("book", "id", id),
				cache.BuildKey("chapter", "book", id),
				cache.BuildKey("book_file", "book", id),
				cache.BuildKey("book_file", "count", id),
			)
		}
		_ = r.c.Del(ctx, delKeys...)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyChapterPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookFilePattern)
		// book_tracker_mappings.book_id is ON DELETE CASCADE; a stale mapping would make
		// tracker sync push progress for a book that no longer exists.
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookTrackerMapPattern)
		_ = r.c.DelByPattern(context.Background(), "podcasts:*")
	}
	return nil
}

func ftsBookMatchQuery(searchText any) (string, bool) {
	term, ok := searchText.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(term)
	if len([]rune(trimmed)) == 0 {
		return "", false
	}

	rawTokens := strings.Fields(trimmed)
	var tokens []string
	for _, tok := range rawTokens {
		cleaned := strings.Map(func(r rune) rune {
			if r == '"' || r == '*' || r == ':' || r == '^' || r == '(' || r == ')' || r == '{' || r == '}' {
				return -1
			}
			return r
		}, tok)
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			tokens = append(tokens, `"`+strings.ReplaceAll(cleaned, `"`, `""`)+`"*`)
		}
	}

	if len(tokens) == 0 {
		return "", false
	}

	return strings.Join(tokens, " "), true
}

func (r *bookDBRepository) ListBookIDsByLibrary(ctx context.Context, libraryID string, limit int64) ([]string, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	cacheKey := cache.BuildKey("book_ids", "library", libraryID, limit)
	if r.c != nil && !r.inTx {
		var cachedIDs []string
		if err := r.c.Get(ctx, cacheKey, &cachedIDs); err == nil {
			return cachedIDs, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		ids, err := r.queries.ListBookIDsByLibrary(ctx, sqlc.ListBookIDsByLibraryParams{
			LibraryID: libraryID,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}
