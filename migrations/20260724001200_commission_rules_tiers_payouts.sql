-- +goose Up
-- Closes 3 gaps from docs/sprint-12/ADDENDUM_roadmap_audit.md ("Backlog Resmi"):
-- TIER commission mode, effective-dated/package-scoped commission rules, and Payout/PayoutItem
-- batching. commission_rules is an ADDITIVE overlay: if no row matches a partner_type
-- (+ optional package + date), commission calculation falls back to the existing
-- partner_types.commission_mode/value flat fields exactly as in Sprint 12 — 100% backward
-- compatible, no existing data/tests break.

-- 1. commission_rules — optional, effective-dated, optionally package-scoped rate definition.
CREATE TABLE commission_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    partner_type_id BIGINT UNSIGNED NOT NULL,
    package_id BIGINT UNSIGNED NULL,          -- NULL = applies to all packages of this partner_type
    mode VARCHAR(30) NOT NULL,                 -- PERCENTAGE | FIXED | TIER
    value DECIMAL(18,2) NULL,                  -- required for PERCENTAGE/FIXED, NULL when TIER (rate lives in commission_tiers)
    effective_from DATE NOT NULL,
    effective_to DATE NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_commission_rules_lookup (partner_type_id, active, effective_from, effective_to),
    KEY idx_commission_rules_package (package_id),
    CONSTRAINT fk_commission_rules_partner_type
        FOREIGN KEY (partner_type_id) REFERENCES partner_types(id),
    CONSTRAINT fk_commission_rules_package
        FOREIGN KEY (package_id) REFERENCES subscription_packages(id),
    CONSTRAINT fk_commission_rules_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_commission_rules_mode
        CHECK (mode IN ('PERCENTAGE', 'FIXED', 'TIER')),
    CONSTRAINT chk_commission_rules_value_by_mode
        CHECK (
            (mode = 'TIER' AND value IS NULL)
            OR (mode IN ('PERCENTAGE', 'FIXED') AND value IS NOT NULL AND value >= 0)
        ),
    CONSTRAINT chk_commission_rules_effective_range
        CHECK (effective_to IS NULL OR effective_to >= effective_from)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. commission_tiers — child rows of a TIER-mode commission_rule. min_closings/max_closings
-- bracket the partner's cumulative CONFIRMED-closing count for the calendar month (business
-- decision: volume-based tiering, not per-closing-amount, not lifetime cumulative).
CREATE TABLE commission_tiers (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    commission_rule_id BIGINT UNSIGNED NOT NULL,
    tier_order INT NOT NULL,
    min_closings INT UNSIGNED NOT NULL,
    max_closings INT UNSIGNED NULL,            -- NULL = open-ended (top tier); last tier must be NULL (enforced in service layer)
    mode VARCHAR(30) NOT NULL,                  -- PERCENTAGE | FIXED (no nested TIER)
    value DECIMAL(18,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_commission_tiers_rule_order (commission_rule_id, tier_order),
    KEY idx_commission_tiers_rule_min (commission_rule_id, min_closings),
    CONSTRAINT fk_commission_tiers_rule
        FOREIGN KEY (commission_rule_id) REFERENCES commission_rules(id) ON DELETE CASCADE,
    CONSTRAINT chk_commission_tiers_mode
        CHECK (mode IN ('PERCENTAGE', 'FIXED')),
    CONSTRAINT chk_commission_tiers_value_non_negative
        CHECK (value >= 0),
    CONSTRAINT chk_commission_tiers_range
        CHECK (max_closings IS NULL OR max_closings >= min_closings)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. partner_payouts — one payout per partner, batching multiple APPROVED commissions.
CREATE TABLE partner_payouts (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(60) NOT NULL,
    partner_id BIGINT UNSIGNED NOT NULL,
    total_amount DECIMAL(18,2) NOT NULL DEFAULT 0,   -- snapshot: sum of item amounts at creation time
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',    -- PENDING | PAID | CANCELLED
    note TEXT NULL,
    prepared_by_user_id BIGINT UNSIGNED NULL,
    paid_by_user_id BIGINT UNSIGNED NULL,
    paid_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_partner_payouts_code (code),
    KEY idx_partner_payouts_partner_status (partner_id, status),
    CONSTRAINT fk_partner_payouts_partner
        FOREIGN KEY (partner_id) REFERENCES partners(id) ON DELETE CASCADE,
    CONSTRAINT fk_partner_payouts_prepared_by
        FOREIGN KEY (prepared_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_partner_payouts_paid_by
        FOREIGN KEY (paid_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_partner_payouts_status
        CHECK (status IN ('PENDING', 'PAID', 'CANCELLED')),
    CONSTRAINT chk_partner_payouts_total_non_negative
        CHECK (total_amount >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. partner_payout_items — links a commission into a payout. released_at + the VIRTUAL
-- generated column below are the double-pay guard: a commission can only have ONE *active*
-- (non-released) item row at a time, DB-enforced, while cancelled payouts still keep their
-- (released) item rows for audit history. Mirrors partner_assignments'
-- active_partner_key/uq_partner_assignments_one_active pattern from migration ...001100.
CREATE TABLE partner_payout_items (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    payout_id BIGINT UNSIGNED NOT NULL,
    commission_id BIGINT UNSIGNED NOT NULL,
    amount DECIMAL(18,2) NOT NULL,             -- snapshot of commission_amount at batching time
    released_at DATETIME NULL,                  -- set when the owning payout is CANCELLED
    active_commission_key BIGINT UNSIGNED GENERATED ALWAYS AS (
        CASE WHEN released_at IS NULL THEN commission_id ELSE NULL END
    ) VIRTUAL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_partner_payout_items_active_commission (active_commission_key),
    KEY idx_partner_payout_items_payout (payout_id),
    KEY idx_partner_payout_items_commission (commission_id),
    CONSTRAINT fk_partner_payout_items_payout
        FOREIGN KEY (payout_id) REFERENCES partner_payouts(id) ON DELETE CASCADE,
    CONSTRAINT fk_partner_payout_items_commission
        FOREIGN KEY (commission_id) REFERENCES partner_commissions(id) ON DELETE CASCADE,
    CONSTRAINT chk_partner_payout_items_amount_non_negative
        CHECK (amount >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. Traceability: which rule/tier produced a commission snapshot. Plain nullable columns,
-- NOT generated — the STORED-via-ALTER-TABLE FK gotcha from migration 20260724001100 does not
-- apply here since these are ordinary (non-generated) columns.
ALTER TABLE partner_commissions
    ADD COLUMN commission_rule_id BIGINT UNSIGNED NULL AFTER commission_value,
    ADD COLUMN tier_ordinal INT UNSIGNED NULL AFTER commission_rule_id;

ALTER TABLE partner_commissions
    ADD CONSTRAINT fk_partner_commissions_rule
        FOREIGN KEY (commission_rule_id) REFERENCES commission_rules(id) ON DELETE SET NULL,
    ADD KEY idx_partner_commissions_rule (commission_rule_id);

-- +goose Down
ALTER TABLE partner_commissions
    DROP FOREIGN KEY fk_partner_commissions_rule,
    DROP INDEX idx_partner_commissions_rule,
    DROP COLUMN tier_ordinal,
    DROP COLUMN commission_rule_id;

DROP TABLE IF EXISTS partner_payout_items;
DROP TABLE IF EXISTS partner_payouts;
DROP TABLE IF EXISTS commission_tiers;
DROP TABLE IF EXISTS commission_rules;
