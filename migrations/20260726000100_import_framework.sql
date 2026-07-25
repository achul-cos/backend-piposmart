-- +goose Up
-- Sprint 14: Import Framework dan Data Customer. Generic staging pipeline: upload -> validate
-- (async via job_queue) -> preview -> commit (async via job_queue). Reuses the Sprint 13 job
-- queue mechanism (internal/platform/jobqueue) with two new job types (IMPORT_VALIDATE,
-- IMPORT_COMMIT) rather than inventing a second queue.

-- 1. import_batches — one row per uploaded file. file_sha256 is UNIQUE so re-uploading the same
-- file returns the existing batch instead of creating a duplicate.
CREATE TABLE import_batches (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(60) NOT NULL,
    profile VARCHAR(30) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_sha256 CHAR(64) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'UPLOADED',
    total_rows INT UNSIGNED NOT NULL DEFAULT 0,
    valid_rows INT UNSIGNED NOT NULL DEFAULT 0,
    invalid_rows INT UNSIGNED NOT NULL DEFAULT 0,
    committed_rows INT UNSIGNED NOT NULL DEFAULT 0,
    validate_job_id BIGINT UNSIGNED NULL,
    commit_job_id BIGINT UNSIGNED NULL,
    error_message TEXT NULL,
    uploaded_by_user_id BIGINT UNSIGNED NOT NULL,
    committed_by_user_id BIGINT UNSIGNED NULL,
    uploaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    validated_at DATETIME NULL,
    committed_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_import_batches_sha256 (file_sha256),
    UNIQUE KEY uq_import_batches_code (code),
    KEY idx_import_batches_status (status),
    CONSTRAINT fk_import_batches_uploaded_by
        FOREIGN KEY (uploaded_by_user_id) REFERENCES users(id),
    CONSTRAINT fk_import_batches_committed_by
        FOREIGN KEY (committed_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_import_batches_validate_job
        FOREIGN KEY (validate_job_id) REFERENCES job_queue(id) ON DELETE SET NULL,
    CONSTRAINT fk_import_batches_commit_job
        FOREIGN KEY (commit_job_id) REFERENCES job_queue(id) ON DELETE SET NULL,
    CONSTRAINT chk_import_batches_profile
        CHECK (profile IN ('OWNER_OUTLET', 'NON_REGISTER', 'PENDING_DETECTION')),
    CONSTRAINT chk_import_batches_status
        CHECK (status IN ('UPLOADED', 'VALIDATING', 'VALIDATED', 'VALIDATION_FAILED', 'COMMITTING', 'COMMITTED', 'COMMIT_FAILED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. import_rows — one row per Excel data row. raw_payload never contains OTP-related columns;
-- they are stripped during parsing before this row is ever built (see internal/importing/excel.go).
CREATE TABLE import_rows (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    batch_id BIGINT UNSIGNED NOT NULL,
    row_index INT UNSIGNED NOT NULL,
    raw_payload JSON NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    validation_errors JSON NULL,
    owner_id BIGINT UNSIGNED NULL,
    outlet_id BIGINT UNSIGNED NULL,
    lead_id BIGINT UNSIGNED NULL,
    commit_error TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_import_rows_batch_row (batch_id, row_index),
    KEY idx_import_rows_batch_status (batch_id, status),
    CONSTRAINT fk_import_rows_batch
        FOREIGN KEY (batch_id) REFERENCES import_batches(id) ON DELETE CASCADE,
    CONSTRAINT fk_import_rows_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id) ON DELETE SET NULL,
    CONSTRAINT fk_import_rows_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id) ON DELETE SET NULL,
    CONSTRAINT fk_import_rows_lead
        FOREIGN KEY (lead_id) REFERENCES customer_leads(id) ON DELETE SET NULL,
    CONSTRAINT chk_import_rows_status
        CHECK (status IN ('PENDING', 'VALID', 'INVALID', 'COMMITTED', 'COMMIT_FAILED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS import_rows;
DROP TABLE IF EXISTS import_batches;
