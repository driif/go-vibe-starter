// Package db opens the Postgres pool and runs the embedded goose migrations. It never imports
// internal/db/gen, so it compiles in a tree without queries. A use case that needs a transaction
// calls pgx.BeginFunc(ctx, pool, fn) in its domain package.
package db

import (
	"context"

	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	migrations "github.com/driif/go-vibe-starter/sql"
)

// Open creates a pool with the pgxpool defaults, which pool_* parameters of url override, and
// pings it.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Migrate runs command ("up", "down" or "status") with the embedded migrations and writes what it
// did to out. With no migration files every command prints "no migrations" and only pings.
func Migrate(ctx context.Context, url, command string, out io.Writer) error {
	return migrate(ctx, url, command, embedded(), out)
}

// CheckSchema returns an error when the database is at a lower migration version than the
// highest embedded one. A newer database passes, so an older binary can run during a rollback.
func CheckSchema(ctx context.Context, pool *pgxpool.Pool) error {
	return checkSchema(ctx, pool, embedded())
}

func embedded() fs.FS {
	fsys, err := fs.Sub(migrations.Migrations, "migrations")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	return fsys
}

func migrate(ctx context.Context, url, command string, fsys fs.FS, out io.Writer) error {
	if command != "up" && command != "down" && command != "status" {
		return fmt.Errorf("unknown migrate command %q", command)
	}
	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return err
	}
	conn := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer conn.Close()

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, conn, fsys, goose.WithSessionLocker(locker))
	if errors.Is(err, goose.ErrNoMigrations) {
		_, _ = fmt.Fprintln(out, "no migrations")
		return conn.PingContext(ctx)
	}
	if err != nil {
		return err
	}

	switch command {
	case "up":
		results, err := provider.Up(ctx)
		for _, r := range results {
			_, _ = fmt.Fprintln(out, r)
		}
		return err
	case "down":
		result, err := provider.Down(ctx)
		if errors.Is(err, goose.ErrNoNextVersion) {
			return nil
		}
		if result != nil {
			_, _ = fmt.Fprintln(out, result)
		}
		return err
	case "status":
		statuses, err := provider.Status(ctx)
		for _, s := range statuses {
			_, _ = fmt.Fprintf(out, "%-8s %s\n", s.State, path.Base(s.Source.Path))
		}
		return err
	}
	return fmt.Errorf("unknown migrate command %q", command)
}

func checkSchema(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	need, err := latestVersion(fsys)
	if err != nil || need == 0 {
		return err
	}
	var have int64
	err = pool.QueryRow(ctx, "SELECT coalesce(max(version_id), 0) FROM goose_db_version").Scan(&have)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		err = nil
	}
	if err != nil {
		return err
	}
	if have < need {
		return fmt.Errorf("database schema is at version %d, this binary needs %d: run \"db migrate up\"", have, need)
	}
	return nil
}

func latestVersion(fsys fs.FS) (int64, error) {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, f := range files {
		v, err := goose.NumericComponent(f)
		if err != nil {
			return 0, err
		}
		latest = max(latest, v)
	}
	return latest, nil
}
