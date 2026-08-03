-- +goose Up
-- Sprint 15a §2: Top Up gets a real PENDING/REJECTED/EXPIRED/ACCEPTED lifecycle. Historically
-- CreateTopup credited the wallet balance synchronously and marked the payment PAID immediately —
-- there was no "waiting for transfer verification" state at all. Now: a topup starts PENDING (no
-- balance/ledger effect), auto-EXPIREs after a 24h session window if untouched, and only credits
-- balance when explicitly ACCEPTED (by admin, or via the Transfer-matching flow in §3).
--
-- All 2018 existing rows are 'PAID' (confirmed via SELECT status, COUNT(*) before writing this
-- migration) — these represent already-completed, already-credited top-ups under the old
-- semantics, so they map straight to ACCEPTED, not to any pending/waiting state.
-- Drop the old CHECK before the data migration below — it only allows PAID/PENDING/FAILED/
-- CANCELED/REFUNDED and would reject the UPDATE to 'ACCEPTED' otherwise.
ALTER TABLE wallet_payments
    DROP CONSTRAINT chk_wallet_payments_status;

UPDATE wallet_payments SET status = 'ACCEPTED' WHERE status = 'PAID';

ALTER TABLE wallet_payments
    ADD CONSTRAINT chk_wallet_payments_status
        CHECK (status IN ('PENDING', 'REJECTED', 'EXPIRED', 'ACCEPTED'));

ALTER TABLE wallet_payments
    ADD COLUMN session_expires_at DATETIME NULL AFTER paid_at,
    ADD COLUMN transfer_date_override DATETIME NULL AFTER session_expires_at,
    ADD COLUMN unique_code VARCHAR(10) NULL AFTER transfer_date_override;

-- +goose Down
ALTER TABLE wallet_payments
    DROP COLUMN unique_code,
    DROP COLUMN transfer_date_override,
    DROP COLUMN session_expires_at;

ALTER TABLE wallet_payments
    DROP CONSTRAINT chk_wallet_payments_status;

UPDATE wallet_payments SET status = 'PAID' WHERE status = 'ACCEPTED';
UPDATE wallet_payments SET status = 'FAILED' WHERE status = 'REJECTED';
UPDATE wallet_payments SET status = 'CANCELED' WHERE status = 'EXPIRED';
-- PENDING top-ups have no equivalent old-schema meaning (the old schema never had a real waiting
-- state — CreateTopup wrote PAID synchronously); map them to PENDING, which the old CHECK also
-- allowed, so a rollback doesn't fail on any top-up still PENDING at rollback time.

ALTER TABLE wallet_payments
    ADD CONSTRAINT chk_wallet_payments_status
        CHECK (status IN ('PAID', 'PENDING', 'FAILED', 'CANCELED', 'REFUNDED'));
