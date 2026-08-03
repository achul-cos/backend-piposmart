-- +goose Up
ALTER TABLE subscription_orders
    ADD COLUMN order_type VARCHAR(40) NOT NULL DEFAULT 'NEW' AFTER currency,
    ADD COLUMN source_subscription_id BIGINT UNSIGNED NULL AFTER status,
    ADD COLUMN upgrade_effective_start_date DATE NULL AFTER source_subscription_id,
    ADD COLUMN upgrade_original_end_date DATE NULL AFTER upgrade_effective_start_date,
    ADD COLUMN upgrade_remaining_days INT NULL AFTER upgrade_original_end_date,
    ADD COLUMN upgrade_daily_price DECIMAL(18,2) NULL AFTER upgrade_remaining_days,
    ADD COLUMN previous_package_snapshot_json JSON NULL AFTER upgrade_daily_price,
    ADD COLUMN previous_plan_snapshot_json JSON NULL AFTER previous_package_snapshot_json,
    ADD KEY idx_subscription_orders_order_type (order_type),
    ADD KEY idx_subscription_orders_source_subscription (source_subscription_id),
    ADD CONSTRAINT fk_subscription_orders_source_subscription
        FOREIGN KEY (source_subscription_id) REFERENCES subscriptions(id)
        ON DELETE SET NULL,
    ADD CONSTRAINT chk_subscription_orders_type
        CHECK (order_type IN ('NEW', 'UPGRADE'));

-- +goose Down
ALTER TABLE subscription_orders
    DROP CONSTRAINT chk_subscription_orders_type,
    DROP FOREIGN KEY fk_subscription_orders_source_subscription,
    DROP KEY idx_subscription_orders_order_type,
    DROP KEY idx_subscription_orders_source_subscription,
    DROP COLUMN previous_plan_snapshot_json,
    DROP COLUMN previous_package_snapshot_json,
    DROP COLUMN upgrade_daily_price,
    DROP COLUMN upgrade_remaining_days,
    DROP COLUMN upgrade_original_end_date,
    DROP COLUMN upgrade_effective_start_date,
    DROP COLUMN source_subscription_id,
    DROP COLUMN order_type;
