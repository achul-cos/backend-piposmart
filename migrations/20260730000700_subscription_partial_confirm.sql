-- +goose Up
-- Sprint 15a §4: PARTIAL_CONFIRM reconciliation outcome + admin-manual orders allowed to exceed
-- owner balance (flagged with a shortfall, not blocked). Both are Admin-only corrections to
-- historical/self-service data already recorded in the real Piposmart app — CreateOrder's
-- balance check still hard-blocks for non-Admin actors (real live debits).
ALTER TABLE subscription_orders
    ADD COLUMN balance_shortfall_amount DECIMAL(18,2) NULL AFTER final_amount;

ALTER TABLE subscription_reconciliations
    DROP CONSTRAINT chk_subscription_reconciliations_status;

ALTER TABLE subscription_reconciliations
    ADD CONSTRAINT chk_subscription_reconciliations_status
        CHECK (status IN ('PENDING', 'CONFIRMED', 'REJECTED', 'PARTIAL_CONFIRM'));

-- admin_final_amount is what actually flows into sales_closings.final_amount (and therefore
-- partner commission calculation) on PARTIAL_CONFIRM — the sales-entered closing amount is
-- overridden to match the admin dashboard's real data. admin_tenure_months is informational only
-- (surfaced to admin so the mismatch is visible), it does not re-resolve plan_id/package_id.
ALTER TABLE subscription_reconciliations
    ADD COLUMN admin_tenure_months INT NULL AFTER amount_difference,
    ADD COLUMN admin_final_amount DECIMAL(18,2) NULL AFTER admin_tenure_months;

-- +goose Down
ALTER TABLE subscription_reconciliations
    DROP COLUMN admin_final_amount,
    DROP COLUMN admin_tenure_months;

ALTER TABLE subscription_reconciliations
    DROP CONSTRAINT chk_subscription_reconciliations_status;

ALTER TABLE subscription_reconciliations
    ADD CONSTRAINT chk_subscription_reconciliations_status
        CHECK (status IN ('PENDING', 'CONFIRMED', 'REJECTED'));

ALTER TABLE subscription_orders
    DROP COLUMN balance_shortfall_amount;
