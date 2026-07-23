-- +goose Up
CREATE TABLE sales_closings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    lead_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    outlet_id BIGINT UNSIGNED NULL,
    sales_id BIGINT UNSIGNED NULL,
    supervisor_id BIGINT UNSIGNED NULL,
    package_id BIGINT UNSIGNED NULL,
    plan_id BIGINT UNSIGNED NULL,
    promotion_id BIGINT UNSIGNED NULL,
    package_snapshot_json JSON NOT NULL,
    plan_snapshot_json JSON NOT NULL,
    promotion_snapshot_json JSON NULL,
    tenure_months INT NOT NULL,
    duration_days INT NOT NULL,
    base_price DECIMAL(18,2) NOT NULL,
    discount_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    additional_charge DECIMAL(18,2) NOT NULL DEFAULT 0,
    unique_transfer_code INT NOT NULL DEFAULT 0,
    final_amount DECIMAL(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(40) NOT NULL DEFAULT 'PENDING_RECONCILIATION',
    note TEXT NULL,
    rejection_reason TEXT NULL,
    closed_at DATETIME(6) NOT NULL,
    confirmed_at DATETIME(6) NULL,
    rejected_at DATETIME(6) NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    updated_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_sales_closings_code (code),
    KEY idx_sales_closings_lead_status (lead_id, status),
    KEY idx_sales_closings_owner_closed (owner_id, closed_at),
    KEY idx_sales_closings_sales_closed (sales_id, closed_at),
    KEY idx_sales_closings_supervisor_closed (supervisor_id, closed_at),
    KEY idx_sales_closings_status_closed (status, closed_at),
    KEY idx_sales_closings_plan_closed (plan_id, closed_at),
    KEY idx_sales_closings_deleted_at (deleted_at),
    CONSTRAINT fk_sales_closings_lead
        FOREIGN KEY (lead_id) REFERENCES customer_leads(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_sales
        FOREIGN KEY (sales_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_supervisor
        FOREIGN KEY (supervisor_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_package
        FOREIGN KEY (package_id) REFERENCES subscription_packages(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_plan
        FOREIGN KEY (plan_id) REFERENCES subscription_plans(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_promotion
        FOREIGN KEY (promotion_id) REFERENCES promotions(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_sales_closings_updated_by
        FOREIGN KEY (updated_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_sales_closings_status
        CHECK (status IN ('PENDING_RECONCILIATION', 'CONFIRMED', 'REJECTED')),
    CONSTRAINT chk_sales_closings_non_negative
        CHECK (base_price >= 0 AND discount_amount >= 0 AND additional_charge >= 0 AND final_amount >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS sales_closings;