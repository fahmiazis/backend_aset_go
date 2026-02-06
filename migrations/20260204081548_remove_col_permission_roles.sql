-- +goose Up
ALTER TABLE roles DROP COLUMN permissions;

-- +goose Down
ALTER TABLE roles ADD COLUMN permissions TEXT AFTER description;