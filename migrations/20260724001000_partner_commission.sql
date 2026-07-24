-- +goose Up
ALTER TABLE partners
    ADD COLUMN commission_rate_percent DECIMAL(5,2) NOT NULL DEFAULT 0.00 AFTER status;

CREATE TABLE partner_commissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(60) NOT NULL,
    partner_id BIGINT UNSIGNED NOT NULL,
    referral_id BIGINT UNSIGNED NOT NULL,
    closing_id BIGINT UNSIGNED NOT NULL,
    commission_rate_percent DECIMAL(5,2) NOT NULL,
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
    CONSTRAINT chk_partner_commissions_non_negative
        CHECK (base_amount >= 0 AND commission_amount >= 0 AND commission_rate_percent >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS partner_commissions;
ALTER TABLE partners DROP COLUMN commission_rate_percent;
