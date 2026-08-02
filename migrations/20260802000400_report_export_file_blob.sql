-- +goose Up
ALTER TABLE report_exports
    ADD COLUMN file_blob LONGBLOB NULL AFTER mime_type;

-- +goose Down
ALTER TABLE report_exports
    DROP COLUMN file_blob;
