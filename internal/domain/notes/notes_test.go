// DEMO: notes

package notes_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/driif/go-vibe-starter/internal/domain/notes"
	"github.com/driif/go-vibe-starter/internal/test"
)

func TestStore_List(t *testing.T) {
	t.Parallel()
	pool, err := db.Open(t.Context(), test.Database(t))
	require.NoError(t, err)
	defer pool.Close()
	store := notes.NewStore(pool)

	for i := range 101 {
		_, err := store.Create(t.Context(), "alice", fmt.Sprintf("note %d", i), "")
		require.NoError(t, err)
	}
	_, err = store.Create(t.Context(), "bob", "private", "")
	require.NoError(t, err)

	list, err := store.List(t.Context(), "alice")
	require.NoError(t, err)
	require.Len(t, list, 100)
	for i, n := range list {
		assert.Equal(t, fmt.Sprintf("note %d", 100-i), n.Title, "newest first")
		assert.Equal(t, "alice", n.Owner)
	}
}
