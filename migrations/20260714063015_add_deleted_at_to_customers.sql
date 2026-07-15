-- +goose Up
ALTER TABLE customers
ADD COLUMN deleted_at TIMESTAMP NULL;

-- +goose Down
ALTER TABLE customers
DROP COLUMN deleted_at;