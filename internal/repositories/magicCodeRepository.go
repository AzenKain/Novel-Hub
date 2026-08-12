package repositories

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/pkg/cache"
)

type MagicCodeRepository interface {
	Create(ctx context.Context, id, code, pollToken, deviceInfo string, expiresAt time.Time) error
	GetByCode(ctx context.Context, code string) (*models.MagicCodeEntity, error)
	GetByPollToken(ctx context.Context, pollToken string) (*models.MagicCodeEntity, error)
	Activate(ctx context.Context, code, userID, jwtToken string) error
	MarkUsed(ctx context.Context, pollToken string) error
	DeleteExpired(ctx context.Context) error
	WithTx(tx *sql.Tx) MagicCodeRepository
}

type magicCodeRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sf   *singleflight.Group
}

func NewMagicCodeRepository(db sqlc.DBTX, c cache.Cache) MagicCodeRepository {
	return &magicCodeRepository{q: sqlc.New(db), c: c, sf: &singleflight.Group{}}
}

func (r *magicCodeRepository) WithTx(tx *sql.Tx) MagicCodeRepository {
	if tx == nil {
		return r
	}
	return &magicCodeRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sf: r.sf}
}

func (r *magicCodeRepository) Create(ctx context.Context, id, code, pollToken, deviceInfo string, expiresAt time.Time) error {
	return r.q.CreateMagicCode(ctx, sqlc.CreateMagicCodeParams{
		ID:         id,
		Code:       code,
		PollToken:  pollToken,
		DeviceInfo: deviceInfo,
		ExpiresAt:  expiresAt,
	})
}

func (r *magicCodeRepository) GetByCode(ctx context.Context, code string) (*models.MagicCodeEntity, error) {
	key := cache.BuildKey("magic_code", "code", code)
	if r.c != nil && !r.inTx {
		var cached models.MagicCodeEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetMagicCodeByCode(ctx, code)
		if err != nil {
			return nil, err
		}
		entity := (&models.MagicCodeEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, 5*time.Minute)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.MagicCodeEntity), nil
}

func (r *magicCodeRepository) GetByPollToken(ctx context.Context, pollToken string) (*models.MagicCodeEntity, error) {
	key := cache.BuildKey("magic_code", "poll", pollToken)
	if r.c != nil && !r.inTx {
		var cached models.MagicCodeEntity
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sf.Do(key, func() (any, error) {
		row, err := r.q.GetMagicCodeByPollToken(ctx, pollToken)
		if err != nil {
			return nil, err
		}
		entity := (&models.MagicCodeEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, 5*time.Minute)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.MagicCodeEntity), nil
}

func (r *magicCodeRepository) Activate(ctx context.Context, code, userID, jwtToken string) error {
	record, _ := r.GetByCode(ctx, code)
	err := r.q.ActivateMagicCode(ctx, sqlc.ActivateMagicCodeParams{
		UserID:   sql.NullString{String: userID, Valid: true},
		JwtToken: jwtToken,
		Code:     code,
	})
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("magic_code", "code", code))
		if record != nil && record.PollToken != "" {
			_ = r.c.Del(ctx, cache.BuildKey("magic_code", "poll", record.PollToken))
		}
	}
	return nil
}

func (r *magicCodeRepository) MarkUsed(ctx context.Context, pollToken string) error {
	record, _ := r.GetByPollToken(ctx, pollToken)
	err := r.q.MarkMagicCodeUsed(ctx, pollToken)
	if err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("magic_code", "poll", pollToken))
		if record != nil && record.Code != "" {
			_ = r.c.Del(ctx, cache.BuildKey("magic_code", "code", record.Code))
		}
	}
	return nil
}

func (r *magicCodeRepository) DeleteExpired(ctx context.Context) error {
	return r.q.DeleteExpiredMagicCodes(ctx)
}
