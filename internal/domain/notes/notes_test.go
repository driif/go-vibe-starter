// DEMO: notes

package notes_test

import (
	"fmt"
	"testing"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/driif/go-vibe-starter/internal/domain/notes"
	"github.com/driif/go-vibe-starter/internal/test"
	"github.com/stretchr/testify/require"
)

func TestListNewestHundred(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	pool, err := db.Open(ctx, test.Database(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	store := notes.NewStore(pool)
	for i := range 101 {
		_, err = store.Create(ctx, "alice", fmt.Sprintf("note %d", i), "")
		require.NoError(t, err)
	}
	_, err = store.Create(ctx, "bob", "private", "")
	require.NoError(t, err)
	rows, err := store.List(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, rows, 100)
	for i, row := range rows {
		require.Equal(t, fmt.Sprintf("note %d", 100-i), row.Title)
	}
}
