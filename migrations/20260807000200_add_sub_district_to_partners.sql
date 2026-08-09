-- +goose Up
ALTER TABLE partners 
    ADD COLUMN sub_district VARCHAR(120) NULL AFTER district;

-- +goose Down
ALTER TABLE partners 
    DROP COLUMN sub_district;
