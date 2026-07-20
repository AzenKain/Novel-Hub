package repositories

import (
	"context"
	"database/sql"

	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
)

type SettingsRepository interface {
	List(ctx context.Context) ([]*models.AppSettingEntity, error)
	Get(ctx context.Context, key string) (*models.AppSettingEntity, error)
	Upsert(ctx context.Context, key string, valueJSON string) error
	GetSetupState(ctx context.Context, key string) (string, error)
	UpsertSetupState(ctx context.Context, key string, value string) error
	CountAdminUsers(ctx context.Context) (int64, error)
	WithTx(tx *sql.Tx) SettingsRepository
}

type settingsRepository struct {
	q *sqlc.Queries
}

func NewSettingsRepository(db sqlc.DBTX) SettingsRepository {
	return &settingsRepository{q: sqlc.New(db)}
}

func (r *settingsRepository) WithTx(tx *sql.Tx) SettingsRepository {
	return &settingsRepository{q: r.q.WithTx(tx)}
}

func mapAppSetting(row sqlc.AppSetting) *models.AppSettingEntity {
	return &models.AppSettingEntity{
		Key:       row.Key,
		ValueJSON: row.ValueJson,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *settingsRepository) List(ctx context.Context) ([]*models.AppSettingEntity, error) {
	rows, err := r.q.ListAppSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*models.AppSettingEntity, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAppSetting(row))
	}
	return out, nil
}

func (r *settingsRepository) Get(ctx context.Context, key string) (*models.AppSettingEntity, error) {
	row, err := r.q.GetAppSetting(ctx, key)
	if err != nil {
		return nil, err
	}
	return mapAppSetting(row), nil
}

func (r *settingsRepository) Upsert(ctx context.Context, key string, valueJSON string) error {
	return r.q.UpsertAppSetting(ctx, sqlc.UpsertAppSettingParams{Key: key, ValueJson: valueJSON})
}

func (r *settingsRepository) GetSetupState(ctx context.Context, key string) (string, error) {
	row, err := r.q.GetSetupState(ctx, key)
	if err != nil {
		return "", err
	}
	return row.Value, nil
}

func (r *settingsRepository) UpsertSetupState(ctx context.Context, key string, value string) error {
	return r.q.UpsertSetupState(ctx, sqlc.UpsertSetupStateParams{Key: key, Value: value})
}

func (r *settingsRepository) CountAdminUsers(ctx context.Context) (int64, error) {
	return r.q.CountAdminUsers(ctx)
}
