-- +goose Up
-- Sprint 15: Import Transaksi, Mitra, dan Data Sales. Extends the Sprint 14 import framework
-- with 5 new profiles (NEW_SUBSCRIBE, MONTHLY_ACTIVE, BONUS_MITRA, SALES_CALL_CHAT, SALES_TARGET)
-- and the reconciliation-candidate row status (UNMATCHED).

-- 1. import_batches: allow the 5 new profile values, plus an optional sheet_name — SALES_CALL_CHAT
-- and SALES_TARGET both come from multi-sheet workbooks with several structurally-identical sheets
-- (different sales reps, legacy/duplicate copies), so auto-detecting the right sheet by header
-- markers alone is unsafe; the admin must declare which sheet to use for those two profiles.
--
-- The Sprint 14 dedup key (UNIQUE on file_sha256 alone) assumed one file == one profile, which no
-- longer holds: SALES_CALL_CHAT and SALES_TARGET deliberately re-upload the SAME PBGC file (once
-- per profile+sheet). Re-scope the uniqueness to (file_sha256, profile, sheet_name) so the second
-- upload creates its own batch instead of returning the first one under the wrong profile.
-- sheet_name is NOT NULL DEFAULT '' (not nullable) on purpose: MySQL unique indexes treat every
-- NULL as distinct from every other NULL, which would silently break the existing Sprint 14
-- dedup-by-hash behavior for profiles that never set a sheet_name.
ALTER TABLE import_batches
    ADD COLUMN sheet_name VARCHAR(255) NOT NULL DEFAULT '' AFTER profile;

-- SALES_CALL_CHAT/SALES_TARGET identify their sales rep only via the sheet-name suffix (e.g.
-- "Call & Chat-Lidya"), never a data column — the admin must say which Sales user this batch's
-- rows belong to at upload time.
ALTER TABLE import_batches
    ADD COLUMN target_sales_user_id BIGINT UNSIGNED NULL AFTER sheet_name;

ALTER TABLE import_batches
    ADD CONSTRAINT fk_import_batches_target_sales_user
        FOREIGN KEY (target_sales_user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE import_batches
    DROP KEY uq_import_batches_sha256;

ALTER TABLE import_batches
    ADD UNIQUE KEY uq_import_batches_sha256_profile_sheet (file_sha256, profile, sheet_name);

ALTER TABLE import_batches
    DROP CHECK chk_import_batches_profile;

ALTER TABLE import_batches
    ADD CONSTRAINT chk_import_batches_profile
        CHECK (profile IN (
            'OWNER_OUTLET', 'NON_REGISTER', 'NEW_SUBSCRIBE', 'MONTHLY_ACTIVE',
            'BONUS_MITRA', 'SALES_CALL_CHAT', 'SALES_TARGET', 'PENDING_DETECTION'
        ));

-- 2. import_rows: add UNMATCHED — a structurally valid row that references an owner/outlet/
-- partner/package/sales-user code not found in the DB. Unlike Sprint 14's auto-create behavior,
-- Sprint 15's transactional profiles assume the referenced entity already exists; a miss goes to
-- manual reconciliation (POST /imports/:id/rows/:row_id/relink) instead of being papered over.
ALTER TABLE import_rows
    DROP CHECK chk_import_rows_status;

ALTER TABLE import_rows
    ADD CONSTRAINT chk_import_rows_status
        CHECK (status IN ('PENDING', 'VALID', 'INVALID', 'UNMATCHED', 'COMMITTED', 'COMMIT_FAILED'));

-- 3. outlet_monthly_activity_snapshot — MONTHLY_ACTIVE's melted per-(outlet, month) fact table.
-- Deliberately append-only and separate from `subscriptions`/`subscription_periods`: the source
-- workbook's monthly codes (e.g. "F-BC(6)") only say "active this month", never an exact start/end
-- date, so backfilling precise subscription periods from them would be a guess, not a fact.
CREATE TABLE outlet_monthly_activity_snapshot (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    outlet_id BIGINT UNSIGNED NOT NULL,
    period_year SMALLINT UNSIGNED NOT NULL,
    period_month TINYINT UNSIGNED NOT NULL,
    raw_code VARCHAR(30) NOT NULL,
    category VARCHAR(30) NOT NULL,
    package_code VARCHAR(20) NULL,
    tenor_months TINYINT UNSIGNED NULL,
    import_batch_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_outlet_monthly_activity (outlet_id, period_year, period_month),
    KEY idx_outlet_monthly_activity_period (period_year, period_month),
    CONSTRAINT fk_outlet_monthly_activity_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id) ON DELETE CASCADE,
    CONSTRAINT fk_outlet_monthly_activity_batch
        FOREIGN KEY (import_batch_id) REFERENCES import_batches(id) ON DELETE CASCADE,
    CONSTRAINT chk_outlet_monthly_activity_month
        CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT chk_outlet_monthly_activity_category
        CHECK (category IN (
            'NEW', 'SUBSCRIBE', 'FOLLOWING', 'UNSUBSCRIBE', 'REACTIVATE', 'TESTING',
            'NOT_ACTIVATED', 'CLIPPER_ACTIVE', 'CLIPPER_INACTIVE'
        ))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS outlet_monthly_activity_snapshot;

ALTER TABLE import_rows
    DROP CHECK chk_import_rows_status;
ALTER TABLE import_rows
    ADD CONSTRAINT chk_import_rows_status
        CHECK (status IN ('PENDING', 'VALID', 'INVALID', 'COMMITTED', 'COMMIT_FAILED'));

ALTER TABLE import_batches
    DROP CHECK chk_import_batches_profile;
ALTER TABLE import_batches
    ADD CONSTRAINT chk_import_batches_profile
        CHECK (profile IN ('OWNER_OUTLET', 'NON_REGISTER', 'PENDING_DETECTION'));

ALTER TABLE import_batches
    DROP KEY uq_import_batches_sha256_profile_sheet;
ALTER TABLE import_batches
    ADD UNIQUE KEY uq_import_batches_sha256 (file_sha256);

ALTER TABLE import_batches
    DROP FOREIGN KEY fk_import_batches_target_sales_user;
ALTER TABLE import_batches
    DROP COLUMN target_sales_user_id;

ALTER TABLE import_batches
    DROP COLUMN sheet_name;
