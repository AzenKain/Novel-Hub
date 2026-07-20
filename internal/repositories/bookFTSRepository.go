package repositories

import (
	"context"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
)

func (r *bookDBRepository) SearchFTS(ctx context.Context, query string, limit, offset int64) ([]*models.FTSResultEntity, error) {
	params := sqlc.SearchFTSParams{
		Content: query,
		Limit:   limit,
		Offset:  offset,
	}
	key := cache.QueryKey("fts:search", params)
	if r.c != nil && !r.inTx {
		var rows []*models.FTSResultEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	rows, err := r.queries.SearchFTS(ctx, params)
	if err != nil {
		return nil, err
	}
	result := (&models.FTSResultEntities{}).FromSqlc(rows)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
	}
	return result, nil
}

func (r *bookDBRepository) DeleteFTSBook(ctx context.Context, bookID string) error {
	if err := r.queries.DeleteFTSBook(ctx, bookID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), "fts:search*")
	}
	return nil
}

func (r *bookDBRepository) InsertFTSChapter(ctx context.Context, bookID, chapterID, title, content string) error {
	if err := r.queries.InsertFTSChapter(ctx, sqlc.InsertFTSChapterParams{
		BookID:    bookID,
		ChapterID: chapterID,
		Title:     title,
		Content:   content,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), "fts:search*")
	}
	return nil
}
