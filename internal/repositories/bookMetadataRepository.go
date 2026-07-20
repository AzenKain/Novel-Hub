package repositories

import (
	"context"
	"fmt"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
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
		_ = r.c.Set(ctx, fmt.Sprintf("author:name:%s", author.Name), author, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetAuthorByName(ctx context.Context, name string) (*models.AuthorEntity, error) {
	key := fmt.Sprintf("author:name:%s", name)
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
	key := fmt.Sprintf("author:id:%s", id)
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
		_ = r.c.Set(ctx, fmt.Sprintf("author:name:%s", author.Name), author, constants.NormalCacheDuration)
	}
	return author, nil
}

func (r *bookDBRepository) GetAuthorsByIDs(ctx context.Context, ids []string) ([]*models.AuthorEntity, error) {
	if len(ids) == 0 {
		return []*models.AuthorEntity{}, nil
	}
	rows, err := r.queries.GetAuthorsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	authors := make([]*models.AuthorEntity, len(rows))
	for i, row := range rows {
		authors[i] = (&models.AuthorEntity{}).FromSqlc(row)
	}
	return authors, nil
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
		_ = r.c.Set(ctx, fmt.Sprintf("tag:name:%s", tag.Name), tag, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetTagByName(ctx context.Context, name string) (*models.TagEntity, error) {
	key := fmt.Sprintf("tag:name:%s", name)
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
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetSeriesByName(ctx context.Context, name string) (*models.SeriesEntity, error) {
	key := fmt.Sprintf("series:name:%s", name)
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
		_ = r.c.Set(ctx, fmt.Sprintf("series:name:%s", series.Name), series, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
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
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookSeries(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookSeries(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetPublisherByName(ctx context.Context, name string) (*models.PublisherEntity, error) {
	key := fmt.Sprintf("publisher:name:%s", name)
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
		_ = r.c.Set(ctx, fmt.Sprintf("publisher:name:%s", publisher.Name), publisher, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
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
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookPublishers(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookPublishers(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetLanguageByName(ctx context.Context, name string) (*models.LanguageEntity, error) {
	key := fmt.Sprintf("language:name:%s", name)
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
		_ = r.c.Set(ctx, fmt.Sprintf("language:name:%s", language.Name), language, constants.NormalCacheDuration)
	}
	if r.c != nil {
		_ = r.c.Del(ctx, "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
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
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookLanguages(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookLanguages(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ClearBookTags(ctx context.Context, bookID string) error {
	if err := r.queries.ClearBookTags(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("book:id:%s", bookID), "feature:library_stats")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) ListAuthorsWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:authors"
	if r.c != nil && !r.inTx {
		var rows []*models.MetadataCountEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListAuthorsWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.MetadataCountEntity, len(rows))
	for i, row := range rows {
		result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) ListSeriesWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:series"
	if r.c != nil && !r.inTx {
		var rows []*models.MetadataCountEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListSeriesWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.MetadataCountEntity, len(rows))
	for i, row := range rows {
		result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount, CoverURL: row.CoverUrl.String}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) ListPublishersWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:publishers"
	if r.c != nil && !r.inTx {
		var rows []*models.MetadataCountEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListPublishersWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.MetadataCountEntity, len(rows))
	for i, row := range rows {
		result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) ListLanguagesWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:languages"
	if r.c != nil && !r.inTx {
		var rows []*models.MetadataCountEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListLanguagesWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.MetadataCountEntity, len(rows))
	for i, row := range rows {
		result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) ListTagsWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:tags"
	if r.c != nil && !r.inTx {
		var rows []*models.MetadataCountEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListTagsWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.MetadataCountEntity, len(rows))
	for i, row := range rows {
		result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) ListFormatsWithCount(ctx context.Context) ([]*models.MetadataCountEntity, error) {
	key := "metadata:formats"
	if r.c != nil && !r.inTx {
		var rows []*models.MetadataCountEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListFormatsWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.MetadataCountEntity, len(rows))
	for i, row := range rows {
		result[i] = &models.MetadataCountEntity{ID: row.ID, Name: row.Name, BookCount: row.BookCount}
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}
