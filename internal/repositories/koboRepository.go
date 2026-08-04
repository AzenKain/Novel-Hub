package repositories

import (
	"context"
	"database/sql"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
	"novelhub/pkg/constants"
)

type KoboRepository interface {
	UpsertAuthToken(ctx context.Context, token, userID string) (*models.KoboAuthTokenEntity, error)
	GetAuthTokenByUser(ctx context.Context, userID string) (*models.KoboAuthTokenEntity, error)
	ResolveToken(ctx context.Context, token string) (*models.KoboAuthTokenEntity, error)
	TouchToken(ctx context.Context, token string) error
	DeleteAuthToken(ctx context.Context, userID string) error

	MarkBookSynced(ctx context.Context, userID, bookID string) error
	SyncedBookIDs(ctx context.Context, userID string) ([]string, error)
	CountSyncedBooks(ctx context.Context, userID string) (int64, error)
	ResetSyncedBooks(ctx context.Context, userID string) error

	WithTx(tx *sql.Tx) KoboRepository
}

type koboRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewKoboRepository(db sqlc.DBTX, c cache.Cache) KoboRepository {
	return &koboRepository{q: sqlc.New(db), c: c, sfg: &singleflight.Group{}}
}

func (r *koboRepository) WithTx(tx *sql.Tx) KoboRepository {
	if tx == nil {
		return r
	}
	return &koboRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sfg: r.sfg}
}

func (r *koboRepository) UpsertAuthToken(ctx context.Context, token, userID string) (*models.KoboAuthTokenEntity, error) {
	row, err := r.q.UpsertKoboAuthToken(ctx, sqlc.UpsertKoboAuthTokenParams{Token: token, UserID: userID})
	if err != nil {
		return nil, err
	}
	r.invalidate(ctx, userID)
	return (&models.KoboAuthTokenEntity{}).FromSqlc(row), nil
}

func (r *koboRepository) GetAuthTokenByUser(ctx context.Context, userID string) (*models.KoboAuthTokenEntity, error) {
	row, err := r.q.GetKoboAuthToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	return (&models.KoboAuthTokenEntity{}).FromSqlc(row), nil
}

// ResolveToken runs on every single device request, so it is cached by token. The token is
// the credential, so the cache key is namespaced and never logged.
func (r *koboRepository) ResolveToken(ctx context.Context, token string) (*models.KoboAuthTokenEntity, error) {
	key := cache.BuildKey("kobo", "token", token)
	if r.c != nil && !r.inTx {
		var entity models.KoboAuthTokenEntity
		if err := r.c.Get(ctx, key, &entity); err == nil {
			return &entity, nil
		}
	}
	value, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetKoboUserByToken(ctx, token)
		if err != nil {
			return nil, err
		}
		entity := (&models.KoboAuthTokenEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.KoboAuthTokenEntity), nil
}

func (r *koboRepository) TouchToken(ctx context.Context, token string) error {
	return r.q.TouchKoboAuthToken(ctx, token)
}

func (r *koboRepository) DeleteAuthToken(ctx context.Context, userID string) error {
	if err := r.q.DeleteKoboAuthToken(ctx, userID); err != nil {
		return err
	}
	r.invalidate(ctx, userID)
	return nil
}

func (r *koboRepository) MarkBookSynced(ctx context.Context, userID, bookID string) error {
	return r.q.MarkKoboBookSynced(ctx, sqlc.MarkKoboBookSyncedParams{UserID: userID, BookID: bookID})
}

func (r *koboRepository) SyncedBookIDs(ctx context.Context, userID string) ([]string, error) {
	return r.q.ListKoboSyncedBookIDs(ctx, userID)
}

func (r *koboRepository) CountSyncedBooks(ctx context.Context, userID string) (int64, error) {
	return r.q.CountKoboSyncedBooks(ctx, userID)
}

func (r *koboRepository) ResetSyncedBooks(ctx context.Context, userID string) error {
	return r.q.DeleteKoboSyncedBooks(ctx, userID)
}

// invalidate drops the token cache. The old token string is not known here, so the whole
// kobo token namespace is cleared — it holds one entry per paired device, so this is cheap.
func (r *koboRepository) invalidate(ctx context.Context, userID string) {
	if r.c == nil {
		return
	}
	_ = r.c.DelByPattern(context.Background(), constants.CacheKeyKoboTokenPattern)
	_ = r.c.Del(ctx, cache.BuildKey("kobo", "user", userID))
}
