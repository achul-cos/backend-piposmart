-- +goose Up
ALTER TABLE sales
ADD COLUMN deleted_at TIMESTAMP NULL;

-- +goose Down
ALTER TABLE sales
DROP COLUMN deleted_at;
