-- +goose Up
-- Recover older OWNER_OUTLET imports that appended ", Kel. ... , Kec. ..." into address instead
-- of persisting district/sub_district separately.
UPDATE owners
SET
    sub_district = CASE
        WHEN COALESCE(sub_district, '') = '' AND address LIKE '%, Kel. %'
            THEN LEFT(TRIM(SUBSTRING_INDEX(SUBSTRING_INDEX(address, ', Kel. ', -1), ', Kec. ', 1)), 50)
        ELSE sub_district
    END,
    district = CASE
        WHEN COALESCE(district, '') = '' AND address LIKE '%, Kec. %'
            THEN LEFT(TRIM(SUBSTRING_INDEX(address, ', Kec. ', -1)), 50)
        ELSE district
    END,
    address = CASE
        WHEN address LIKE '%, Kel. %'
            THEN TRIM(SUBSTRING_INDEX(address, ', Kel. ', 1))
        WHEN address LIKE '%, Kec. %'
            THEN TRIM(SUBSTRING_INDEX(address, ', Kec. ', 1))
        ELSE address
    END
WHERE
    (COALESCE(district, '') = '' OR COALESCE(sub_district, '') = '')
    AND address IS NOT NULL
    AND (address LIKE '%, Kel. %' OR address LIKE '%, Kec. %');

UPDATE outlets
SET
    sub_district = CASE
        WHEN COALESCE(sub_district, '') = '' AND address LIKE '%, Kel. %'
            THEN LEFT(TRIM(SUBSTRING_INDEX(SUBSTRING_INDEX(address, ', Kel. ', -1), ', Kec. ', 1)), 50)
        ELSE sub_district
    END,
    district = CASE
        WHEN COALESCE(district, '') = '' AND address LIKE '%, Kec. %'
            THEN LEFT(TRIM(SUBSTRING_INDEX(address, ', Kec. ', -1)), 50)
        ELSE district
    END,
    address = CASE
        WHEN address LIKE '%, Kel. %'
            THEN TRIM(SUBSTRING_INDEX(address, ', Kel. ', 1))
        WHEN address LIKE '%, Kec. %'
            THEN TRIM(SUBSTRING_INDEX(address, ', Kec. ', 1))
        ELSE address
    END
WHERE
    (COALESCE(district, '') = '' OR COALESCE(sub_district, '') = '')
    AND address IS NOT NULL
    AND (address LIKE '%, Kel. %' OR address LIKE '%, Kec. %');

-- +goose Down
-- Irreversible data cleanup/backfill; no-op on rollback.
SELECT 1;
