package dbtx

import (
	"context"
	"database/sql"
)

// DB adalah query surface yang dipenuhi *sql.DB dan *sql.Tx.
// Repo pakai ini supaya call yang sama jalan di koneksi biasa
// maupun di dalam transaksi context.
type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type ctxKey struct{}

// WithTx menjalankan fn dalam satu transaksi dari db yang sama.
// Kalau ctx sudah membawa tx (nested service call), fn join tx itu
// tanpa begin/commit baru. Gagal → rollback, sukses → commit.
func WithTx(db *sql.DB, ctx context.Context, fn func(context.Context) error) error {
	if _, ok := ctx.Value(ctxKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(context.WithValue(ctx, ctxKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// From mengembalikan tx yang di-inject WithTx; ok=false bila di luar TX.
func From(ctx context.Context) (DB, bool) {
	tx, ok := ctx.Value(ctxKey{}).(*sql.Tx)
	if !ok || tx == nil {
		return nil, false
	}
	return tx, true
}
