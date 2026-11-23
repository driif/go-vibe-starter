-- DEMO: notes

-- name: CreateNote :one
INSERT INTO notes (owner, title, body)
VALUES (@owner, @title, @body)
RETURNING *;

-- name: GetNote :one
SELECT * FROM notes
WHERE id = @id AND owner = @owner;

-- name: ListNotes :many
SELECT * FROM notes
WHERE owner = @owner
ORDER BY created_at DESC, id DESC
LIMIT 100;

-- name: DeleteNote :execrows
DELETE FROM notes
WHERE id = @id AND owner = @owner;
