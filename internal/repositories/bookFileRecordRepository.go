package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

func (r *bookDBRepository) CreateBookFile(ctx context.Context, params BookFileRecordParams) error {
	file, err := r.queries.CreateBookFile(ctx, sqlc.CreateBookFileParams{
		ID:        params.ID,
		BookID:    params.BookID,
		Path:      params.Path,
		Format:    params.Format,
		SizeBytes: params.SizeBytes,
		ModTime:   params.ModTime,
		Hash:      convert.StrPtrToNullString(params.Hash),
		State:     convert.StrPtrToNullString(params.State),
	})
	if err != nil {
		return err
	}
	if r.c != nil {
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
	return nil
}

func (r *bookDBRepository) UpsertBookFile(ctx context.Context, params BookFileRecordParams) error {
	file, err := r.queries.UpsertBookFile(ctx, sqlc.UpsertBookFileParams{
		ID:        params.ID,
		BookID:    params.BookID,
		Path:      params.Path,
		Format:    params.Format,
		SizeBytes: params.SizeBytes,
		ModTime:   params.ModTime,
		Hash:      convert.StrPtrToNullString(params.Hash),
		State:     convert.StrPtrToNullString(params.State),
	})
	if err != nil {
		return err
	}
	if r.c != nil {
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
	return nil
}

func (r *bookDBRepository) GetFilesByBookId(ctx context.Context, bookID string) ([]*models.BookFileEntity, error) {
	key := fmt.Sprintf("book_file:book:%s", bookID)
	if r.c != nil && !r.inTx {
		var files []*models.BookFileEntity
		if err := r.c.Get(ctx, key, &files); err == nil {
			return files, nil
		}
	}
	files, err := r.queries.GetFilesByBookId(ctx, bookID)
	if err != nil {
		return nil, err
	}
	result := (&models.BookFileEntities{}).FromSqlc(files)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
		for _, file := range result {
			_ = r.c.Set(ctx, fmt.Sprintf("book_file:id:%s", file.ID), file, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, fmt.Sprintf("book_file:path:%s", file.Path), file, constants.NormalCacheDuration)
		}
	}
	return result, nil
}

func (r *bookDBRepository) GetFilesByBookIDs(ctx context.Context, bookIDs []string) ([]*models.BookFileEntity, error) {
	if len(bookIDs) == 0 {
		return []*models.BookFileEntity{}, nil
	}
	rows, err := r.queries.GetFilesByBookIDs(ctx, bookIDs)
	if err != nil {
		return nil, err
	}
	return (&models.BookFileEntities{}).FromSqlc(rows), nil
}

func (r *bookDBRepository) GetBookFileByPath(ctx context.Context, path string) (*models.BookFileEntity, error) {
	key := fmt.Sprintf("book_file:path:%s", path)
	if r.c != nil && !r.inTx {
		var file models.BookFileEntity
		if err := r.c.Get(ctx, key, &file); err == nil {
			return &file, nil
		}
	}
	file, err := r.queries.GetBookFileByPath(ctx, path)
	if err != nil {
		return nil, err
	}
	result := (&models.BookFileEntity{}).FromSqlc(file)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, fmt.Sprintf("book_file:id:%s", result.ID), result, constants.NormalCacheDuration)
		_ = r.c.Set(ctx, fmt.Sprintf("book_file:path:%s", result.Path), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) GetBookFileById(ctx context.Context, id string) (*models.BookFileEntity, error) {
	key := fmt.Sprintf("book_file:id:%s", id)
	if r.c != nil && !r.inTx {
		var file models.BookFileEntity
		if err := r.c.Get(ctx, key, &file); err == nil {
			return &file, nil
		}
	}
	file, err := r.queries.GetBookFileById(ctx, id)
	if err != nil {
		return nil, err
	}
	result := (&models.BookFileEntity{}).FromSqlc(file)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, fmt.Sprintf("book_file:id:%s", result.ID), result, constants.NormalCacheDuration)
		_ = r.c.Set(ctx, fmt.Sprintf("book_file:path:%s", result.Path), result, constants.NormalCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) UpdateBookFileHash(ctx context.Context, id string, hash string) error {
	file, _ := r.queries.GetBookFileById(ctx, id)
	if err := r.queries.UpdateFileHash(ctx, sqlc.UpdateFileHashParams{
		Hash: sql.NullString{String: hash, Valid: hash != ""},
		ID:   id,
	}); err != nil {
		return err
	}
	if r.c != nil {
		if file.ID != "" {
			_ = r.c.Del(
				ctx,
				fmt.Sprintf("book_file:id:%s", file.ID),
				fmt.Sprintf("book_file:path:%s", file.Path),
				fmt.Sprintf("book_file:book:%s", file.BookID),
				fmt.Sprintf("book_file:count:%s", file.BookID),
				"book_file:all",
				"book_file:duplicates",
			)
		} else {
			_ = r.c.Del(ctx, fmt.Sprintf("book_file:id:%s", id), "book_file:all", "book_file:duplicates")
		}
	}
	return nil
}

func (r *bookDBRepository) GetDuplicateFiles(ctx context.Context) ([]*models.DuplicateFileEntity, error) {
	key := "book_file:duplicates"
	if r.c != nil && !r.inTx {
		var rows []*models.DuplicateFileEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.GetDuplicateFiles(ctx)
	if err != nil {
		return nil, err
	}
	result := (&models.DuplicateFileEntities{}).FromSqlc(rows)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) ListAllFiles(ctx context.Context) ([]*models.FileRefEntity, error) {
	key := "book_file:all"
	if r.c != nil && !r.inTx {
		var rows []*models.FileRefEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.ListAllFiles(ctx)
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
	file, _ := r.queries.GetBookFileById(ctx, id)
	if err := r.queries.DeleteFile(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		if file.ID != "" {
			_ = r.c.Del(
				ctx,
				fmt.Sprintf("book_file:id:%s", file.ID),
				fmt.Sprintf("book_file:path:%s", file.Path),
				fmt.Sprintf("book_file:book:%s", file.BookID),
				fmt.Sprintf("book_file:count:%s", file.BookID),
				"book_file:all",
				"book_file:duplicates",
			)
		} else {
			_ = r.c.Del(ctx, fmt.Sprintf("book_file:id:%s", id), "book_file:all", "book_file:duplicates")
		}
	}
	return nil
}

func (r *bookDBRepository) CountFilesForBook(ctx context.Context, bookID string) (int64, error) {
	key := fmt.Sprintf("book_file:count:%s", bookID)
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
