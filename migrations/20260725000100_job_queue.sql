-- +goose Up
-- Generic MySQL-backed job queue (no Redis) — locked technical decision since Sprint 1.
-- WorkerConfig (PollInterval, MaxAttempts, StaleJobTimeout) has existed unused since baseline;
-- this table is what it was scaffolded for. First consumer is Sprint 13's KPI recompute job;
-- Sprint 14's "Background job import" reuses this same table with a new job_type, not a new
-- queue mechanism.
CREATE TABLE job_queue (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_type VARCHAR(60) NOT NULL,
    payload JSON NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempts INT UNSIGNED NOT NULL DEFAULT 0,
    max_attempts INT UNSIGNED NOT NULL DEFAULT 5,
    last_error TEXT NULL,
    available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    claimed_at DATETIME NULL,
    completed_at DATETIME NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_job_queue_dispatch (status, available_at),
    KEY idx_job_queue_type (job_type, status),
    CONSTRAINT fk_job_queue_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_job_queue_status
        CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS job_queue;
