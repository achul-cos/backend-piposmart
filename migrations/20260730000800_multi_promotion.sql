-- +goose Up
-- Sprint 15a §4b: a sales closing (and the subscription order derived from it) can now stack
-- MULTIPLE promotions for the same plan, not just one — per pitching feedback, as long as each
-- promotion is actually eligible for that plan. The legacy singular promotion_id/
-- promotion_snapshot_json columns on sales_closings/subscription_orders are left untouched
-- (still populated with the FIRST promotion applied, for any old consumer reading that single
-- field) — these join tables are the new authoritative "which promotions applied" list; the
-- aggregate additional_charge/final_amount on the parent row already sums across all of them.
CREATE TABLE sales_closing_promotions (
    id                     BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    closing_id             BIGINT UNSIGNED NOT NULL,
    promotion_id           BIGINT UNSIGNED NOT NULL,
    promotion_snapshot_json JSON NOT NULL,
    additional_charge      DECIMAL(18,2) NOT NULL DEFAULT 0,
    created_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_sales_closing_promotions_closing (closing_id),
    CONSTRAINT fk_sales_closing_promotions_closing
        FOREIGN KEY (closing_id) REFERENCES sales_closings(id) ON DELETE CASCADE,
    CONSTRAINT fk_sales_closing_promotions_promotion
        FOREIGN KEY (promotion_id) REFERENCES promotions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE subscription_order_promotions (
    id                     BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id               BIGINT UNSIGNED NOT NULL,
    promotion_id           BIGINT UNSIGNED NOT NULL,
    promotion_snapshot_json JSON NOT NULL,
    additional_charge      DECIMAL(18,2) NOT NULL DEFAULT 0,
    created_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_subscription_order_promotions_order (order_id),
    CONSTRAINT fk_subscription_order_promotions_order
        FOREIGN KEY (order_id) REFERENCES subscription_orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_subscription_order_promotions_promotion
        FOREIGN KEY (promotion_id) REFERENCES promotions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS subscription_order_promotions;
DROP TABLE IF EXISTS sales_closing_promotions;
