-- +goose Up
-- Commission rate itself already lives on partner_types (commission_mode, commission_value,
-- since the Sprint 1 baseline schema). This migration adds the ledger that tracks commission
-- actually earned per confirmed closing, snapshotting the partner_type's mode/value at the
-- time of calculation so later rate changes don't retroactively alter historical commissions.
CREATE TABLE partner_commissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(60) NOT NULL,
    partner_id BIGINT UNSIGNED NOT NULL,
    referral_id BIGINT UNSIGNED NOT NULL,
    closing_id BIGINT UNSIGNED NOT NULL,
    commission_mode VARCHAR(30) NOT NULL,
    commission_value DECIMAL(18,2) NOT NULL,
    base_amount DECIMAL(18,2) NOT NULL,
    commission_amount DECIMAL(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    note TEXT NULL,
    approved_by_user_id BIGINT UNSIGNED NULL,
    approved_at DATETIME NULL,
    paid_by_user_id BIGINT UNSIGNED NULL,
    paid_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_partner_commissions_code (code),
    UNIQUE KEY uq_partner_commissions_closing (closing_id),
    KEY idx_partner_commissions_partner_status (partner_id, status),
    KEY idx_partner_commissions_referral (referral_id),
    CONSTRAINT fk_partner_commissions_partner
        FOREIGN KEY (partner_id) REFERENCES partners(id) ON DELETE CASCADE,
    CONSTRAINT fk_partner_commissions_referral
        FOREIGN KEY (referral_id) REFERENCES partner_referrals(id) ON DELETE CASCADE,
    CONSTRAINT fk_partner_commissions_closing
        FOREIGN KEY (closing_id) REFERENCES sales_closings(id) ON DELETE CASCADE,
    CONSTRAINT fk_partner_commissions_approved_by
        FOREIGN KEY (approved_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_partner_commissions_paid_by
        FOREIGN KEY (paid_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_partner_commissions_status
        CHECK (status IN ('PENDING', 'APPROVED', 'PAID', 'CANCELLED')),
    CONSTRAINT chk_partner_commissions_mode
        CHECK (commission_mode IN ('PERCENTAGE', 'FIXED')),
    CONSTRAINT chk_partner_commissions_non_negative
        CHECK (base_amount >= 0 AND commission_amount >= 0 AND commission_value >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS partner_commissions;
