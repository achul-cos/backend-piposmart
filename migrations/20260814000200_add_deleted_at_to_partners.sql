-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE partners ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL AFTER status;
ALTER TABLE partners ADD INDEX idx_partners_deleted_at (deleted_at);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE partners DROP INDEX idx_partners_deleted_at;
ALTER TABLE partners DROP COLUMN deleted_at;
