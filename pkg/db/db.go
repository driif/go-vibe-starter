package db

import (
	"context"
	"database/sql"
	"log/slog"
)

// DBTX is the interface satisfied by both *sql.DB and *sql.Tx,
// matching the interface sqlc generates for query methods.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// TxFn is a function that can be executed in a transaction
type TxFn func(DBTX) error

// WithTransaction executes the given function in a transaction
func WithTransaction(ctx context.Context, db *sql.DB, fn TxFn) error {
	return WithConfiguredTransaction(ctx, db, nil, fn)
}

// WithConfiguredTransaction executes the given function in a transaction with the given options
func WithConfiguredTransaction(ctx context.Context, db *sql.DB, options *sql.TxOptions, fn TxFn) error {
	tx, err := db.BeginTx(ctx, options)
	if err != nil {
		slog.WarnContext(ctx, "Failed to start transaction", "error", err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			slog.ErrorContext(ctx, "Recovered from panic, rolling back transaction and panicking again", "panic", p)

			if txErr := tx.Rollback(); txErr != nil {
				slog.WarnContext(ctx, "Failed to roll back transaction after recovering from panic", "error", txErr)
			}

			panic(p)
		} else if err != nil {
			slog.WarnContext(ctx, "Received error, rolling back transaction", "error", err)

			if txErr := tx.Rollback(); txErr != nil {
				slog.WarnContext(ctx, "Failed to roll back transaction after receiving error", "error", txErr)
			}
		} else {
			err = tx.Commit()
			if err != nil {
				slog.WarnContext(ctx, "Failed to commit transaction", "error", err)
			}
		}
	}()

	err = fn(tx)

	return err
}

// NullInt64FromPtr converts a *int64 to sql.NullInt64
func NullInt64FromPtr(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// NullFloat64FromPtr converts a *float64 to sql.NullFloat64
func NullFloat64FromPtr(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}
