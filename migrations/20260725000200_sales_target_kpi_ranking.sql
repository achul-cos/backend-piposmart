-- +goose Up
-- Sprint 13: Sales Target, KPI, dan Ranking. Target dan KPI Definition memakai metric_codes
-- (seeded sejak Sprint 2: CALL_CUSTOMER_COUNT, TRAINING_COUNT, CONFIRMED_CLOSING_COUNT,
-- CONFIRMED_CLOSING_AMOUNT, PARTNER_CALL_COUNT) sebagai sumber metric, bukan enum baru.

-- 1. sales_targets — bulk (INSERT IGNORE, tidak pernah menimpa) vs override (selalu menang),
-- dibedakan lewat kolom source untuk audit trail, bukan lewat tabel terpisah.
CREATE TABLE sales_targets (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sales_id BIGINT UNSIGNED NOT NULL,
    metric_code_id BIGINT UNSIGNED NOT NULL,
    period_year SMALLINT UNSIGNED NOT NULL,
    period_month TINYINT UNSIGNED NOT NULL,
    target_value DECIMAL(18,2) NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'BULK',
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_sales_targets_period (sales_id, metric_code_id, period_year, period_month),
    KEY idx_sales_targets_period (period_year, period_month),
    CONSTRAINT fk_sales_targets_sales
        FOREIGN KEY (sales_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_sales_targets_metric
        FOREIGN KEY (metric_code_id) REFERENCES metric_codes(id),
    CONSTRAINT fk_sales_targets_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_sales_targets_month
        CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT chk_sales_targets_value
        CHECK (target_value >= 0),
    CONSTRAINT chk_sales_targets_source
        CHECK (source IN ('BULK', 'OVERRIDE'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. kpi_definitions — per periode & metric. "Weight aktif per periode harus 100%" divalidasi
-- di service layer saat recompute (agregat lintas baris, tidak bisa jadi CHECK constraint).
CREATE TABLE kpi_definitions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    metric_code_id BIGINT UNSIGNED NOT NULL,
    period_year SMALLINT UNSIGNED NOT NULL,
    period_month TINYINT UNSIGNED NOT NULL,
    weight DECIMAL(5,2) NOT NULL,
    threshold_achieved DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    threshold_near DECIMAL(5,2) NOT NULL DEFAULT 80.00,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_kpi_definitions_period_metric (period_year, period_month, metric_code_id),
    KEY idx_kpi_definitions_period_active (period_year, period_month, active),
    CONSTRAINT fk_kpi_definitions_metric
        FOREIGN KEY (metric_code_id) REFERENCES metric_codes(id),
    CONSTRAINT fk_kpi_definitions_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_kpi_definitions_month
        CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT chk_kpi_definitions_weight
        CHECK (weight >= 0 AND weight <= 100),
    CONSTRAINT chk_kpi_definitions_threshold
        CHECK (threshold_near >= 0 AND threshold_near <= threshold_achieved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. sales_kpi_metric_results — per-metric detail snapshot of one recompute run.
CREATE TABLE sales_kpi_metric_results (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sales_id BIGINT UNSIGNED NOT NULL,
    kpi_definition_id BIGINT UNSIGNED NOT NULL,
    period_year SMALLINT UNSIGNED NOT NULL,
    period_month TINYINT UNSIGNED NOT NULL,
    target_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    actual_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    achievement_pct DECIMAL(7,2) NOT NULL DEFAULT 0,
    weighted_score DECIMAL(7,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_kpi_metric_results (sales_id, kpi_definition_id, period_year, period_month),
    KEY idx_kpi_metric_results_period (period_year, period_month),
    CONSTRAINT fk_kpi_metric_results_sales
        FOREIGN KEY (sales_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_kpi_metric_results_definition
        FOREIGN KEY (kpi_definition_id) REFERENCES kpi_definitions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. sales_kpi_results — overall per-sales-per-period summary + rank. Recompute is
-- delete-then-insert scoped to (period_year, period_month) inside one transaction, so
-- re-running for the same period is idempotent by construction.
CREATE TABLE sales_kpi_results (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sales_id BIGINT UNSIGNED NOT NULL,
    period_year SMALLINT UNSIGNED NOT NULL,
    period_month TINYINT UNSIGNED NOT NULL,
    total_score DECIMAL(7,2) NOT NULL DEFAULT 0,
    classification VARCHAR(20) NOT NULL,
    rank_position INT UNSIGNED NULL,
    computed_at DATETIME NOT NULL,
    job_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_sales_kpi_results_period (sales_id, period_year, period_month),
    KEY idx_sales_kpi_results_period_rank (period_year, period_month, rank_position),
    CONSTRAINT fk_sales_kpi_results_sales
        FOREIGN KEY (sales_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_sales_kpi_results_job
        FOREIGN KEY (job_id) REFERENCES job_queue(id) ON DELETE SET NULL,
    CONSTRAINT chk_sales_kpi_results_classification
        CHECK (classification IN ('ACHIEVED', 'NEAR_ACHIEVED', 'NOT_ACHIEVED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS sales_kpi_results;
DROP TABLE IF EXISTS sales_kpi_metric_results;
DROP TABLE IF EXISTS kpi_definitions;
DROP TABLE IF EXISTS sales_targets;
