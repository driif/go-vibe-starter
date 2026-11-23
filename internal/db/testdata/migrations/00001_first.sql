-- +goose Up
CREATE TABLE first (id int PRIMARY KEY);

-- +goose Down
DROP TABLE first;
