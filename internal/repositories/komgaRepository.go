package repositories

import (
	"context"
	"database/sql"
	"strings"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type KomgaRepository interface {
	ListSeries(ctx context.Context, libraryIDs []string, search string, limit, offset int64) ([]*models.KomgaSeriesEntity, error)
	CountSeries(ctx context.Context, libraryIDs []string, search string) (int64, error)
	GetSeriesByIDs(ctx context.Context, seriesIDs []string, libraryIDs []string) ([]*models.KomgaSeriesEntity, error)
	ListSeriesBooks(ctx context.Context, seriesID string, libraryIDs []string) ([]*models.KomgaSeriesBookEntity, error)
	GetBookSeries(ctx context.Context, bookID string) (*models.KomgaBookSeriesRefEntity, error)
	SeriesProgress(ctx context.Context, userID, seriesID string, libraryIDs []string) (*models.KomgaSeriesProgressEntity, error)
	SeriesProgressByIDs(ctx context.Context, userID string, seriesIDs []string, libraryIDs []string) (map[string]*models.KomgaSeriesProgressEntity, error)

	WithTx(tx *sql.Tx) KomgaRepository
}

type komgaRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewKomgaRepository(db sqlc.DBTX, c cache.Cache) KomgaRepository {
	return &komgaRepository{q: sqlc.New(db), c: c, sfg: &singleflight.Group{}}
}

func (r *komgaRepository) WithTx(tx *sql.Tx) KomgaRepository {
	if tx == nil {
		return r
	}
	return &komgaRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sfg: &singleflight.Group{}}
}

func (r *komgaRepository) ListSeries(ctx context.Context, libraryIDs []string, search string, limit, offset int64) ([]*models.KomgaSeriesEntity, error) {
	scope, err := jsonx.MarshalString(libraryIDs)
	if err != nil {
		return nil, err
	}
	key := cache.BuildKey("komga", "series", strings.Join(libraryIDs, ","), search, limit, offset)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetSeriesByIDs(ctx, ids, libraryIDs)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		var searchArg any
		if search != "" {
			searchArg = search
		}
		ids, err := r.q.ListKomgaSeriesIDs(ctx, sqlc.ListKomgaSeriesIDsParams{
			LibraryIds: scope,
			Search:     searchArg,
			Limit:      limit,
			Offset:     offset,
		})
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []string{}
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetSeriesByIDs(ctx, v.([]string), libraryIDs)
}

func (r *komgaRepository) GetSeriesByIDs(ctx context.Context, seriesIDs []string, libraryIDs []string) ([]*models.KomgaSeriesEntity, error) {
	if len(seriesIDs) == 0 {
		return []*models.KomgaSeriesEntity{}, nil
	}
	scopeKey := strings.Join(libraryIDs, ",")
	keys := make([]string, len(seriesIDs))
	for i, id := range seriesIDs {
		keys[i] = cache.BuildKey("komga", "series", "id", id, scopeKey)
	}

	found := make(map[string]*models.KomgaSeriesEntity, len(seriesIDs))
	missingIDs := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		for i, bytes := range r.c.MGet(ctx, keys...) {
			if len(bytes) > 0 {
				var entity models.KomgaSeriesEntity
				if err := jsonx.Unmarshal(bytes, &entity); err == nil {
					found[entity.ID] = &entity
					continue
				}
			}
			missingIDs = append(missingIDs, seriesIDs[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIDs = seriesIDs
		missingKeys = keys
	}

	if len(missingIDs) > 0 {
		scope, err := jsonx.MarshalString(libraryIDs)
		if err != nil {
			return nil, err
		}
		wanted, err := jsonx.MarshalString(missingIDs)
		if err != nil {
			return nil, err
		}
		rows, err := r.q.GetKomgaSeriesByIDs(ctx, sqlc.GetKomgaSeriesByIDsParams{SeriesIds: wanted, LibraryIds: scope})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			found[row.ID] = (&models.KomgaSeriesEntity{}).FromSqlc(row)
		}

		if r.c != nil && !r.inTx {
			toCache := make(map[string]any, len(missingIDs))
			for i, id := range missingIDs {
				if entity, ok := found[id]; ok {
					toCache[missingKeys[i]] = entity
				}
			}
			if len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
		}
	}

	ordered := make([]*models.KomgaSeriesEntity, 0, len(seriesIDs))
	for _, id := range seriesIDs {
		if entity, ok := found[id]; ok {
			ordered = append(ordered, entity)
		}
	}
	return ordered, nil
}

func (r *komgaRepository) CountSeries(ctx context.Context, libraryIDs []string, search string) (int64, error) {
	scope, err := jsonx.MarshalString(libraryIDs)
	if err != nil {
		return 0, err
	}
	key := cache.BuildKey("komga", "series_count", strings.Join(libraryIDs, ","), search)
	if r.c != nil && !r.inTx {
		var cached int64
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		var searchArg any
		if search != "" {
			searchArg = search
		}
		count, err := r.q.CountKomgaSeries(ctx, sqlc.CountKomgaSeriesParams{LibraryIds: scope, Search: searchArg})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, count, constants.ListCacheDuration)
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (r *komgaRepository) ListSeriesBooks(ctx context.Context, seriesID string, libraryIDs []string) ([]*models.KomgaSeriesBookEntity, error) {
	scope, err := jsonx.MarshalString(libraryIDs)
	if err != nil {
		return nil, err
	}
	key := cache.BuildKey("komga", "series_books", seriesID, strings.Join(libraryIDs, ","))
	if r.c != nil && !r.inTx {
		var cached []*models.KomgaSeriesBookEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListKomgaSeriesBooks(ctx, sqlc.ListKomgaSeriesBooksParams{SeriesID: seriesID, LibraryIds: scope})
		if err != nil {
			return nil, err
		}
		result := make([]*models.KomgaSeriesBookEntity, 0, len(rows))
		for _, row := range rows {
			result = append(result, (&models.KomgaSeriesBookEntity{}).FromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.ListCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.KomgaSeriesBookEntity), nil
}

func (r *komgaRepository) GetBookSeries(ctx context.Context, bookID string) (*models.KomgaBookSeriesRefEntity, error) {
	key := cache.BuildKey("komga", "book_series", bookID)
	if r.c != nil && !r.inTx {
		var cached models.KomgaBookSeriesRefEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetKomgaBookSeries(ctx, bookID)
		if err != nil {
			return nil, err
		}
		result := (&models.KomgaBookSeriesRefEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.KomgaBookSeriesRefEntity), nil
}

func (r *komgaRepository) SeriesProgress(ctx context.Context, userID, seriesID string, libraryIDs []string) (*models.KomgaSeriesProgressEntity, error) {
	scope, err := jsonx.MarshalString(libraryIDs)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetKomgaSeriesProgress(ctx, sqlc.GetKomgaSeriesProgressParams{
		UserID:     userID,
		SeriesID:   seriesID,
		LibraryIds: scope,
	})
	if err != nil {
		return nil, err
	}
	return (&models.KomgaSeriesProgressEntity{}).FromSqlc(row), nil
}

func (r *komgaRepository) SeriesProgressByIDs(ctx context.Context, userID string, seriesIDs []string, libraryIDs []string) (map[string]*models.KomgaSeriesProgressEntity, error) {
	out := make(map[string]*models.KomgaSeriesProgressEntity, len(seriesIDs))
	if len(seriesIDs) == 0 {
		return out, nil
	}
	scope, err := jsonx.MarshalString(libraryIDs)
	if err != nil {
		return nil, err
	}
	wanted, err := jsonx.MarshalString(seriesIDs)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListKomgaSeriesProgress(ctx, sqlc.ListKomgaSeriesProgressParams{
		UserID:     userID,
		SeriesIds:  wanted,
		LibraryIds: scope,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.SeriesID] = (&models.KomgaSeriesProgressEntity{}).FromSqlcList(row)
	}
	return out, nil
}
