-- +goose Up
ALTER TABLE import_batches
    ADD COLUMN file_blob LONGBLOB NULL AFTER file_path;

-- +goose Down
ALTER TABLE import_batches
    DROP COLUMN file_blob;
