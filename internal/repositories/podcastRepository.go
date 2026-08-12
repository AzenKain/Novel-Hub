package repositories

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
)

type PodcastRepository interface {
	CreatePodcast(ctx context.Context, id, libraryID, feedURL, title string, description, coverURL, author *string) (*models.PodcastEntity, error)
	GetPodcast(ctx context.Context, id string) (*models.PodcastEntity, error)
	GetPodcastByFeedURL(ctx context.Context, feedURL string) (*models.PodcastEntity, error)
	ListPodcastIDs(ctx context.Context) ([]string, error)
	ListPodcastsWithCounts(ctx context.Context) ([]*models.PodcastWithCountEntity, error)
	UpdatePodcast(ctx context.Context, id string, title string, description, coverURL, author *string, autoDownload bool, lastCheckedAt *time.Time) (*models.PodcastEntity, error)
	DeletePodcast(ctx context.Context, id string) error
	UpsertEpisode(ctx context.Context, id, podcastID, guid, title string, description *string, audioURL string, durationSec *int64, publishedAt *time.Time) (*models.PodcastEpisodeEntity, error)
	ListEpisodeGuids(ctx context.Context, podcastID string) ([]string, error)
	ListEpisodes(ctx context.Context, podcastID string) ([]*models.PodcastEpisodeEntity, error)
	GetEpisode(ctx context.Context, id string) (*models.PodcastEpisodeEntity, error)
	MarkEpisodeDownloaded(ctx context.Context, episodeID string, bookID string) error
	WithTx(tx *sql.Tx) PodcastRepository
}

type podcastRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewPodcastRepository(db *sql.DB, c cache.Cache) PodcastRepository {
	return &podcastRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

func (r *podcastRepository) WithTx(tx *sql.Tx) PodcastRepository {
	if tx == nil {
		return r
	}
	return &podcastRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
		c:       r.c,
		inTx:    true,
		sfg:     &singleflight.Group{},
	}
}

func (r *podcastRepository) CreatePodcast(ctx context.Context, id, libraryID, feedURL, title string, description, coverURL, author *string) (*models.PodcastEntity, error) {
	row, err := r.queries.CreatePodcast(ctx, sqlc.CreatePodcastParams{
		ID:          id,
		LibraryID:   libraryID,
		FeedUrl:     feedURL,
		Title:       title,
		Description: convert.StrPtrToNullString(description),
		CoverUrl:    convert.StrPtrToNullString(coverURL),
		Author:      convert.StrPtrToNullString(author),
		AutoDownload: 0,
		LastCheckedAt: sql.NullTime{},
	})
	if err != nil {
		return nil, err
	}
	result := (&models.PodcastEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("podcasts", "list"))
	}
	return result, nil
}

func (r *podcastRepository) GetPodcast(ctx context.Context, id string) (*models.PodcastEntity, error) {
	key := cache.BuildKey("podcasts", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.PodcastEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetPodcast(ctx, id)
		if err != nil {
			return nil, err
		}
		result := (&models.PodcastEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.PodcastEntity), nil
}

func (r *podcastRepository) GetPodcastByFeedURL(ctx context.Context, feedURL string) (*models.PodcastEntity, error) {
	key := cache.BuildKey("podcasts", "feed_url", feedURL)
	if r.c != nil && !r.inTx {
		var cached models.PodcastEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetPodcastByFeedURL(ctx, feedURL)
		if err != nil {
			if err == sql.ErrNoRows {
				return (*models.PodcastEntity)(nil), nil
			}
			return nil, err
		}
		result := (&models.PodcastEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.PodcastEntity), nil
}

func (r *podcastRepository) ListPodcastIDs(ctx context.Context) ([]string, error) {
	key := cache.BuildKey("podcasts", "ids")
	if r.c != nil && !r.inTx {
		var cached []string
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.queries.ListPodcastIDs(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.NormalCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *podcastRepository) ListPodcastsWithCounts(ctx context.Context) ([]*models.PodcastWithCountEntity, error) {
	key := cache.BuildKey("podcasts", "list")
	if r.c != nil && !r.inTx {
		var cached []*models.PodcastWithCountEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.ListPodcastsWithCounts(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.PodcastWithCountEntity, len(rows))
		for i, row := range rows {
			result[i] = (&models.PodcastWithCountEntity{}).FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.PodcastWithCountEntity), nil
}

func (r *podcastRepository) UpdatePodcast(ctx context.Context, id string, title string, description, coverURL, author *string, autoDownload bool, lastCheckedAt *time.Time) (*models.PodcastEntity, error) {
	row, err := r.queries.UpdatePodcast(ctx, sqlc.UpdatePodcastParams{
		ID:            id,
		Title:         title,
		Description:   convert.StrPtrToNullString(description),
		CoverUrl:      convert.StrPtrToNullString(coverURL),
		Author:        convert.StrPtrToNullString(author),
		AutoDownload:  convert.BoolToInt64(autoDownload),
		LastCheckedAt: convert.TimePtrToNullTime(lastCheckedAt),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.PodcastEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("podcasts", "list"), cache.BuildKey("podcasts", "id", id), cache.BuildKey("podcasts", "feed_url", result.FeedURL))
	}
	return result, nil
}

func (r *podcastRepository) DeletePodcast(ctx context.Context, id string) error {
	if err := r.queries.DeletePodcast(ctx, id); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("podcasts", "list"), cache.BuildKey("podcasts", "id", id), cache.BuildKey("podcasts", "episodes", id), cache.BuildKey("podcasts", "episode_guids", id))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("podcasts", "feed_url", "*"))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("podcasts", "episode", "id", "*"))
	}
	return nil
}

func (r *podcastRepository) UpsertEpisode(ctx context.Context, id, podcastID, guid, title string, description *string, audioURL string, durationSec *int64, publishedAt *time.Time) (*models.PodcastEpisodeEntity, error) {
	row, err := r.queries.UpsertPodcastEpisode(ctx, sqlc.UpsertPodcastEpisodeParams{
		ID:          id,
		PodcastID:   podcastID,
		Guid:        guid,
		Title:       title,
		Description: convert.StrPtrToNullString(description),
		AudioUrl:    audioURL,
		DurationSec: convert.IntPtrToNullInt64(durationSec),
		PublishedAt: convert.TimePtrToNullTime(publishedAt),
	})
	if err != nil {
		return nil, err
	}
	result := (&models.PodcastEpisodeEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Del(ctx, cache.BuildKey("podcasts", "episodes", podcastID), cache.BuildKey("podcasts", "list"), cache.BuildKey("podcasts", "episode", "id", id), cache.BuildKey("podcasts", "episode_guids", podcastID))
	}
	return result, nil
}

func (r *podcastRepository) ListEpisodeGuids(ctx context.Context, podcastID string) ([]string, error) {
	key := cache.BuildKey("podcasts", "episode_guids", podcastID)
	if r.c != nil && !r.inTx {
		var cached []string
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		guids, err := r.queries.ListEpisodeGuidsByPodcast(ctx, podcastID)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, guids, constants.ListCacheDuration)
		}
		return guids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *podcastRepository) ListEpisodes(ctx context.Context, podcastID string) ([]*models.PodcastEpisodeEntity, error) {
	key := cache.BuildKey("podcasts", "episodes", podcastID)
	if r.c != nil && !r.inTx {
		var cached models.PodcastEpisodeEntities
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.queries.ListEpisodesByPodcast(ctx, podcastID)
		if err != nil {
			return nil, err
		}
		result := (&models.PodcastEpisodeEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.PodcastEpisodeEntity), nil
}

func (r *podcastRepository) GetEpisode(ctx context.Context, id string) (*models.PodcastEpisodeEntity, error) {
	key := cache.BuildKey("podcasts", "episode", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.PodcastEpisodeEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.queries.GetPodcastEpisode(ctx, id)
		if err != nil {
			return nil, err
		}
		result := (&models.PodcastEpisodeEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.PodcastEpisodeEntity), nil
}

func (r *podcastRepository) MarkEpisodeDownloaded(ctx context.Context, episodeID string, bookID string) error {
	if err := r.queries.UpdateEpisodeDownloaded(ctx, sqlc.UpdateEpisodeDownloadedParams{
		BookID:    sql.NullString{String: bookID, Valid: true},
		ID:        episodeID,
	}); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(ctx, "podcasts:episodes:*")
		_ = r.c.Del(ctx, cache.BuildKey("podcasts", "list"), cache.BuildKey("podcasts", "episode", "id", episodeID))
	}
	return nil
}