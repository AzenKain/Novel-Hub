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

type TOTPRepository interface {
	Get(ctx context.Context, userID string) (*models.TOTPEntity, error)
	Upsert(ctx context.Context, userID string, secret string) (*models.TOTPEntity, error)
	Confirm(ctx context.Context, userID string) (bool, error)
	Delete(ctx context.Context, userID string) error
	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error
	ConsumeRecoveryCode(ctx context.Context, userID string, hash string) (bool, error)
	CountUnusedRecoveryCodes(ctx context.Context, userID string) (int64, error)
	WithTx(tx *sql.Tx) TOTPRepository
}

type totpRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewTOTPRepository(db sqlc.DBTX, c cache.Cache) TOTPRepository {
	return &totpRepository{q: sqlc.New(db), c: c, sf: &singleflight.Group{}}
}

func (r *totpRepository) WithTx(tx *sql.Tx) TOTPRepository {
	if tx == nil {
		return r
	}
	return &totpRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sf: r.sf}
}

func (r *totpRepository) Get(ctx context.Context, userID string) (*models.TOTPEntity, error) {
	key := cache.BuildKey("totp", "user", userID)
	if r.c != nil && !r.inTx {
		var cached models.TOTPEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetUserTOTP(ctx, userID)
		if err != nil {
			return nil, err
		}
		entity := models.TOTPFromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.TOTPEntity), nil
}

func (r *totpRepository) Upsert(ctx context.Context, userID string, secret string) (*models.TOTPEntity, error) {
	row, err := r.q.UpsertUserTOTP(ctx, sqlc.UpsertUserTOTPParams{UserID: userID, Secret: secret})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("totp", "user", userID))
	}
	return models.TOTPFromSqlc(row), nil
}

func (r *totpRepository) Confirm(ctx context.Context, userID string) (bool, error) {
	affected, err := r.q.ConfirmUserTOTP(ctx, userID)
	if err != nil {
		return false, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("totp", "user", userID))
	}
	return affected > 0, nil
}

func (r *totpRepository) Delete(ctx context.Context, userID string) error {
	if err := r.q.DeleteUserTOTP(ctx, userID); err != nil {
		return err
	}
	if err := r.q.DeleteRecoveryCodes(ctx, userID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("totp", "user", userID))
	}
	return nil
}

func (r *totpRepository) ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	if err := r.q.DeleteRecoveryCodes(ctx, userID); err != nil {
		return err
	}
	for _, hash := range hashes {
		if err := r.q.CreateRecoveryCode(ctx, sqlc.CreateRecoveryCodeParams{UserID: userID, CodeHash: hash}); err != nil {
			return err
		}
	}
	return nil
}

func (r *totpRepository) ConsumeRecoveryCode(ctx context.Context, userID string, hash string) (bool, error) {
	affected, err := r.q.ConsumeRecoveryCode(ctx, sqlc.ConsumeRecoveryCodeParams{UserID: userID, CodeHash: hash})
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *totpRepository) CountUnusedRecoveryCodes(ctx context.Context, userID string) (int64, error) {
	return r.q.CountUnusedRecoveryCodes(ctx, userID)
}
