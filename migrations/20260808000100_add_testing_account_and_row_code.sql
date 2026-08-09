-- +goose Up
-- "Kode Baris" was a raw per-row identifier from the original "01. Owner & Outlet" source
-- spreadsheet, tracked per outlet-row (a single owner can have several rows/outlets, each with its
-- own Kode Baris) — never persisted anywhere during the owner/outlet migration, so it's added here.
--
-- is_testing_account marks an owner whose data only exists because a piposmart employee installed
-- the app to learn/demo it (not a real prospective customer) — per admin request, these must never
-- be shared to sales for call/chat/closing. testing_marked_by/testing_marked_at audit who flagged it
-- and when (an admin action going forward; NULL for accounts inferred as testing purely from the
-- original "Kategori Akun" = "Akun Testing" import data, since no specific admin click happened).
ALTER TABLE owners
    ADD COLUMN is_testing_account TINYINT(1) NOT NULL DEFAULT 0 AFTER status,
    ADD COLUMN testing_marked_by_user_id BIGINT UNSIGNED NULL AFTER is_testing_account,
    ADD COLUMN testing_marked_at DATETIME NULL AFTER testing_marked_by_user_id,
    ADD KEY idx_owners_is_testing_account (is_testing_account),
    ADD CONSTRAINT fk_owners_testing_marked_by
        FOREIGN KEY (testing_marked_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL;

ALTER TABLE outlets
    ADD COLUMN row_code VARCHAR(60) NULL AFTER code;

-- +goose Down
ALTER TABLE outlets
    DROP COLUMN row_code;

ALTER TABLE owners
    DROP FOREIGN KEY fk_owners_testing_marked_by,
    DROP KEY idx_owners_is_testing_account,
    DROP COLUMN testing_marked_at,
    DROP COLUMN testing_marked_by_user_id,
    DROP COLUMN is_testing_account;
