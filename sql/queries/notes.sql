-- DEMO: reference vertical slice — delete with `make init` or by hand.

-- name: CreateNote :one
-- CreateNote inserts a note owned by the caller. The id and timestamps come
-- from the column defaults.
INSERT INTO notes (owner_id, title, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetNote :one
-- GetNote matches on owner_id as well as id, so a caller can never address
-- another caller's row: a foreign id matches nothing.
SELECT * FROM notes
WHERE id = $1 AND owner_id = $2;

-- name: ListNotesByOwner :many
-- ListNotesByOwner returns the caller's notes, newest first.
SELECT * FROM notes
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: DeleteNote :exec
-- DeleteNote is scoped to the owner, so deleting a foreign or missing id is a
-- no-op rather than an error.
DELETE FROM notes
WHERE id = $1 AND owner_id = $2;
