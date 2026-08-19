package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

const (
	metadataAlphaOther = "#"
	dStrokeUpper       = "Đ"
	dStrokeLower       = "đ"
)

type MetadataFacetFilter struct {
	Cursor     string
	Limit      int64
	Search     string
	Alpha      string
	LibraryIDs []string
}

func (f MetadataFacetFilter) cacheKey(facet string) string {
	return cache.BuildKey("metadata", facet, f.Cursor, f.Limit, f.Search, f.Alpha, f.scopeKey())
}

// BookCount is computed under the caller's readable-library set, so every key holding it must
// carry that set or a wider caller's count leaks to a narrower one.
func (f MetadataFacetFilter) scopeKey() string {
	return strings.Join(f.LibraryIDs, ",")
}

func (f MetadataFacetFilter) libraryScope() string {
	data, err := jsonx.Marshal(f.LibraryIDs)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func (f MetadataFacetFilter) sqlcArgs() (search any, alphaUpper any, alphaLower sql.NullString, alphaOther any, dUpper, dLower sql.NullString, cursorName any, cursorID sql.NullString) {
	if trimmed := strings.TrimSpace(f.Search); trimmed != "" {
		search = trimmed
	}
	switch alpha := strings.TrimSpace(f.Alpha); alpha {
	case "", "All":
	case metadataAlphaOther:
		alphaOther = 1
		dUpper = sql.NullString{String: dStrokeUpper, Valid: true}
		dLower = sql.NullString{String: dStrokeLower, Valid: true}
	case dStrokeUpper, dStrokeLower:
		alphaUpper = dStrokeUpper
		alphaLower = sql.NullString{String: dStrokeLower, Valid: true}
	default:
		upper := strings.ToUpper(alpha)
		alphaUpper = upper
		alphaLower = sql.NullString{String: strings.ToLower(alpha), Valid: true}
	}
	if f.Cursor != "" {
		if parts := convert.DecodeCursor(f.Cursor); len(parts) == 2 {
			cursorName = parts[0]
			cursorID = sql.NullString{String: parts[1], Valid: true}
		}
	}
	return
}

func (r *bookDBRepository) CreateAuthor(ctx context.Context, author *models.AuthorEntity) error {
	params := sqlc.CreateAuthorParams{
		ID:   author.ID,
		Name: author.Name,
		Bio:  convert.StrPtrToNullString(author.Bio),
	}
	res, err := r.queries.CreateAuthor(ctx, params)
	if err != nil {
		return err
	}
	author.CreatedAt = res.CreatedAt.Time
	author.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("author", "name", author.Name), author, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) GetAuthorByName(ctx context.Context, name string) (*models.AuthorEntity, error) {
	key := cache.BuildKey("author", "name", name)
	if r.c != nil && !r.inTx {
		var author models.AuthorEntity
		if err := r.c.Get(ctx, key, &author); err == nil {
			return &author, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetAuthorByName(ctx, name)
		if err != nil {
			return nil, err
		}
		author := (&models.AuthorEntity{}).FromSqlc(res)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, author, constants.NormalCacheDuration)
		}
		return author, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.AuthorEntity), nil
}

func (r *bookDBRepository) GetAuthorByID(ctx context.Context, id string) (*models.AuthorEntity, error) {
	key := cache.BuildKey("author", "id", id)
	if r.c != nil && !r.inTx {
		var author models.AuthorEntity
		if err := r.c.Get(ctx, key, &author); err == nil {
			return &author, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetAuthorById(ctx, id)
		if err != nil {
			return nil, err
		}
		author := (&models.AuthorEntity{}).FromSqlc(res)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, author, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("author", "name", author.Name), author, constants.NormalCacheDuration)
		}
		return author, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.AuthorEntity), nil
}

func (r *bookDBRepository) GetAuthorsByIDs(ctx context.Context, ids []string) ([]*models.AuthorEntity, error) {
	if len(ids) == 0 {
		return []*models.AuthorEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("author", "id", id)
	}

	authors := make([]*models.AuthorEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var author models.AuthorEntity
				if err := jsonx.Unmarshal(bytes, &author); err == nil {
					authors = append(authors, &author)
					continue
				}
			}
			missingIds = append(missingIds, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIds = ids
		missingKeys = keys
	}

	if len(missingIds) > 0 {
		rows, err := queryInChunks(missingIds, func(chunk []string) ([]sqlc.Author, error) {
			return r.queries.GetAuthorsByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}
		missingMap := make(map[string]*models.AuthorEntity)
		for _, row := range rows {
			author := (&models.AuthorEntity{}).FromSqlc(row)
			missingMap[author.ID] = author
			authors = append(authors, author)
		}

		if r.c != nil && !r.inTx {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if a, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = a
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	authorMap := make(map[string]*models.AuthorEntity)
	for _, a := range authors {
		authorMap[a.ID] = a
	}
	ordered := make([]*models.AuthorEntity, 0, len(ids))
	for _, id := range ids {
		if a, ok := authorMap[id]; ok {
			ordered = append(ordered, a)
		}
	}

	return ordered, nil
}

func (r *bookDBRepository) CreateTag(ctx context.Context, tag *models.TagEntity) error {
	params := sqlc.CreateTagParams{
		ID:   tag.ID,
		Name: tag.Name,
	}
	res, err := r.queries.CreateTag(ctx, params)
	if err != nil {
		return err
	}
	tag.CreatedAt = res.CreatedAt.Time
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("tag", "name", tag.Name), tag, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) GetTagByName(ctx context.Context, name string) (*models.TagEntity, error) {
	key := cache.BuildKey("tag", "name", name)
	if r.c != nil && !r.inTx {
		var tag models.TagEntity
		if err := r.c.Get(ctx, key, &tag); err == nil {
			return &tag, nil
		}
	}
	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetTagByName(ctx, name)
		if err != nil {
			return nil, err
		}
		tag := (&models.TagEntity{}).FromSqlc(res)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, tag, constants.NormalCacheDuration)
		}
		return tag, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.TagEntity), nil
}

func (r *bookDBRepository) AddBookTag(ctx context.Context, bookID, tagID string) error {
	if err := r.queries.AddBookTag(ctx, sqlc.AddBookTagParams{BookID: bookID, TagID: tagID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) GetSeriesByName(ctx context.Context, name string) (*models.SeriesEntity, error) {
	key := cache.BuildKey("series", "name", name)
	if r.c != nil && !r.inTx {
		var series models.SeriesEntity
		if err := r.c.Get(ctx, key, &series); err == nil {
			return &series, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		series, err := r.queries.GetSeriesByName(ctx, name)
		if err != nil {
			return nil, err
		}
		result := (&models.SeriesEntity{}).FromSqlc(series)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.SeriesEntity), nil
}

func (r *bookDBRepository) CreateSeries(ctx context.Context, series *models.SeriesEntity) error {
	res, err := r.queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		ID:   series.ID,
		Name: series.Name,
	})
	if err != nil {
		return err
	}
	series.CreatedAt = res.CreatedAt.Time
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("series", "name", series.Name), series, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) LinkBookSeries(ctx context.Context, bookID, seriesID string, seriesIndex *string) error {
	if err := r.queries.LinkBookSeries(ctx, sqlc.LinkBookSeriesParams{
		BookID:      bookID,
		SeriesID:    seriesID,
		SeriesIndex: convert.StrPtrToNullString(seriesIndex),
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
	}
	return nil
}

func (r *bookDBRepository) ClearBookSeries(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookSeries(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaSeriesPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKomgaBookSeriesPattern)
	}
	return nil
}

func (r *bookDBRepository) GetBookSeries(ctx context.Context, bookID string) ([]*models.BookSeriesEntity, error) {
	key := cache.BuildKey("book_series", "book", bookID)
	if r.c != nil && !r.inTx {
		var cached []*models.BookSeriesEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.GetBookSeries(ctx, bookID)
		if err != nil {
			return nil, err
		}
		result := models.BookSeriesFromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.BookSeriesEntity), nil
}

func (r *bookDBRepository) GetNextBookInSeries(ctx context.Context, seriesID, currentBookID string) (*models.NextInSeriesEntity, error) {
	key := cache.BuildKey("book_series", "next", seriesID, currentBookID)
	if r.c != nil && !r.inTx {
		var cached models.NextInSeriesEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetNextBookInSeries(ctx, sqlc.GetNextBookInSeriesParams{
			SeriesID:      seriesID,
			CurrentBookID: currentBookID,
		})
		if err != nil {
			return nil, err
		}
		result := &models.NextInSeriesEntity{
			SeriesID:    seriesID,
			BookID:      row.ID,
			LibraryID:   row.LibraryID,
			Title:       row.Title,
			CoverURL:    convert.NullStringToStrPtr(row.CoverUrl),
			SeriesIndex: convert.NullStringToStrPtr(row.SeriesIndex),
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.NextInSeriesEntity), nil
}

func (r *bookDBRepository) GetPublisherByName(ctx context.Context, name string) (*models.PublisherEntity, error) {
	key := cache.BuildKey("publisher", "name", name)
	if r.c != nil && !r.inTx {
		var publisher models.PublisherEntity
		if err := r.c.Get(ctx, key, &publisher); err == nil {
			return &publisher, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		publisher, err := r.queries.GetPublisherByName(ctx, name)
		if err != nil {
			return nil, err
		}
		result := (&models.PublisherEntity{}).FromSqlc(publisher)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.PublisherEntity), nil
}

func (r *bookDBRepository) CreatePublisher(ctx context.Context, publisher *models.PublisherEntity) error {
	res, err := r.queries.CreatePublisher(ctx, sqlc.CreatePublisherParams{
		ID:   publisher.ID,
		Name: publisher.Name,
	})
	if err != nil {
		return err
	}
	publisher.CreatedAt = res.CreatedAt.Time
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("publisher", "name", publisher.Name), publisher, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) LinkBookPublisher(ctx context.Context, bookID, publisherID string) error {
	if err := r.queries.LinkBookPublisher(ctx, sqlc.LinkBookPublisherParams{
		BookID:      bookID,
		PublisherID: publisherID,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) ClearBookPublishers(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookPublishers(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) GetLanguageByName(ctx context.Context, name string) (*models.LanguageEntity, error) {
	key := cache.BuildKey("language", "name", name)
	if r.c != nil && !r.inTx {
		var language models.LanguageEntity
		if err := r.c.Get(ctx, key, &language); err == nil {
			return &language, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		language, err := r.queries.GetLanguageByName(ctx, name)
		if err != nil {
			return nil, err
		}
		result := (&models.LanguageEntity{}).FromSqlc(language)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.LanguageEntity), nil
}

func (r *bookDBRepository) CreateLanguage(ctx context.Context, language *models.LanguageEntity) error {
	res, err := r.queries.CreateLanguage(ctx, sqlc.CreateLanguageParams{
		ID:   language.ID,
		Name: language.Name,
	})
	if err != nil {
		return err
	}
	language.CreatedAt = res.CreatedAt.Time
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("language", "name", language.Name), language, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) LinkBookLanguage(ctx context.Context, bookID, languageID string) error {
	if err := r.queries.LinkBookLanguage(ctx, sqlc.LinkBookLanguageParams{
		BookID:     bookID,
		LanguageID: languageID,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) ClearBookLanguages(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookLanguages(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) ClearBookTags(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookTags(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), constants.CacheKeyLibraryStats)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookSearchPattern)
	}
	return nil
}

func (r *bookDBRepository) ListAuthorsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("authors")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "author", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, alphaUpper, alphaLower, alphaOther, dUpper, dLower, cursorName, cursorID := filter.sqlcArgs()
		rows, err := r.queries.ListAuthorsWithCount(ctx, sqlc.ListAuthorsWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search, AlphaUpper: alphaUpper, AlphaLower: alphaLower, AlphaOther: alphaOther,
			DstrokeUpper: dUpper, DstrokeLower: dLower,
			CursorName: cursorName, CursorID: cursorID, Limit: filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "author", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) ListSeriesWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("series")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "series", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, alphaUpper, alphaLower, alphaOther, dUpper, dLower, cursorName, cursorID := filter.sqlcArgs()
		rows, err := r.queries.ListSeriesWithCount(ctx, sqlc.ListSeriesWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search, AlphaUpper: alphaUpper, AlphaLower: alphaLower, AlphaOther: alphaOther,
			DstrokeUpper: dUpper, DstrokeLower: dLower,
			CursorName: cursorName, CursorID: cursorID, Limit: filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount, CoverURL: row.CoverUrl.String}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "series", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) ListPublishersWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("publishers")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "publisher", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, alphaUpper, alphaLower, alphaOther, dUpper, dLower, cursorName, cursorID := filter.sqlcArgs()
		rows, err := r.queries.ListPublishersWithCount(ctx, sqlc.ListPublishersWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search, AlphaUpper: alphaUpper, AlphaLower: alphaLower, AlphaOther: alphaOther,
			DstrokeUpper: dUpper, DstrokeLower: dLower,
			CursorName: cursorName, CursorID: cursorID, Limit: filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "publisher", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) ListLanguagesWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("languages")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "language", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, alphaUpper, alphaLower, alphaOther, dUpper, dLower, cursorName, cursorID := filter.sqlcArgs()
		rows, err := r.queries.ListLanguagesWithCount(ctx, sqlc.ListLanguagesWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search, AlphaUpper: alphaUpper, AlphaLower: alphaLower, AlphaOther: alphaOther,
			DstrokeUpper: dUpper, DstrokeLower: dLower,
			CursorName: cursorName, CursorID: cursorID, Limit: filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "language", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) ListTagsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("tags")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "tag", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, alphaUpper, alphaLower, alphaOther, dUpper, dLower, cursorName, cursorID := filter.sqlcArgs()
		rows, err := r.queries.ListTagsWithCount(ctx, sqlc.ListTagsWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search, AlphaUpper: alphaUpper, AlphaLower: alphaLower, AlphaOther: alphaOther,
			DstrokeUpper: dUpper, DstrokeLower: dLower,
			CursorName: cursorName, CursorID: cursorID, Limit: filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "tag", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) ListFormatsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("formats")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "format", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, alphaUpper, alphaLower, alphaOther, dUpper, dLower, cursorName, cursorID := filter.sqlcArgs()
		rows, err := r.queries.ListFormatsWithCount(ctx, sqlc.ListFormatsWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search, AlphaUpper: alphaUpper, AlphaLower: alphaLower, AlphaOther: alphaOther,
			DstrokeUpper: dUpper, DstrokeLower: dLower,
			CursorName: cursorName, CursorID: cursorID, Limit: filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "format", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) ListRatingsWithCount(ctx context.Context, filter MetadataFacetFilter) ([]*models.MetadataCountEntity, error) {
	key := filter.cacheKey("ratings")
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "rating", filter.scopeKey(), ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		search, _, _, _, _, _, _, _ := filter.sqlcArgs()
		rows, err := r.queries.ListRatingsWithCount(ctx, sqlc.ListRatingsWithCountParams{
			LibraryIds: filter.libraryScope(),
			Search:     search,
			Limit:      filter.Limit,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(rows))
		result := make([]*models.MetadataCountEntity, len(rows))
		for i, row := range rows {
			name := fmt.Sprint(row.Name)
			result[i] = &models.MetadataCountEntity{ID: row.ID, Name: name, BookCount: row.BookCount}
			ids[i] = row.ID
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheMetadataCountEntities(ctx, "rating", filter.scopeKey(), result)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.MetadataCountEntity), nil
}

func (r *bookDBRepository) getMetadataCountByIDs(ctx context.Context, entityType, scope string, ids []string) ([]*models.MetadataCountEntity, bool) {
	if len(ids) == 0 {
		return []*models.MetadataCountEntity{}, true
	}
	if r.c == nil || r.inTx {
		return nil, false
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.QueryKeyParts(scope, "metadata_count", entityType, "id", id)
	}

	cachedBytes := r.c.MGet(ctx, keys...)
	ordered := make([]*models.MetadataCountEntity, 0, len(ids))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.MetadataCountEntity
		if err := jsonx.Unmarshal(bytes, &entity); err != nil {
			return nil, false
		}
		ordered = append(ordered, &entity)
	}

	return ordered, true
}

func (r *bookDBRepository) cacheMetadataCountEntities(ctx context.Context, entityType, scope string, entities []*models.MetadataCountEntity) {
	if r.c == nil || len(entities) == 0 {
		return
	}
	toCache := make(map[string]any, len(entities))
	for _, entity := range entities {
		toCache[cache.QueryKeyParts(scope, "metadata_count", entityType, "id", entity.ID)] = entity
	}
	_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
}
