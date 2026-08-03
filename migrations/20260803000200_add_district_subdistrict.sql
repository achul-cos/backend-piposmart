-- +goose Up
ALTER TABLE owners
ADD COLUMN district VARCHAR(120) NULL AFTER city,
ADD COLUMN sub_district VARCHAR(120) NULL AFTER district;

ALTER TABLE outlets
ADD COLUMN district VARCHAR(120) NULL AFTER city,
ADD COLUMN sub_district VARCHAR(120) NULL AFTER district;

-- +goose Down
ALTER TABLE owners
DROP COLUMN sub_district,
DROP COLUMN district;

ALTER TABLE outlets
DROP COLUMN sub_district,
DROP COLUMN district;
