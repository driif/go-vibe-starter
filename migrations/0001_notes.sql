-- DEMO: reference vertical slice — delete with `make init` or by hand.

-- +goose Up
CREATE TABLE notes (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   text NOT NULL,
    title      text NOT NULL,
    body       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- every read is scoped to one owner, so that filter carries the index
CREATE INDEX notes_owner_id_idx ON notes (owner_id);

-- +goose Down
DROP TABLE notes;
