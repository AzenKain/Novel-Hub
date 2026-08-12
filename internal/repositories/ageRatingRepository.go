package repositories

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/convert"
	"novelhub/pkg/jsonx"
)

type AgeRatingRepository interface {
	GetContentWarnings(ctx context.Context) ([]*models.ContentWarningEntity, error)
	GetBookContentWarnings(ctx context.Context, bookID string) ([]*models.ContentWarningEntity, error)
	UpdateBookAgeRatingAndWarnings(ctx context.Context, bookID string, ageRating string, warningIDs []string) error
	GetUserKidsModeInfo(ctx context.Context, userID string) (*models.KidsModeInfoEntity, error)
	SetKidsModePin(ctx context.Context, userID string, pinHash string) error
	SetKidsModeStatus(ctx context.Context, userID string, enable bool) error
	WithTx(tx *sql.Tx) AgeRatingRepository
}

type ageRatingRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewAgeRatingRepository(db sqlc.DBTX, c cache.Cache) AgeRatingRepository {
	return &ageRatingRepository{q: sqlc.New(db), c: c, sf: &singleflight.Group{}}
}

func (r *ageRatingRepository) WithTx(tx *sql.Tx) AgeRatingRepository {
	if tx == nil {
		return r
	}
	return &ageRatingRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sf: r.sf}
}

func (r *ageRatingRepository) GetContentWarnings(ctx context.Context) ([]*models.ContentWarningEntity, error) {
	// 1. Query IDs from DB
	ids, err := r.q.ListContentWarningIDs(ctx)
	if err != nil || len(ids) == 0 {
		return []*models.ContentWarningEntity{}, err
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("content_warning", id)
	}

	resultMap := make(map[string]*models.ContentWarningEntity, len(ids))
	missingIDs := make([]string, 0, len(ids))

	// 2. Perform MGet from RAM Cache (Cache-by-IDs pattern)
	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var entity models.ContentWarningEntity
				if err := jsonx.Unmarshal(bytes, &entity); err == nil {
					resultMap[ids[i]] = &entity
					continue
				}
			}
			missingIDs = append(missingIDs, ids[i])
		}
	} else {
		missingIDs = ids
	}

	// 3. Singleflight DB query fallback for missing IDs
	if len(missingIDs) > 0 {
		sfgKey := "content_warnings:ids:" + strings.Join(missingIDs, ",")
		v, err, _ := r.sf.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetContentWarningsByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetchedMap := make(map[string]*models.ContentWarningEntity, len(rows))
			cachePairs := make(map[string]any, len(rows))
			for _, row := range rows {
				entity := (&models.ContentWarningEntity{}).FromSqlc(row)
				fetchedMap[row.ID] = entity
				cachePairs[cache.BuildKey("content_warning", row.ID)] = entity
			}

			if r.c != nil && !r.inTx && len(cachePairs) > 0 {
				_ = r.c.MSet(ctx, cachePairs, 1*time.Hour)
			}
			return fetchedMap, nil
		})
		if err != nil {
			return nil, err
		}
		if fetched, ok := v.(map[string]*models.ContentWarningEntity); ok {
			for k, val := range fetched {
				resultMap[k] = val
			}
		}
	}

	// 4. Construct ordered list
	out := make([]*models.ContentWarningEntity, 0, len(ids))
	for _, id := range ids {
		if cw, ok := resultMap[id]; ok && cw != nil {
			out = append(out, cw)
		}
	}
	return out, nil
}

func (r *ageRatingRepository) GetBookContentWarnings(ctx context.Context, bookID string) ([]*models.ContentWarningEntity, error) {
	key := cache.BuildKey("content_warnings", "book", bookID)
	if r.c != nil && !r.inTx {
		var cached []*models.ContentWarningEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		rows, err := r.q.GetBookContentWarnings(ctx, bookID)
		if err != nil {
			return nil, err
		}
		result := make([]*models.ContentWarningEntity, 0, len(rows))
		for _, row := range rows {
			result = append(result, (&models.ContentWarningEntity{}).FromBookRow(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, 30*time.Minute)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.ContentWarningEntity), nil
}

func (r *ageRatingRepository) UpdateBookAgeRatingAndWarnings(ctx context.Context, bookID string, ageRating string, warningIDs []string) error {
	if err := r.q.UpdateBookAgeRating(ctx, sqlc.UpdateBookAgeRatingParams{
		AgeRating: ageRating,
		ID:        bookID,
	}); err != nil {
		return err
	}

	if err := r.q.ClearBookContentWarnings(ctx, bookID); err != nil {
		return err
	}

	for _, wID := range warningIDs {
		if err := r.q.AddBookContentWarning(ctx, sqlc.AddBookContentWarningParams{
			BookID:    bookID,
			WarningID: wID,
		}); err != nil {
			return err
		}
	}

	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("content_warnings", "book", bookID))
		_ = r.c.Del(ctx, cache.BuildKey("book", bookID))
	}
	return nil
}

func (r *ageRatingRepository) GetUserKidsModeInfo(ctx context.Context, userID string) (*models.KidsModeInfoEntity, error) {
	key := cache.BuildKey("user", "kids_mode", userID)
	if r.c != nil && !r.inTx {
		var cached models.KidsModeInfoEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetUserKidsModeInfo(ctx, userID)
		if err != nil {
			return nil, err
		}
		entity := (&models.KidsModeInfoEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, 15*time.Minute)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.KidsModeInfoEntity), nil
}

func (r *ageRatingRepository) SetKidsModePin(ctx context.Context, userID string, pinHash string) error {
	err := r.q.UpdateUserKidsModePin(ctx, sqlc.UpdateUserKidsModePinParams{
		KidsModePinHash: convert.StrPtrToNullString(&pinHash),
		ID:              userID,
	})
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "kids_mode", userID))
	}
	return nil
}

func (r *ageRatingRepository) SetKidsModeStatus(ctx context.Context, userID string, enable bool) error {
	err := r.q.UpdateUserKidsModeStatus(ctx, sqlc.UpdateUserKidsModeStatusParams{
		IsKidsMode: convert.BoolToInt64(enable),
		ID:         userID,
	})
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "kids_mode", userID))
	}
	return nil
}
