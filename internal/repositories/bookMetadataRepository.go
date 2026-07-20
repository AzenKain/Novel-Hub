package repositories

import (
	"context"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/cache"
)

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
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
	res, err := r.queries.GetAuthorByName(ctx, name)
	if err != nil {
		return nil, err
	}
	author := (&models.AuthorEntity{}).FromSqlc(res)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, author, constants.NormalCacheDuration)
	}
	return author, nil
}

func (r *bookDBRepository) GetAuthorByID(ctx context.Context, id string) (*models.AuthorEntity, error) {
	key := cache.BuildKey("author", "id", id)
	if r.c != nil && !r.inTx {
		var author models.AuthorEntity
		if err := r.c.Get(ctx, key, &author); err == nil {
			return &author, nil
		}
	}
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
		rows, err := r.queries.GetAuthorsByIDs(ctx, missingIds)
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
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
	res, err := r.queries.GetTagByName(ctx, name)
	if err != nil {
		return nil, err
	}
	tag := (&models.TagEntity{}).FromSqlc(res)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, tag, constants.NormalCacheDuration)
	}
	return tag, nil
}

func (r *bookDBRepository) AddBookTag(ctx context.Context, bookID, tagID string) error {
	if err := r.queries.AddBookTag(ctx, sqlc.AddBookTagParams{BookID: bookID, TagID: tagID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
	series, err := r.queries.GetSeriesByName(ctx, name)
	if err != nil {
		return nil, err
	}
	result := (&models.SeriesEntity{}).FromSqlc(series)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
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
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookSeries(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookSeries(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetPublisherByName(ctx context.Context, name string) (*models.PublisherEntity, error) {
	key := cache.BuildKey("publisher", "name", name)
	if r.c != nil && !r.inTx {
		var publisher models.PublisherEntity
		if err := r.c.Get(ctx, key, &publisher); err == nil {
			return &publisher, nil
		}
	}
	publisher, err := r.queries.GetPublisherByName(ctx, name)
	if err != nil {
		return nil, err
	}
	result := (&models.PublisherEntity{}).FromSqlc(publisher)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
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
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookPublishers(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookPublishers(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
	language, err := r.queries.GetLanguageByName(ctx, name)
	if err != nil {
		return nil, err
	}
	result := (&models.LanguageEntity{}).FromSqlc(language)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
	}
	return result, nil
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
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
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
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookLanguages(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookLanguages(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookTags(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookTags(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("book", "id", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ListAuthorsWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:authors"
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "author", ids); ok {
				return result, nil
			}
		}
	}
	rows, err := r.queries.ListAuthorsWithCount(ctx)
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
		r.cacheMetadataCountEntities(ctx, "author", result)
	}
	return result, nil
}

func (r *bookDBRepository) ListSeriesWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:series"
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "series", ids); ok {
				return result, nil
			}
		}
	}
	rows, err := r.queries.ListSeriesWithCount(ctx)
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
		r.cacheMetadataCountEntities(ctx, "series", result)
	}
	return result, nil
}

func (r *bookDBRepository) ListPublishersWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:publishers"
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "publisher", ids); ok {
				return result, nil
			}
		}
	}
	rows, err := r.queries.ListPublishersWithCount(ctx)
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
		r.cacheMetadataCountEntities(ctx, "publisher", result)
	}
	return result, nil
}

func (r *bookDBRepository) ListLanguagesWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:languages"
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "language", ids); ok {
				return result, nil
			}
		}
	}
	rows, err := r.queries.ListLanguagesWithCount(ctx)
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
		r.cacheMetadataCountEntities(ctx, "language", result)
	}
	return result, nil
}

func (r *bookDBRepository) ListTagsWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:tags"
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "tag", ids); ok {
				return result, nil
			}
		}
	}
	rows, err := r.queries.ListTagsWithCount(ctx)
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
		r.cacheMetadataCountEntities(ctx, "tag", result)
	}
	return result, nil
}

func (r *bookDBRepository) ListFormatsWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:formats"
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getMetadataCountByIDs(ctx, "format", ids); ok {
				return result, nil
			}
		}
	}
	rows, err := r.queries.ListFormatsWithCount(ctx)
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
		r.cacheMetadataCountEntities(ctx, "format", result)
	}
	return result, nil
}

func (r *bookDBRepository) getMetadataCountByIDs(ctx context.Context, entityType string, ids []string) ([]*models.MetadataCountEntity, bool) {
	if len(ids) == 0 {
		return []*models.MetadataCountEntity{}, true
	}
	if r.c == nil || r.inTx {
		return nil, false
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("metadata_count", entityType, "id", id)
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

func (r *bookDBRepository) cacheMetadataCountEntities(ctx context.Context, entityType string, entities []*models.MetadataCountEntity) {
	if r.c == nil || len(entities) == 0 {
		return
	}
	toCache := make(map[string]any, len(entities))
	for _, entity := range entities {
		toCache[cache.BuildKey("metadata_count", entityType, "id", entity.ID)] = entity
	}
	_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
}
