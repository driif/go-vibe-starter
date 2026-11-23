# Migrations

Write `sql/migrations/NNNNN_snake_name.sql`, five digits, with both goose sections.

```sql
-- +goose Up
CREATE TABLE tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner text NOT NULL,
    title text NOT NULL,
    create_time timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tasks_owner ON tasks (owner);

-- +goose Down
DROP TABLE tasks;
```

Use `make migrate-up`, `make migrate-down`, `make migrate-status`. The migrations are embedded;
adding one needs a rebuild, which the Make targets do. `run` refuses an older schema. Design rollout
compatibility and a meaningful Down; do not edit applied migrations or execute ad-hoc schema SQL.
