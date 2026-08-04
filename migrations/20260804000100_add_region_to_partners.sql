-- +goose Up
ALTER TABLE partners 
    ADD COLUMN province VARCHAR(120) NULL AFTER name, 
    ADD COLUMN city VARCHAR(120) NULL AFTER province, 
    ADD COLUMN district VARCHAR(120) NULL AFTER city;

-- +goose Down
ALTER TABLE partners 
    DROP COLUMN province, 
    DROP COLUMN city, 
    DROP COLUMN district;
