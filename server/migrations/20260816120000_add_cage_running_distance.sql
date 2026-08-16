-- +goose Up
ALTER TABLE users ADD COLUMN cage_running_distance_meters INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN cage_running_distance_meters;
