package repositories

import (
	"context"
	"database/sql"
	"fmt"

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
}

type libraryRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
	c       cache.Cache
}

func NewLibraryRepository(db *sql.DB, c cache.Cache) LibraryRepository {
	return &libraryRepository{
		db:      db,
		queries: sqlc.New(db),
		c:       c,
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
		_ = r.c.Del(ctx, "library:list")
	}
	return nil
}

func (r *libraryRepository) GetLibrary(ctx context.Context, id string) (*models.LibraryEntity, error) {
	key := fmt.Sprintf("library:id:%s", id)
	if r.c != nil {
		var library models.LibraryEntity
		if err := r.c.Get(ctx, key, &library); err == nil {
			return &library, nil
		}
	}

	res, err := r.queries.GetLibrary(ctx, id)
	if err != nil {
		return nil, err
	}
	library := (&models.LibraryEntity{}).FromSqlc(res)

	if r.c != nil {
		_ = r.c.Set(ctx, key, library, constants.NormalCacheDuration)
	}
	return library, nil
}

func (r *libraryRepository) ListLibraries(ctx context.Context) ([]*models.LibraryEntity, error) {
	key := "library:list"
	if r.c != nil {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetLibrariesByIDs(ctx, ids)
		}
	}

	ids, err := r.queries.ListLibraryIDs(ctx)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
	}
	return r.GetLibrariesByIDs(ctx, ids)
}

func (r *libraryRepository) GetLibrariesByIDs(ctx context.Context, ids []string) ([]*models.LibraryEntity, error) {
	if len(ids) == 0 {
		return []*models.LibraryEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("library:id:%s", id)
	}

	libraries := make([]*models.LibraryEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil {
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
		rows, err := r.queries.GetLibrariesByIDs(ctx, missingIds)
		if err != nil {
			return nil, err
		}

		missingMap := make(map[string]*models.LibraryEntity)
		for _, row := range rows {
			lib := (&models.LibraryEntity{}).FromSqlc(row)
			missingMap[lib.ID] = lib
			libraries = append(libraries, lib)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for i, missingId := range missingIds {
				if l, ok := missingMap[missingId]; ok {
					missingToCache[missingKeys[i]] = l
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
		_ = r.c.Del(ctx, fmt.Sprintf("library:id:%s", library.ID))
		_ = r.c.Del(ctx, "library:list")
		_ = r.c.Del(ctx, "feature:library_stats")
	}
	return nil
}

func (r *libraryRepository) DeleteLibrary(ctx context.Context, id string) error {
	err := r.queries.DeleteLibrary(ctx, id)
	if err == nil && r.c != nil {
		_ = r.c.Del(ctx, fmt.Sprintf("library:id:%s", id))
		_ = r.c.Del(ctx, "library:list")
		_ = r.c.Del(ctx, "feature:library_stats")
	}
	return err
}
