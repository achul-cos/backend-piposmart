-- +goose Up
-- Sprint 15: Add progress tracking to import batches. Allows frontend to show upload progress
-- (percentage completed) without polling for status alone. Initialized to 0 and updated by the
-- validation worker as it processes rows.

ALTER TABLE import_batches ADD COLUMN progress_percentage INT UNSIGNED NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE import_batches DROP COLUMN progress_percentage;
