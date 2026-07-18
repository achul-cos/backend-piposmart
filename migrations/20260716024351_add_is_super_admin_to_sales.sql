-- +goose Up
ALTER TABLE sales
ADD COLUMN is_super_admin BOOLEAN;

-- +goose Down
ALTER TABLE sales
DROP COLUMN is_super_admin;
