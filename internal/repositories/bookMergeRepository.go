package repositories

import (
	"context"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
)

// ListAllTitleAuthor returns every book's id/title/author for the fuzzy duplicate scan.
func (r *bookDBRepository) ListAllTitleAuthor(ctx context.Context) ([]*models.BookTitleAuthorEntity, error) {
	key := cache.BuildKey("book", "title_author_all")
	if r.c != nil && !r.inTx {
		var rows []*models.BookTitleAuthorEntity
		if err := r.c.Get(ctx, key, &rows); err == nil {
			return rows, nil
		}
	}
	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.ListBooksTitleAuthor(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.BookTitleAuthorEntity, len(rows))
		for i, row := range rows {
			result[i] = (&models.BookTitleAuthorEntity{}).FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.BookTitleAuthorEntity), nil
}

// MergeBookData re-points every row that references the source book to the target, folds counters where rows collide, and deletes the source rows.
func (r *bookDBRepository) MergeBookData(ctx context.Context, sourceID string, targetID string) error {
	params := func() sqlc.MergeBookTagsParams {
		return sqlc.MergeBookTagsParams{SourceID: sourceID, TargetID: targetID}
	}
	type pair struct {
		merge func() error
		del   func() error
	}
	steps := []pair{
		{merge: func() error {
			return r.queries.MergeChapters(ctx, sqlc.MergeChaptersParams{SourceID: sourceID, TargetID: targetID})
		}},
		{merge: func() error {
			return r.queries.MergeHighlights(ctx, sqlc.MergeHighlightsParams{SourceID: sourceID, TargetID: targetID})
		}},
		{merge: func() error {
			return r.queries.MergeFTSChapters(ctx, sqlc.MergeFTSChaptersParams{SourceID: sourceID, TargetID: targetID})
		}},

		{merge: func() error { return r.queries.MergeBookTags(ctx, params()) }, del: func() error { return r.queries.DeleteBookTags(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookSeries(ctx, sqlc.MergeBookSeriesParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookSeries(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookPublishers(ctx, sqlc.MergeBookPublishersParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookPublishers(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookLanguages(ctx, sqlc.MergeBookLanguagesParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookLanguages(ctx, sourceID) }},

		{merge: func() error {
			return r.queries.MergeBookFilesRest(ctx, sqlc.MergeBookFilesRestParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookFiles(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeCollectionBooks(ctx, sqlc.MergeCollectionBooksParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteCollectionBooks(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeReadListBooks(ctx, sqlc.MergeReadListBooksParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteReadListBooks(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookmarks(ctx, sqlc.MergeBookmarksParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookmarks(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookReviews(ctx, sqlc.MergeBookReviewsParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookReviews(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookContentWarnings(ctx, sqlc.MergeBookContentWarningsParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookContentWarnings(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeReadingProgress(ctx, sqlc.MergeReadingProgressParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteReadingProgress(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookTrackerMappings(ctx, sqlc.MergeBookTrackerMappingsParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookTrackerMappings(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeKoboSyncedBooks(ctx, sqlc.MergeKoboSyncedBooksParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteKoboSyncedBooksByBook(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeBookShareEvents(ctx, sqlc.MergeBookShareEventsParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteBookShareEvents(ctx, sourceID) }},
		{merge: func() error {
			return r.queries.MergeAudiobookChapters(ctx, sqlc.MergeAudiobookChaptersParams{SourceID: sourceID, TargetID: targetID})
		}, del: func() error { return r.queries.DeleteAudiobookChapters(ctx, sourceID) }},
	}
	for _, step := range steps {
		if step.merge != nil {
			if err := step.merge(); err != nil {
				return err
			}
		}
		if step.del != nil {
			if err := step.del(); err != nil {
				return err
			}
		}
	}

	if err := r.queries.MergeReadingSessionsRest(ctx, sqlc.MergeReadingSessionsRestParams{SourceID: sourceID, TargetID: targetID}); err != nil {
		return err
	}
	if err := r.queries.FoldReadingSessions(ctx, sqlc.FoldReadingSessionsParams{SourceID: sourceID, TargetID: targetID}); err != nil {
		return err
	}
	if err := r.queries.DeleteReadingSessions(ctx, sourceID); err != nil {
		return err
	}

	for _, ensure := range []func() error{
		func() error { return r.queries.EnsureBookReadStats(ctx, targetID) },
		func() error { return r.queries.EnsureBookDownloadStats(ctx, targetID) },
		func() error { return r.queries.EnsureBookSocialStats(ctx, targetID) },
	} {
		if err := ensure(); err != nil {
			return err
		}
	}
	mergeStats := []func() error{
		func() error {
			return r.queries.MergeBookReadStats(ctx, sqlc.MergeBookReadStatsParams{SourceID: sourceID, TargetID: targetID})
		},
		func() error {
			return r.queries.MergeBookDownloadStats(ctx, sqlc.MergeBookDownloadStatsParams{SourceID: sourceID, TargetID: targetID})
		},
		func() error {
			return r.queries.MergeBookSocialStats(ctx, sqlc.MergeBookSocialStatsParams{SourceID: sourceID, TargetID: targetID})
		},
	}
	for _, m := range mergeStats {
		if err := m(); err != nil {
			return err
		}
	}
	deleteStats := []func() error{
		func() error { return r.queries.DeleteBookReadStats(ctx, sourceID) },
		func() error { return r.queries.DeleteBookDownloadStats(ctx, sourceID) },
		func() error { return r.queries.DeleteBookSocialStats(ctx, sourceID) },
	}
	for _, d := range deleteStats {
		if err := d(); err != nil {
			return err
		}
	}

	if r.c != nil {
		for _, key := range []string{
			cache.BuildKey("book", "id", targetID),
			cache.BuildKey("chapter", "book", targetID),
		} {
			_ = r.c.Del(ctx, key)
		}
		_ = r.c.Del(ctx, cache.BuildKey("book", "title_author_all"))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("book_file", "book", targetID))
	}
	return nil
}
