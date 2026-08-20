package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type CustomizationRepository interface {
	GetSoundscapeByID(ctx context.Context, id string) (*models.SoundscapeEntity, error)
	GetSoundscapesByIDs(ctx context.Context, ids []string) ([]*models.SoundscapeEntity, error)
	ListSoundscapes(ctx context.Context, userID *string, cursor *time.Time, cursorID string, limit int64) ([]*models.SoundscapeEntity, error)
	CreateSoundscape(ctx context.Context, arg sqlc.CreateSoundscapeParams) (*models.SoundscapeEntity, error)
	UpdateSoundscape(ctx context.Context, arg sqlc.UpdateSoundscapeParams) (*models.SoundscapeEntity, error)
	DeleteSoundscape(ctx context.Context, id string) error

	GetCustomFontByID(ctx context.Context, id string) (*models.CustomFontEntity, error)
	GetCustomFontsByIDs(ctx context.Context, ids []string) ([]*models.CustomFontEntity, error)
	ListCustomFonts(ctx context.Context, userID *string, cursor *time.Time, cursorID string, limit int64) ([]*models.CustomFontEntity, error)
	CreateCustomFont(ctx context.Context, arg sqlc.CreateCustomFontParams) (*models.CustomFontEntity, error)
	DeleteCustomFont(ctx context.Context, id string) error

	GetCustomThemeByID(ctx context.Context, id string) (*models.CustomThemeEntity, error)
	GetCustomThemesByIDs(ctx context.Context, ids []string) ([]*models.CustomThemeEntity, error)
	ListCustomThemes(ctx context.Context, userID *string, cursor *time.Time, cursorID string, limit int64) ([]*models.CustomThemeEntity, error)
	CreateCustomTheme(ctx context.Context, arg sqlc.CreateCustomThemeParams) (*models.CustomThemeEntity, error)
	UpdateCustomTheme(ctx context.Context, arg sqlc.UpdateCustomThemeParams) (*models.CustomThemeEntity, error)
	DeleteCustomTheme(ctx context.Context, id string) error

	WithTx(tx *sql.Tx) CustomizationRepository
}

type customizationRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewCustomizationRepository(db sqlc.DBTX, c cache.Cache) CustomizationRepository {
	return &customizationRepository{
		q:  sqlc.New(db),
		c:  c,
		sf: &singleflight.Group{},
	}
}

func (r *customizationRepository) WithTx(tx *sql.Tx) CustomizationRepository {
	if tx == nil {
		return r
	}
	return &customizationRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sf:   r.sf,
	}
}

func (r *customizationRepository) GetSoundscapeByID(ctx context.Context, id string) (*models.SoundscapeEntity, error) {
	key := cache.BuildKey("soundscape", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.SoundscapeEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetSoundscapeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		entity := (&models.SoundscapeEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.SoundscapeEntity), nil
}

func (r *customizationRepository) GetSoundscapesByIDs(ctx context.Context, ids []string) ([]*models.SoundscapeEntity, error) {
	if len(ids) == 0 {
		return []*models.SoundscapeEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("soundscape", "id", id)
	}

	entities := make([]*models.SoundscapeEntity, len(ids))
	missingIDs := make([]string, 0)
	missingIndices := make([]int, 0)

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, b := range cachedBytes {
			if len(b) > 0 {
				var e models.SoundscapeEntity
				if err := jsonx.Unmarshal(b, &e); err == nil {
					entities[i] = &e
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
			missingIndices = append(missingIndices, i)
		}
	} else {
		missingIDs = ids
		for i := range ids {
			missingIndices = append(missingIndices, i)
		}
	}

	if len(missingIDs) > 0 {
		sfKey := "soundscape:mget:" + fmt.Sprint(missingIDs)
		v, err, _ := r.sf.Do(sfKey, func() (any, error) {
			rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.Soundscape, error) {
				return r.q.GetSoundscapesByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			fetched := (&models.SoundscapeEntities{}).FromSqlc(rows)
			if r.c != nil && !r.inTx {
				pairs := make(map[string]any, len(fetched))
				for _, item := range fetched {
					pairs[cache.BuildKey("soundscape", "id", item.ID)] = item
				}
				_ = r.c.MSet(ctx, pairs, constants.NormalCacheDuration)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}

		fetchedList := v.([]*models.SoundscapeEntity)
		fetchedMap := make(map[string]*models.SoundscapeEntity, len(fetchedList))
		for _, item := range fetchedList {
			fetchedMap[item.ID] = item
		}
		for idx, missingID := range missingIDs {
			origIdx := missingIndices[idx]
			if item, found := fetchedMap[missingID]; found {
				entities[origIdx] = item
			}
		}
	}

	result := make([]*models.SoundscapeEntity, 0, len(entities))
	for _, e := range entities {
		if e != nil {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *customizationRepository) ListSoundscapes(ctx context.Context, userID *string, cursor *time.Time, cursorID string, limit int64) ([]*models.SoundscapeEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	userScope := "public"
	if userID != nil && *userID != "" {
		userScope = *userID
	}
	cursorStr := ""
	if cursor != nil {
		cursorStr = cursor.Format(time.RFC3339Nano)
	}

	listKey := cache.BuildKey("soundscape", "list", userScope, fmt.Sprintf("%s_%s_%d", cursorStr, cursorID, limit))
	var ids []string

	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err != nil {
			ids = nil
		}
	}

	if ids == nil {
		v, err, _ := r.sf.Do(listKey, func() (any, error) {
			var fetchedIDs []string
			var err error
			if userID != nil && *userID != "" {
				fetchedIDs, err = r.q.ListSoundscapeIDsAccessible(ctx, sqlc.ListSoundscapeIDsAccessibleParams{
					UserID:          sql.NullString{String: *userID, Valid: true},
					CursorUpdatedAt: cursorTimeArg(cursor),
					CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
					Limit:           limit,
				})
			} else {
				fetchedIDs, err = r.q.ListSystemSoundscapeIDs(ctx, sqlc.ListSystemSoundscapeIDsParams{
					CursorUpdatedAt: cursorTimeArg(cursor),
					CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
					Limit:           limit,
				})
			}
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, listKey, fetchedIDs, constants.ListCacheDuration)
			}
			return fetchedIDs, nil
		})
		if err != nil {
			return nil, err
		}
		ids = v.([]string)
	}

	return r.GetSoundscapesByIDs(ctx, ids)
}

func (r *customizationRepository) CreateSoundscape(ctx context.Context, arg sqlc.CreateSoundscapeParams) (*models.SoundscapeEntity, error) {
	row, err := r.q.CreateSoundscape(ctx, arg)
	if err != nil {
		return nil, err
	}
	entity := (&models.SoundscapeEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("soundscape", "id", entity.ID))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("soundscape", "list", "*"))
	}
	return entity, nil
}

func (r *customizationRepository) UpdateSoundscape(ctx context.Context, arg sqlc.UpdateSoundscapeParams) (*models.SoundscapeEntity, error) {
	row, err := r.q.UpdateSoundscape(ctx, arg)
	if err != nil {
		return nil, err
	}
	entity := (&models.SoundscapeEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("soundscape", "id", entity.ID))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("soundscape", "list", "*"))
	}
	return entity, nil
}

func (r *customizationRepository) DeleteSoundscape(ctx context.Context, id string) error {
	if err := r.q.DeleteSoundscape(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("soundscape", "id", id))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("soundscape", "list", "*"))
	}
	return nil
}


func (r *customizationRepository) GetCustomFontByID(ctx context.Context, id string) (*models.CustomFontEntity, error) {
	key := cache.BuildKey("custom_font", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.CustomFontEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetCustomFontByID(ctx, id)
		if err != nil {
			return nil, err
		}
		entity := (&models.CustomFontEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.CustomFontEntity), nil
}

func (r *customizationRepository) GetCustomFontsByIDs(ctx context.Context, ids []string) ([]*models.CustomFontEntity, error) {
	if len(ids) == 0 {
		return []*models.CustomFontEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("custom_font", "id", id)
	}

	entities := make([]*models.CustomFontEntity, len(ids))
	missingIDs := make([]string, 0)
	missingIndices := make([]int, 0)

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, b := range cachedBytes {
			if len(b) > 0 {
				var e models.CustomFontEntity
				if err := jsonx.Unmarshal(b, &e); err == nil {
					entities[i] = &e
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
			missingIndices = append(missingIndices, i)
		}
	} else {
		missingIDs = ids
		for i := range ids {
			missingIndices = append(missingIndices, i)
		}
	}

	if len(missingIDs) > 0 {
		sfKey := "custom_font:mget:" + fmt.Sprint(missingIDs)
		v, err, _ := r.sf.Do(sfKey, func() (any, error) {
			rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.CustomFont, error) {
				return r.q.GetCustomFontsByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			fetched := (&models.CustomFontEntities{}).FromSqlc(rows)
			if r.c != nil && !r.inTx {
				pairs := make(map[string]any, len(fetched))
				for _, item := range fetched {
					pairs[cache.BuildKey("custom_font", "id", item.ID)] = item
				}
				_ = r.c.MSet(ctx, pairs, constants.NormalCacheDuration)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}

		fetchedList := v.([]*models.CustomFontEntity)
		fetchedMap := make(map[string]*models.CustomFontEntity, len(fetchedList))
		for _, item := range fetchedList {
			fetchedMap[item.ID] = item
		}
		for idx, missingID := range missingIDs {
			origIdx := missingIndices[idx]
			if item, found := fetchedMap[missingID]; found {
				entities[origIdx] = item
			}
		}
	}

	result := make([]*models.CustomFontEntity, 0, len(entities))
	for _, e := range entities {
		if e != nil {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *customizationRepository) ListCustomFonts(ctx context.Context, userID *string, cursor *time.Time, cursorID string, limit int64) ([]*models.CustomFontEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	userScope := "public"
	if userID != nil && *userID != "" {
		userScope = *userID
	}
	cursorStr := ""
	if cursor != nil {
		cursorStr = cursor.Format(time.RFC3339Nano)
	}

	listKey := cache.BuildKey("custom_font", "list", userScope, fmt.Sprintf("%s_%s_%d", cursorStr, cursorID, limit))
	var ids []string

	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err != nil {
			ids = nil
		}
	}

	if ids == nil {
		v, err, _ := r.sf.Do(listKey, func() (any, error) {
			var fetchedIDs []string
			var err error
			if userID != nil && *userID != "" {
				fetchedIDs, err = r.q.ListCustomFontIDsAccessible(ctx, sqlc.ListCustomFontIDsAccessibleParams{
					UserID:          sql.NullString{String: *userID, Valid: true},
					CursorUpdatedAt: cursorTimeArg(cursor),
					CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
					Limit:           limit,
				})
			} else {
				fetchedIDs, err = r.q.ListSystemCustomFontIDs(ctx, sqlc.ListSystemCustomFontIDsParams{
					CursorUpdatedAt: cursorTimeArg(cursor),
					CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
					Limit:           limit,
				})
			}
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, listKey, fetchedIDs, constants.ListCacheDuration)
			}
			return fetchedIDs, nil
		})
		if err != nil {
			return nil, err
		}
		ids = v.([]string)
	}

	return r.GetCustomFontsByIDs(ctx, ids)
}

func (r *customizationRepository) CreateCustomFont(ctx context.Context, arg sqlc.CreateCustomFontParams) (*models.CustomFontEntity, error) {
	row, err := r.q.CreateCustomFont(ctx, arg)
	if err != nil {
		return nil, err
	}
	entity := (&models.CustomFontEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("custom_font", "id", entity.ID))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("custom_font", "list", "*"))
	}
	return entity, nil
}

func (r *customizationRepository) DeleteCustomFont(ctx context.Context, id string) error {
	if err := r.q.DeleteCustomFont(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("custom_font", "id", id))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("custom_font", "list", "*"))
	}
	return nil
}

func (r *customizationRepository) GetCustomThemeByID(ctx context.Context, id string) (*models.CustomThemeEntity, error) {
	key := cache.BuildKey("custom_theme", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.CustomThemeEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetCustomThemeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		entity := (&models.CustomThemeEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.CustomThemeEntity), nil
}

func (r *customizationRepository) GetCustomThemesByIDs(ctx context.Context, ids []string) ([]*models.CustomThemeEntity, error) {
	if len(ids) == 0 {
		return []*models.CustomThemeEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("custom_theme", "id", id)
	}

	entities := make([]*models.CustomThemeEntity, len(ids))
	missingIDs := make([]string, 0)
	missingIndices := make([]int, 0)

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, b := range cachedBytes {
			if len(b) > 0 {
				var e models.CustomThemeEntity
				if err := jsonx.Unmarshal(b, &e); err == nil {
					entities[i] = &e
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
			missingIndices = append(missingIndices, i)
		}
	} else {
		missingIDs = ids
		for i := range ids {
			missingIndices = append(missingIndices, i)
		}
	}

	if len(missingIDs) > 0 {
		sfKey := "custom_theme:mget:" + fmt.Sprint(missingIDs)
		v, err, _ := r.sf.Do(sfKey, func() (any, error) {
			rows, err := queryInChunks(missingIDs, func(chunk []string) ([]sqlc.CustomTheme, error) {
				return r.q.GetCustomThemesByIDs(ctx, chunk)
			})
			if err != nil {
				return nil, err
			}
			fetched := (&models.CustomThemeEntities{}).FromSqlc(rows)
			if r.c != nil && !r.inTx {
				pairs := make(map[string]any, len(fetched))
				for _, item := range fetched {
					pairs[cache.BuildKey("custom_theme", "id", item.ID)] = item
				}
				_ = r.c.MSet(ctx, pairs, constants.NormalCacheDuration)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}

		fetchedList := v.([]*models.CustomThemeEntity)
		fetchedMap := make(map[string]*models.CustomThemeEntity, len(fetchedList))
		for _, item := range fetchedList {
			fetchedMap[item.ID] = item
		}
		for idx, missingID := range missingIDs {
			origIdx := missingIndices[idx]
			if item, found := fetchedMap[missingID]; found {
				entities[origIdx] = item
			}
		}
	}

	result := make([]*models.CustomThemeEntity, 0, len(entities))
	for _, e := range entities {
		if e != nil {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *customizationRepository) ListCustomThemes(ctx context.Context, userID *string, cursor *time.Time, cursorID string, limit int64) ([]*models.CustomThemeEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	userScope := "public"
	if userID != nil && *userID != "" {
		userScope = *userID
	}
	cursorStr := ""
	if cursor != nil {
		cursorStr = cursor.Format(time.RFC3339Nano)
	}

	listKey := cache.BuildKey("custom_theme", "list", userScope, fmt.Sprintf("%s_%s_%d", cursorStr, cursorID, limit))
	var ids []string

	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, listKey, &ids); err != nil {
			ids = nil
		}
	}

	if ids == nil {
		v, err, _ := r.sf.Do(listKey, func() (any, error) {
			var fetchedIDs []string
			var err error
			if userID != nil && *userID != "" {
				fetchedIDs, err = r.q.ListCustomThemeIDsAccessible(ctx, sqlc.ListCustomThemeIDsAccessibleParams{
					UserID:          sql.NullString{String: *userID, Valid: true},
					CursorUpdatedAt: cursorTimeArg(cursor),
					CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
					Limit:           limit,
				})
			} else {
				fetchedIDs, err = r.q.ListSystemCustomThemeIDs(ctx, sqlc.ListSystemCustomThemeIDsParams{
					CursorUpdatedAt: cursorTimeArg(cursor),
					CursorID:        convert.StrPtrToNullStringNonEmpty(&cursorID),
					Limit:           limit,
				})
			}
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, listKey, fetchedIDs, constants.ListCacheDuration)
			}
			return fetchedIDs, nil
		})
		if err != nil {
			return nil, err
		}
		ids = v.([]string)
	}

	return r.GetCustomThemesByIDs(ctx, ids)
}

func (r *customizationRepository) CreateCustomTheme(ctx context.Context, arg sqlc.CreateCustomThemeParams) (*models.CustomThemeEntity, error) {
	row, err := r.q.CreateCustomTheme(ctx, arg)
	if err != nil {
		return nil, err
	}
	entity := (&models.CustomThemeEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("custom_theme", "id", entity.ID))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("custom_theme", "list", "*"))
	}
	return entity, nil
}

func (r *customizationRepository) UpdateCustomTheme(ctx context.Context, arg sqlc.UpdateCustomThemeParams) (*models.CustomThemeEntity, error) {
	row, err := r.q.UpdateCustomTheme(ctx, arg)
	if err != nil {
		return nil, err
	}
	entity := (&models.CustomThemeEntity{}).FromSqlc(row)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("custom_theme", "id", entity.ID))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("custom_theme", "list", "*"))
	}
	return entity, nil
}

func (r *customizationRepository) DeleteCustomTheme(ctx context.Context, id string) error {
	if err := r.q.DeleteCustomTheme(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("custom_theme", "id", id))
		_ = r.c.DelByPattern(ctx, cache.BuildKey("custom_theme", "list", "*"))
	}
	return nil
}
