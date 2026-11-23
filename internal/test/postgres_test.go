package test_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/driif/go-vibe-starter/internal/test"
)

func TestDatabase_Parallel(t *testing.T) {
	t.Parallel()
	for range 8 {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			pool, err := db.Open(t.Context(), test.Database(t))
			require.NoError(t, err)
			defer pool.Close()

			_, err = pool.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS probe (name text)")
			require.NoError(t, err)
			_, err = pool.Exec(t.Context(), "INSERT INTO probe VALUES ($1)", t.Name())
			require.NoError(t, err)
			var names []string
			rows, err := pool.Query(t.Context(), "SELECT name FROM probe")
			require.NoError(t, err)
			for rows.Next() {
				var name string
				require.NoError(t, rows.Scan(&name))
				names = append(names, name)
			}
			require.NoError(t, rows.Err())
			assert.Equal(t, []string{t.Name()}, names)
		})
	}
}

func TestDatabase_LongName(t *testing.T) {
	t.Parallel()
	t.Run(strings.Repeat("a very long subtest name ", 5), func(t *testing.T) {
		pool, err := db.Open(t.Context(), test.Database(t))
		require.NoError(t, err)
		defer pool.Close()

		var name string
		require.NoError(t, pool.QueryRow(t.Context(), "SELECT current_database()").Scan(&name))
		assert.Len(t, name, 63)
	})
}
