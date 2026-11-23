-- DEMO: notes
-- +goose Up
CREATE TABLE notes (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner      text NOT NULL,
    title      text NOT NULL,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notes_owner_created_at_idx ON notes (owner, created_at DESC);

-- +goose Down
DROP TABLE notes;
