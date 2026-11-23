package db_test

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/driif/go-vibe-starter/internal/test"
	migrations "github.com/driif/go-vibe-starter/sql"
)

const unreachable = "postgres://postgres:postgres@127.0.0.1:1/postgres?sslmode=disable&connect_timeout=1"

func TestOpen(t *testing.T) {
	t.Parallel()
	pool, err := db.Open(t.Context(), test.Database(t))
	require.NoError(t, err)
	defer pool.Close()
	var one int
	require.NoError(t, pool.QueryRow(t.Context(), "SELECT 1").Scan(&one))
	assert.Equal(t, 1, one)

	pool, err = db.Open(t.Context(), unreachable)
	require.Error(t, err)
	assert.Nil(t, pool)
}

func TestMigrate(t *testing.T) {
	t.Parallel()
	url := test.EmptyDatabase(t)
	fixtures := os.DirFS("testdata/migrations")
	var out bytes.Buffer
	require.NoError(t, db.MigrateFS(t.Context(), url, "status", fixtures, &out))
	assert.Contains(t, out.String(), "pending")
	assert.Contains(t, out.String(), "00001_first.sql")
	assert.Contains(t, out.String(), "00002_second.sql")

	out.Reset()
	require.NoError(t, db.MigrateFS(t.Context(), url, "up", fixtures, &out))
	pool, err := db.Open(t.Context(), url)
	require.NoError(t, err)
	defer pool.Close()
	var tables int
	require.NoError(t, pool.QueryRow(t.Context(), "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('first', 'second')").Scan(&tables))
	assert.Equal(t, 2, tables)
	out.Reset()
	require.NoError(t, db.MigrateFS(t.Context(), url, "status", fixtures, &out))
	assert.Contains(t, out.String(), "applied")
	assert.NotContains(t, out.String(), "pending")
	out.Reset()
	require.NoError(t, db.MigrateFS(t.Context(), url, "up", fixtures, &out))
	assert.Empty(t, out.String(), "an already applied migration does no work")

	require.NoError(t, db.MigrateFS(t.Context(), url, "down", fixtures, io.Discard))
	var firstExists, secondMissing bool
	require.NoError(t, pool.QueryRow(t.Context(), "SELECT to_regclass('public.first') IS NOT NULL, to_regclass('public.second') IS NULL").Scan(&firstExists, &secondMissing))
	assert.True(t, firstExists)
	assert.True(t, secondMissing, "down rolls back only the newest migration")
	require.NoError(t, db.MigrateFS(t.Context(), url, "down", fixtures, io.Discard))
	require.NoError(t, db.MigrateFS(t.Context(), url, "down", fixtures, io.Discard), "nothing applied is harmless")
	require.ErrorContains(t, db.MigrateFS(t.Context(), url, "reset", fixtures, io.Discard), "unknown migrate command")
}

func TestMigrate_NoMigrations(t *testing.T) {
	t.Parallel()
	url := test.EmptyDatabase(t)
	for _, tt := range []struct {
		name string
		fsys fs.FS
	}{
		{"empty FS", fstest.MapFS{}},
		{"placeholder only", os.DirFS("testdata/empty")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, command := range []string{"up", "down", "status"} {
				var out bytes.Buffer
				require.NoError(t, db.MigrateFS(t.Context(), url, command, tt.fsys, &out))
				assert.Equal(t, "no migrations\n", out.String())
			}
			require.Error(t, db.MigrateFS(t.Context(), unreachable, "up", tt.fsys, io.Discard))
			require.ErrorContains(t, db.MigrateFS(t.Context(), url, "reset", tt.fsys, io.Discard), "unknown migrate command")
		})
	}
}

func TestMigrate_Embedded(t *testing.T) {
	t.Parallel()
	url := test.EmptyDatabase(t)
	require.NoError(t, db.Migrate(t.Context(), url, "up", io.Discard))
	files, err := fs.Glob(migrations.Migrations, "migrations/*.sql")
	require.NoError(t, err)
	for range files {
		require.NoError(t, db.Migrate(t.Context(), url, "down", io.Discard))
	}
	require.NoError(t, db.Migrate(t.Context(), url, "up", io.Discard))
	pool, err := db.Open(t.Context(), url)
	require.NoError(t, err)
	defer pool.Close()
	require.NoError(t, db.CheckSchema(t.Context(), pool))
}

func TestCheckSchema(t *testing.T) {
	t.Parallel()
	require.NoError(t, db.CheckSchemaFS(t.Context(), nil, fstest.MapFS{}), "no migrations needs no pool")
	first, err := os.ReadFile("testdata/migrations/00001_first.sql")
	require.NoError(t, err)
	firstFS := fstest.MapFS{"00001_first.sql": &fstest.MapFile{Data: first}}
	allFS := os.DirFS("testdata/migrations")
	for _, tt := range []struct {
		name    string
		applied fs.FS
		needed  fs.FS
		wantErr string
	}{
		{"never migrated", nil, allFS, "version 0, this binary needs 2"},
		{"behind", firstFS, allFS, "version 1, this binary needs 2"},
		{"level", allFS, allFS, ""},
		{"ahead", allFS, firstFS, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url := test.EmptyDatabase(t)
			if tt.applied != nil {
				require.NoError(t, db.MigrateFS(t.Context(), url, "up", tt.applied, io.Discard))
			}
			pool, err := db.Open(t.Context(), url)
			require.NoError(t, err)
			defer pool.Close()
			err = db.CheckSchemaFS(t.Context(), pool, tt.needed)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), `run "db migrate up"`)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInvalidDatabaseURLRedacted(t *testing.T) {
	t.Parallel()
	const invalid = "postgres://test:fixture@localhost/example?sslpassword=fixture-query&pool_max_conns=invalid"
	_, err := db.Open(t.Context(), invalid)
	require.ErrorContains(t, err, "DATABASE_URL")
	require.NotContains(t, err.Error(), "fixture")
	err = db.Migrate(t.Context(), invalid, "status", io.Discard)
	require.ErrorContains(t, err, "DATABASE_URL")
	require.NotContains(t, err.Error(), "fixture")
}
