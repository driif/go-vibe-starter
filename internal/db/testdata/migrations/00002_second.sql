-- +goose Up
CREATE TABLE second (id int PRIMARY KEY);

-- +goose Down
DROP TABLE second;
