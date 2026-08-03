package repositories

import (
	"context"
	"database/sql"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

func (r *bookDBRepository) CreateBookFile(ctx context.Context, params sqlc.CreateBookFileParams) error {
	file, err := r.queries.CreateBookFile(ctx, params)
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			cache.BuildKey("book_file", "id", file.ID),
			cache.BuildKey("book_file", "path", file.Path),
			cache.BuildKey("book_file", "book", file.BookID),
			cache.BuildKey("book_file", "count", file.BookID),
		)
		_ = r.c.DelByPattern(context.Background(), "book_file:all*")
		_ = r.c.DelByPattern(context.Background(), "book_file:duplicates*")
		// Format facets (ListFormatsWithCount) read book_files exclusively, and
		// SearchBookIDs filters on it via filter_has_files/filter_has_formats/file_format.
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) UpsertBookFile(ctx context.Context, params sqlc.UpsertBookFileParams) error {
	file, err := r.queries.UpsertBookFile(ctx, params)
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(
			ctx,
			cache.BuildKey("book_file", "id", file.ID),
			cache.BuildKey("book_file", "path", file.Path),
			cache.BuildKey("book_file", "book", file.BookID),
			cache.BuildKey("book_file", "count", file.BookID),
		)
		_ = r.c.DelByPattern(context.Background(), "book_file:all*")
		_ = r.c.DelByPattern(context.Background(), "book_file:duplicates*")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) GetFilesByBookId(ctx context.Context, bookID string) ([]*models.BookFileEntity, error) {
	key := cache.BuildKey("book_file", "book", bookID)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			if result, ok := r.getBookFilesByIDs(ctx, ids); ok {
				return result, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		idRows, err := r.queries.ListFileIDsByBookId(ctx, bookID)
		if err != nil {
			return nil, err
		}

		if len(idRows) == 0 {
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, key, []string{}, constants.ListCacheDuration)
			}
			return []*models.BookFileEntity{}, nil
		}

		rows, err := queryInChunks(idRows, func(chunk []string) ([]sqlc.BookFile, error) {
			return r.queries.GetBookFilesByIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}

		out := (&models.BookFileEntities{}).FromSqlc(rows)

		fileMap := make(map[string]*models.BookFileEntity, len(out))
		for _, entity := range out {
			fileMap[entity.ID] = entity
		}

		// ponytail: append instead of indexing by idRows position — a file deleted
		// between ListFileIDsByBookId and GetBookFilesByIDs makes out shorter than idRows
		ordered := make([]*models.BookFileEntity, 0, len(idRows))
		ids := make([]string, 0, len(idRows))
		for _, id := range idRows {
			if entity, ok := fileMap[id]; ok {
				ordered = append(ordered, entity)
				ids = append(ids, id)
			}
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
			r.cacheBookFileEntities(ctx, ordered)
		}
		return ordered, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.BookFileEntity), nil
}

func (r *bookDBRepository) GetFilesByBookIDs(ctx context.Context, bookIDs []string) ([]*models.BookFileEntity, error) {
	if len(bookIDs) == 0 {
		return []*models.BookFileEntity{}, nil
	}
	keys := make([]string, len(bookIDs))
	for i, id := range bookIDs {
		keys[i] = cache.BuildKey("book_file", "book", id)
	}

	allFiles := make([]*models.BookFileEntity, 0)
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var ids []string
				if err := jsonx.Unmarshal(bytes, &ids); err == nil {
					if result, ok := r.getBookFilesByIDs(ctx, ids); ok {
						allFiles = append(allFiles, result...)
						continue
					}
				}
			}
			missingIds = append(missingIds, bookIDs[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIds = bookIDs
		missingKeys = keys
	}

	if len(missingIds) > 0 {
		rows, err := queryInChunks(missingIds, func(chunk []string) ([]sqlc.BookFile, error) {
			return r.queries.GetFilesByBookIDs(ctx, chunk)
		})
		if err != nil {
			return nil, err
		}

		missingFiles := (&models.BookFileEntities{}).FromSqlc(rows)
		allFiles = append(allFiles, missingFiles...)

		if r.c != nil && !r.inTx {
			missingMap := make(map[string][]string)
			for _, id := range missingIds {
				missingMap[id] = []string{}
			}
			for _, file := range missingFiles {
				missingMap[file.BookID] = append(missingMap[file.BookID], file.ID)
			}

			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				missingToCache[missingKeys[i]] = missingMap[missingId]
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.ListCacheDuration)
			}
			r.cacheBookFileEntities(ctx, missingFiles)
		}
	}

	return allFiles, nil
}

func (r *bookDBRepository) getBookFilesByIDs(ctx context.Context, ids []string) ([]*models.BookFileEntity, bool) {
	if len(ids) == 0 {
		return []*models.BookFileEntity{}, true
	}
	if r.c == nil || r.inTx {
		return nil, false
	}

	cacheKeys := make([]string, len(ids))
	for i, id := range ids {
		cacheKeys[i] = cache.BuildKey("book_file", "id", id)
	}

	cachedBytes := r.c.MGet(ctx, cacheKeys...)
	ordered := make([]*models.BookFileEntity, 0, len(ids))

	for _, bytes := range cachedBytes {
		if len(bytes) == 0 {
			return nil, false
		}
		var entity models.BookFileEntity
		if err := jsonx.Unmarshal(bytes, &entity); err != nil {
			return nil, false
		}
		ordered = append(ordered, &entity)
	}

	return ordered, true
}

func (r *bookDBRepository) cacheBookFileEntities(ctx context.Context, entities []*models.BookFileEntity) {
	if r.c == nil || r.inTx || len(entities) == 0 {
		return
	}
	toCache := make(map[string]any, len(entities)*2)
	for _, entity := range entities {
		toCache[cache.BuildKey("book_file", "id", entity.ID)] = entity
		toCache[cache.BuildKey("book_file", "path", entity.Path)] = entity
	}
	_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
}

func (r *bookDBRepository) GetBookFileByPath(ctx context.Context, path string) (*models.BookFileEntity, error) {
	key := cache.BuildKey("book_file", "path", path)
	if r.c != nil && !r.inTx {
		var file models.BookFileEntity
		if err := r.c.Get(ctx, key, &file); err == nil {
			return &file, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		file, err := r.queries.GetBookFileByPath(ctx, path)
		if err != nil {
			return nil, err
		}
		result := (&models.BookFileEntity{}).FromSqlc(file)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cache.BuildKey("book_file", "id", result.ID), result, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("book_file", "path", result.Path), result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookFileEntity), nil
}

func (r *bookDBRepository) GetBookFileById(ctx context.Context, id string) (*models.BookFileEntity, error) {
	key := cache.BuildKey("book_file", "id", id)
	if r.c != nil && !r.inTx {
		var file models.BookFileEntity
		if err := r.c.Get(ctx, key, &file); err == nil {
			return &file, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		file, err := r.queries.GetBookFileById(ctx, id)
		if err != nil {
			return nil, err
		}
		result := (&models.BookFileEntity{}).FromSqlc(file)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cache.BuildKey("book_file", "id", result.ID), result, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("book_file", "path", result.Path), result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.BookFileEntity), nil
}

func (r *bookDBRepository) UpdateBookFileHash(ctx context.Context, id string, hash string) error {
	file, preErr := r.queries.GetBookFileById(ctx, id)
	if err := r.queries.UpdateFileHash(ctx, sqlc.UpdateFileHashParams{
		Hash: sql.NullString{String: hash, Valid: hash != ""},
		ID:   id,
	}); err != nil {
		return err
	}
	if r.c != nil {
		if preErr == nil && file.ID != "" {
			_ = r.c.Del(
				ctx,
				cache.BuildKey("book_file", "id", file.ID),
				cache.BuildKey("book_file", "path", file.Path),
				cache.BuildKey("book_file", "book", file.BookID),
				cache.BuildKey("book_file", "count", file.BookID),
			)
		} else {
			_ = r.c.DelByPattern(context.Background(), "book_file:*")
		}
		_ = r.c.DelByPattern(context.Background(), "book_file:all*")
		_ = r.c.DelByPattern(context.Background(), "book_file:duplicates*")
	}
	return nil
}

func (r *bookDBRepository) GetDuplicateFiles(ctx context.Context, limit, offset int64) ([]*models.DuplicateFileEntity, error) {
	key := cache.BuildKey("book_file", "duplicates", limit, offset)
	if r.c != nil && !r.inTx {
		var rows []*models.DuplicateFileEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.GetDuplicateFiles(ctx, sqlc.GetDuplicateFilesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	result := (&models.DuplicateFileEntities{}).FromSqlc(rows)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) GetDuplicateFileDetails(ctx context.Context) ([]*models.DuplicateFileDetailEntity, error) {
	rows, err := r.queries.GetDuplicateFileDetails(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.DuplicateFileDetailEntity, len(rows))
	for i, row := range rows {
		result[i] = (&models.DuplicateFileDetailEntity{}).FromSqlc(row)
	}
	return result, nil
}

func (r *bookDBRepository) ListAllFiles(ctx context.Context, limit, offset int64) ([]*models.FileRefEntity, error) {
	key := cache.BuildKey("book_file", "all", limit, offset)
	if r.c != nil && !r.inTx {
		var rows []*models.FileRefEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListAllFiles(ctx, sqlc.ListAllFilesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	result := (&models.FileRefEntities{}).FromSqlc(rows)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) DeleteFile(ctx context.Context, id string) error {
	file, preErr := r.queries.GetBookFileById(ctx, id)
	if err := r.queries.DeleteFile(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		if preErr == nil && file.ID != "" {
			_ = r.c.Del(
				ctx,
				cache.BuildKey("book_file", "id", file.ID),
				cache.BuildKey("book_file", "path", file.Path),
				cache.BuildKey("book_file", "book", file.BookID),
				cache.BuildKey("book_file", "count", file.BookID),
			)
		} else {
			// The pre-read failed, so the path/book/count keys are unknown — sweep the
			// domain rather than leaving book_file:path:<path> pointing at a deleted row.
			_ = r.c.DelByPattern(context.Background(), "book_file:*")
		}
		_ = r.c.DelByPattern(context.Background(), "book_file:all*")
		_ = r.c.DelByPattern(context.Background(), "book_file:duplicates*")
		_ = r.c.DelByPattern(context.Background(), "metadata:*")
		_ = r.c.DelByPattern(context.Background(), "metadata_count:*")
		_ = r.c.DelByPattern(context.Background(), "book:search*")
	}
	return nil
}

func (r *bookDBRepository) CountFilesForBook(ctx context.Context, bookID string) (int64, error) {
	key := cache.BuildKey("book_file", "count", bookID)
	if r.c != nil && !r.inTx {
		var count int64
		if err := r.c.Get(ctx, key, &count); err == nil {
			return count, nil
		}
	}
	count, err := r.queries.CountFilesForBook(ctx, bookID)
	if err != nil {
		return 0, err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, count, constants.ListCacheDuration)
	}
	return count, nil
}
