// DEMO: notes

package notes_test

import (
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	"github.com/driif/go-vibe-starter/internal/test"
	"github.com/stretchr/testify/require"
)

func TestNotes(t *testing.T) {
	t.Parallel()
	s := test.NewServer(t)
	ctx := t.Context()
	alice := apiv1connect.NewNotesServiceClient(s.Client(s.Token("alice")))
	bob := apiv1connect.NewNotesServiceClient(s.Client(s.Token("bob")))
	created, err := alice.CreateNote(ctx, &apiv1.CreateNoteRequest{Title: "First", Body: "hello"})
	require.NoError(t, err)
	id := created.GetNote().GetId()
	fetched, err := alice.GetNote(ctx, &apiv1.GetNoteRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, "hello", fetched.GetNote().GetBody())
	var owner string
	require.NoError(t, s.DB.QueryRow(ctx, "SELECT owner FROM notes WHERE id=$1", id).Scan(&owner))
	require.Equal(t, "alice", owner)
	listed, err := bob.ListNotes(ctx, &apiv1.ListNotesRequest{})
	require.NoError(t, err)
	require.Empty(t, listed.GetNotes())
	_, err = bob.GetNote(ctx, &apiv1.GetNoteRequest{Id: id})
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Equal(t, "NOTE_NOT_FOUND", test.Reason(err))
	_, err = bob.DeleteNote(ctx, &apiv1.DeleteNoteRequest{Id: id})
	require.Equal(t, "NOTE_NOT_FOUND", test.Reason(err))
	listed, err = alice.ListNotes(ctx, &apiv1.ListNotesRequest{})
	require.NoError(t, err)
	require.Len(t, listed.GetNotes(), 1)
	_, err = alice.DeleteNote(ctx, &apiv1.DeleteNoteRequest{Id: id})
	require.NoError(t, err)
	_, err = alice.GetNote(ctx, &apiv1.GetNoteRequest{Id: id})
	require.Equal(t, "NOTE_NOT_FOUND", test.Reason(err))
	for _, title := range []string{"", strings.Repeat("a", 201), "nul\x00title"} {
		_, err = alice.CreateNote(ctx, &apiv1.CreateNoteRequest{Title: title})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Contains(t, test.Violations(t, err), "title")
	}
	_, err = alice.CreateNote(ctx, &apiv1.CreateNoteRequest{Title: "valid", Body: "a\x00b"})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, test.Violations(t, err), "body")
	_, err = alice.GetNote(ctx, &apiv1.GetNoteRequest{Id: "invalid"})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, test.Violations(t, err), "id")
	anon := apiv1connect.NewNotesServiceClient(s.Client(""))
	_, err = anon.ListNotes(ctx, &apiv1.ListNotesRequest{})
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
