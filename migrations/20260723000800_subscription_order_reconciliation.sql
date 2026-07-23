-- +goose Up
CREATE TABLE subscription_orders (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    owner_id BIGINT UNSIGNED NULL,
    outlet_id BIGINT UNSIGNED NULL,
    closing_id BIGINT UNSIGNED NULL,
    sales_id BIGINT UNSIGNED NULL,
    supervisor_id BIGINT UNSIGNED NULL,
    wallet_account_id BIGINT UNSIGNED NULL,
    wallet_transaction_id BIGINT UNSIGNED NULL,
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
    final_amount DECIMAL(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(40) NOT NULL DEFAULT 'PAID',
    idempotency_key VARCHAR(160) NOT NULL,
    external_reference VARCHAR(120) NULL,
    purchased_at DATETIME(6) NOT NULL,
    subscription_start_date DATE NOT NULL,
    note TEXT NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    updated_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_subscription_orders_code (code),
    UNIQUE KEY uq_subscription_orders_idempotency (idempotency_key),
    UNIQUE KEY uq_subscription_orders_external_reference (external_reference),
    KEY idx_subscription_orders_owner_purchased (owner_id, purchased_at),
    KEY idx_subscription_orders_closing (closing_id),
    KEY idx_subscription_orders_sales_purchased (sales_id, purchased_at),
    KEY idx_subscription_orders_status_purchased (status, purchased_at),
    KEY idx_subscription_orders_wallet_transaction (wallet_transaction_id),
    KEY idx_subscription_orders_deleted_at (deleted_at),
    CONSTRAINT fk_subscription_orders_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_closing
        FOREIGN KEY (closing_id) REFERENCES sales_closings(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_sales
        FOREIGN KEY (sales_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_supervisor
        FOREIGN KEY (supervisor_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_wallet_account
        FOREIGN KEY (wallet_account_id) REFERENCES wallet_accounts(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_wallet_transaction
        FOREIGN KEY (wallet_transaction_id) REFERENCES wallet_transactions(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_package
        FOREIGN KEY (package_id) REFERENCES subscription_packages(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_plan
        FOREIGN KEY (plan_id) REFERENCES subscription_plans(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_promotion
        FOREIGN KEY (promotion_id) REFERENCES promotions(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_orders_updated_by
        FOREIGN KEY (updated_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_subscription_orders_status
        CHECK (status IN ('PAID', 'RECONCILED', 'REJECTED', 'CANCELED')),
    CONSTRAINT chk_subscription_orders_amount_non_negative
        CHECK (base_price >= 0 AND discount_amount >= 0 AND additional_charge >= 0 AND final_amount >= 0),
    CONSTRAINT chk_subscription_orders_duration_positive
        CHECK (tenure_months > 0 AND duration_days > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE subscriptions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    owner_id BIGINT UNSIGNED NULL,
    outlet_id BIGINT UNSIGNED NULL,
    order_id BIGINT UNSIGNED NULL,
    package_id BIGINT UNSIGNED NULL,
    plan_id BIGINT UNSIGNED NULL,
    status VARCHAR(40) NOT NULL DEFAULT 'ACTIVE',
    active_from DATE NOT NULL,
    active_until DATE NOT NULL,
    total_duration_days INT NOT NULL,
    source_type VARCHAR(40) NOT NULL DEFAULT 'SUBSCRIPTION_ORDER',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_subscriptions_code (code),
    UNIQUE KEY uq_subscriptions_order (order_id),
    KEY idx_subscriptions_owner_status (owner_id, status),
    KEY idx_subscriptions_active_until (active_until),
    KEY idx_subscriptions_deleted_at (deleted_at),
    CONSTRAINT fk_subscriptions_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscriptions_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscriptions_order
        FOREIGN KEY (order_id) REFERENCES subscription_orders(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscriptions_package
        FOREIGN KEY (package_id) REFERENCES subscription_packages(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscriptions_plan
        FOREIGN KEY (plan_id) REFERENCES subscription_plans(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_subscriptions_status
        CHECK (status IN ('ACTIVE', 'EXPIRED', 'CANCELED')),
    CONSTRAINT chk_subscriptions_duration_positive
        CHECK (total_duration_days > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE subscription_periods (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    subscription_id BIGINT UNSIGNED NULL,
    order_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    period_index INT NOT NULL DEFAULT 1,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    duration_days INT NOT NULL,
    package_id BIGINT UNSIGNED NULL,
    plan_id BIGINT UNSIGNED NULL,
    promotion_id BIGINT UNSIGNED NULL,
    status VARCHAR(40) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_subscription_periods_subscription_index (subscription_id, period_index),
    KEY idx_subscription_periods_owner_dates (owner_id, start_date, end_date),
    KEY idx_subscription_periods_order (order_id),
    KEY idx_subscription_periods_deleted_at (deleted_at),
    CONSTRAINT fk_subscription_periods_subscription
        FOREIGN KEY (subscription_id) REFERENCES subscriptions(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_periods_order
        FOREIGN KEY (order_id) REFERENCES subscription_orders(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_periods_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_periods_package
        FOREIGN KEY (package_id) REFERENCES subscription_packages(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_periods_plan
        FOREIGN KEY (plan_id) REFERENCES subscription_plans(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_periods_promotion
        FOREIGN KEY (promotion_id) REFERENCES promotions(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_subscription_periods_status
        CHECK (status IN ('ACTIVE', 'EXPIRED', 'CANCELED')),
    CONSTRAINT chk_subscription_periods_duration_positive
        CHECK (duration_days > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE subscription_reconciliations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    order_id BIGINT UNSIGNED NULL,
    closing_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    status VARCHAR(40) NOT NULL DEFAULT 'PENDING',
    match_type VARCHAR(40) NOT NULL DEFAULT 'AUTO',
    issue_code VARCHAR(80) NULL,
    amount_difference DECIMAL(18,2) NOT NULL DEFAULT 0,
    note TEXT NULL,
    reason TEXT NULL,
    confirmed_at DATETIME(6) NULL,
    rejected_at DATETIME(6) NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    updated_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_subscription_reconciliations_code (code),
    UNIQUE KEY uq_subscription_reconciliations_order (order_id),
    UNIQUE KEY uq_subscription_reconciliations_closing (closing_id),
    KEY idx_subscription_reconciliations_owner_status (owner_id, status),
    KEY idx_subscription_reconciliations_deleted_at (deleted_at),
    CONSTRAINT fk_subscription_reconciliations_order
        FOREIGN KEY (order_id) REFERENCES subscription_orders(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_reconciliations_closing
        FOREIGN KEY (closing_id) REFERENCES sales_closings(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_reconciliations_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_reconciliations_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_subscription_reconciliations_updated_by
        FOREIGN KEY (updated_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_subscription_reconciliations_status
        CHECK (status IN ('PENDING', 'CONFIRMED', 'REJECTED')),
    CONSTRAINT chk_subscription_reconciliations_match_type
        CHECK (match_type IN ('AUTO', 'MANUAL'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE reconciliation_issues (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    order_id BIGINT UNSIGNED NULL,
    closing_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    issue_type VARCHAR(80) NOT NULL,
    status VARCHAR(40) NOT NULL DEFAULT 'OPEN',
    description TEXT NULL,
    detected_at DATETIME(6) NOT NULL,
    resolved_at DATETIME(6) NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    updated_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_reconciliation_issues_code (code),
    KEY idx_reconciliation_issues_status_type (status, issue_type),
    KEY idx_reconciliation_issues_order (order_id),
    KEY idx_reconciliation_issues_closing (closing_id),
    KEY idx_reconciliation_issues_owner (owner_id),
    KEY idx_reconciliation_issues_deleted_at (deleted_at),
    CONSTRAINT fk_reconciliation_issues_order
        FOREIGN KEY (order_id) REFERENCES subscription_orders(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_reconciliation_issues_closing
        FOREIGN KEY (closing_id) REFERENCES sales_closings(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_reconciliation_issues_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_reconciliation_issues_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_reconciliation_issues_updated_by
        FOREIGN KEY (updated_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_reconciliation_issues_status
        CHECK (status IN ('OPEN', 'RESOLVED', 'DISMISSED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS reconciliation_issues;
DROP TABLE IF EXISTS subscription_reconciliations;
DROP TABLE IF EXISTS subscription_periods;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS subscription_orders;