package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type LibraryRepository interface {
	CreateLibrary(ctx context.Context, library *models.LibraryEntity) error
	GetLibrary(ctx context.Context, id string) (*models.LibraryEntity, error)
	ListLibraries(ctx context.Context) ([]*models.LibraryEntity, error)
	GetLibrariesByIDs(ctx context.Context, ids []string) ([]*models.LibraryEntity, error)
	UpdateLibrary(ctx context.Context, library *models.LibraryEntity) error
	DeleteLibrary(ctx context.Context, id string) error
	WithTx(tx *sql.Tx) LibraryRepository
}

type libraryRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
	inTx    bool
	sfg     *singleflight.Group
}

func NewLibraryRepository(db *sql.DB, c cache.Cache) LibraryRepository {
	return &libraryRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
		sfg:     &singleflight.Group{},
	}
}

func (r *libraryRepository) WithTx(tx *sql.Tx) LibraryRepository {
	if tx == nil {
		return r
	}
	return &libraryRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
		c:       r.c,
		inTx:    true,
		sfg:     r.sfg,
	}
}

func (r *libraryRepository) CreateLibrary(ctx context.Context, library *models.LibraryEntity) error {
	params := sqlc.CreateLibraryParams{
		ID:   library.ID,
		Name: library.Name,
	}
	res, err := r.queries.CreateLibrary(ctx, params)
	if err != nil {
		return err
	}
	library.CreatedAt = res.CreatedAt.Time
	library.UpdatedAt = res.UpdatedAt.Time
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyLibraryList)
	}
	return nil
}

func (r *libraryRepository) GetLibrary(ctx context.Context, id string) (*models.LibraryEntity, error) {
	key := cache.BuildKey("library", "id", id)
	if r.c != nil && !r.inTx {
		var library models.LibraryEntity
		if err := r.c.Get(ctx, key, &library); err == nil {
			return &library, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		res, err := r.queries.GetLibrary(ctx, id)
		if err != nil {
			return nil, err
		}
		library := (&models.LibraryEntity{}).FromSqlc(res)

		if r.c != nil {
			_ = r.c.Set(ctx, key, library, constants.NormalCacheDuration)
		}
		return library, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.LibraryEntity), nil
}

func (r *libraryRepository) ListLibraries(ctx context.Context) ([]*models.LibraryEntity, error) {
	key := constants.CacheKeyLibraryList
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetLibrariesByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.queries.ListLibraryIDs(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetLibrariesByIDs(ctx, v.([]string))
}

func (r *libraryRepository) GetLibrariesByIDs(ctx context.Context, ids []string) ([]*models.LibraryEntity, error) {
	if len(ids) == 0 {
		return []*models.LibraryEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("library", "id", id)
	}

	libraries := make([]*models.LibraryEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var lib models.LibraryEntity
				if err := jsonx.Unmarshal(bytes, &lib); err == nil {
					libraries = append(libraries, &lib)
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
		sort.Strings(missingIds)
		sfgKey := "libraries:ids:" + strings.Join(missingIds, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := queryInChunks(missingIds, func(chunk []string) ([]sqlc.Library, error) {
				return r.queries.GetLibrariesByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			missingMap := make(map[string]*models.LibraryEntity)
			for _, row := range rows {
				lib := (&models.LibraryEntity{}).FromSqlc(row)
				missingMap[lib.ID] = lib
			}
			return missingMap, nil
		})
		if err != nil {
			return nil, err
		}
		missingMap := v.(map[string]*models.LibraryEntity)

		for _, lib := range missingMap {
			libraries = append(libraries, lib)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for _, missingId := range missingIds {
				if l, ok := missingMap[missingId]; ok {
					missingToCache[cache.BuildKey("library", "id", missingId)] = l
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	libMap := make(map[string]*models.LibraryEntity)
	for _, l := range libraries {
		libMap[l.ID] = l
	}
	ordered := make([]*models.LibraryEntity, 0, len(ids))
	for _, id := range ids {
		if l, ok := libMap[id]; ok {
			ordered = append(ordered, l)
		}
	}

	return ordered, nil
}

func (r *libraryRepository) UpdateLibrary(ctx context.Context, library *models.LibraryEntity) error {
	params := sqlc.UpdateLibraryParams{
		ID:   library.ID,
		Name: library.Name,
	}
	res, err := r.queries.UpdateLibrary(ctx, params)
	if err != nil {
		return err
	}
	library.UpdatedAt = res.UpdatedAt.Time

	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("library", "id", library.ID))
		_ = r.c.Del(ctx, constants.CacheKeyLibraryList)
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
	}
	return nil
}

func (r *libraryRepository) DeleteLibrary(ctx context.Context, id string) error {
	err := r.queries.DeleteLibrary(ctx, id)
	if err == nil && r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("library", "id", id))
		_ = r.c.Del(ctx, constants.CacheKeyLibraryList)
		_ = r.c.Del(ctx, constants.CacheKeyLibraryStats)
		// books.library_id is ON DELETE CASCADE, so this also removed every book in the
		// library plus their chapters, files and tag links. We don't know which ids those
		// were, so sweep by pattern the way BulkDeleteBooks does for the same DB effect.
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookAllPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookIDsPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyChapterPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookFilePattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyMetadataCountPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyFTSPattern)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyBookTrackerMapPattern)
	}
	return err
}
