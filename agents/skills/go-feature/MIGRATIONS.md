# Migrations (step 1 of the slice)

`migrations/` is the schema. Nothing else describes it: sqlc reads this directory as its schema
(`sqlc.yaml`), and the running database is only ever changed by applying a file from here. Worked
example: `migrations/0001_notes.sql`.

## The file

Name it `NNNN_snake_name.sql`, `NNNN` being the next zero-padded number in `migrations/` (`0002`
after `0001`). goose orders by that number, so it is the version.

```sql
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
```

Wrap a statement in `-- +goose StatementBegin` / `-- +goose StatementEnd` when it contains its own
semicolons — `CREATE FUNCTION`, a `DO $$ ... $$` block, a trigger body. goose splits on semicolons
otherwise and feeds the driver half a statement.

```sql
-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION touch_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
```

`Down` reverses `Up` exactly: the table `Up` created is dropped, the column it added is dropped, the
index it built is dropped. Write it as you write `Up`, while the change is in front of you.

## Schema conventions

- `uuid PRIMARY KEY DEFAULT gen_random_uuid()` for identifiers, `timestamptz` for times, `text` over
  `varchar(n)`.
- `NOT NULL` with a `DEFAULT` wherever the column has a sensible zero, so sqlc generates `string`
  rather than `sql.NullString`.
- An owner column (`owner_id text NOT NULL`) on every table a caller can read, plus an index on it:
  ownership is enforced in the `WHERE` clause of every query, so that column is in every plan.
- One migration per logical change. A migration that both adds a table and backfills another is two
  migrations.

## Applying it

```bash
make db-up                      # postgres via docker compose
make migrate-up                 # ./bin/app db migrate up
./bin/app db migrate status     # what is applied
./bin/app db migrate down       # roll back one version
```

Round-trip the new file — `up`, `down`, `up` — before moving on. That is the only proof `Down`
works, and the moment it fails is the cheapest moment to fix it.

Change the schema by adding a migration, never by running SQL against the database by hand and never
by editing a file that has already been applied. An edited applied file leaves every other machine
on the old schema with the new checksum, and sqlc then generates against a schema nobody is running.

## Before step 2

sqlc parses the `Up` direction of every file in `migrations/` to learn the types. The migration file
must exist on disk before `make sqlc` runs, or the query referencing the new column fails to
generate with `column not found`.
