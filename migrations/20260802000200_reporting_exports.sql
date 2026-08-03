-- +goose Up
CREATE TABLE report_exports (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    report_key VARCHAR(60) NOT NULL,
    format VARCHAR(10) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    filters_json JSON NULL,
    requested_by_user_id BIGINT UNSIGNED NULL,
    job_id BIGINT UNSIGNED NULL,
    file_path VARCHAR(255) NULL,
    file_name VARCHAR(255) NULL,
    mime_type VARCHAR(160) NULL,
    file_blob LONGBLOB NULL,
    storage_disk VARCHAR(20) NOT NULL DEFAULT 'local',
    row_count INT UNSIGNED NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    completed_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_report_exports_code (code),
    KEY idx_report_exports_report_status (report_key, status),
    KEY idx_report_exports_requested_by (requested_by_user_id, created_at),
    KEY idx_report_exports_job (job_id),
    CONSTRAINT fk_report_exports_requested_by
        FOREIGN KEY (requested_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_report_exports_format
        CHECK (format IN ('CSV', 'XLSX')),
    CONSTRAINT chk_report_exports_status
        CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS report_exports;
