package database

import (
	"context"
	"database/sql"
)

type TxManager interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	DB() *sql.DB
}

type sqlTxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) TxManager {
	return &sqlTxManager{db: db}
}

func (m *sqlTxManager) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return m.db.BeginTx(ctx, opts)
}

func (m *sqlTxManager) DB() *sql.DB {
	return m.db
}
