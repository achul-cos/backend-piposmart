-- +goose Up
-- New & Subscribe archive migration (2023-2026): the source data distinguishes TF/BRI, Midtrans,
-- and "Saldo Aplikasi" (leftover balance reused, no topup this row) payments, and separately
-- records a Midtrans admin fee + settlement amount per transaction. None of that was structurally
-- representable before — payment_channel is free text with no enum, and there was nowhere to put
-- fee/settlement. is_synthetic_backfill marks topups the seeder injects to cover a "Saldo Aplikasi"
-- purchase whose original topup predates the migrated data range, so they stay auditable/distinct
-- from genuine historical transactions.
ALTER TABLE wallet_payments
    ADD COLUMN payment_method VARCHAR(30) NULL AFTER payment_channel,
    ADD CONSTRAINT chk_wallet_payments_payment_method
        CHECK (payment_method IS NULL OR payment_method IN ('TF_BRI', 'MIDTRANS', 'SALDO_APLIKASI')),
    ADD COLUMN fee_amount DECIMAL(18, 2) NULL AFTER amount,
    ADD CONSTRAINT chk_wallet_payments_fee_amount
        CHECK (fee_amount IS NULL OR fee_amount >= 0),
    ADD COLUMN settlement_amount DECIMAL(18, 2) NULL AFTER fee_amount,
    ADD CONSTRAINT chk_wallet_payments_settlement_amount
        CHECK (settlement_amount IS NULL OR settlement_amount >= 0),
    ADD COLUMN is_synthetic_backfill TINYINT(1) NOT NULL DEFAULT 0 AFTER settlement_amount;

-- +goose Down
ALTER TABLE wallet_payments
    DROP CONSTRAINT chk_wallet_payments_payment_method,
    DROP CONSTRAINT chk_wallet_payments_fee_amount,
    DROP CONSTRAINT chk_wallet_payments_settlement_amount,
    DROP COLUMN payment_method,
    DROP COLUMN fee_amount,
    DROP COLUMN settlement_amount,
    DROP COLUMN is_synthetic_backfill;
