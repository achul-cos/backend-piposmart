-- +goose Up
-- Sprint 15 continuation: persist historical "Data Bonus Mitra" rows as imported fact data.
-- This is intentionally a snapshot table, not the live partner_commissions ledger: the source
-- workbook contains payout-history/status columns that may predate or differ from the CRM's own
-- referral/commission lifecycle. The snapshot gives us stable imported facts plus DB links to the
-- matched owner/outlet/lead when available.
CREATE TABLE partner_bonus_referral_snapshots (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    import_batch_id BIGINT UNSIGNED NOT NULL,
    source_row_index INT UNSIGNED NOT NULL,

    partner_name VARCHAR(255) NOT NULL,
    partner_owner_code VARCHAR(50) NULL,
    partner_owner_name VARCHAR(255) NULL,
    partner_brand_name VARCHAR(255) NULL,
    partner_category VARCHAR(255) NULL,

    referred_owner_code VARCHAR(50) NOT NULL,
    referred_owner_name VARCHAR(255) NULL,
    referred_project_name VARCHAR(255) NULL,
    referred_outlet_name VARCHAR(255) NULL,

    package_name VARCHAR(120) NULL,
    sales_pic_name VARCHAR(120) NULL,
    top_up_date DATE NULL,
    renewal_date DATE NULL,
    payout_date_1 DATE NULL,
    payout_date_2 DATE NULL,
    subscription_status VARCHAR(120) NULL,
    payout_status VARCHAR(120) NULL,
    cmo_name VARCHAR(120) NULL,

    unpaid_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    stage1_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    stage2_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    paid_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    total_amount DECIMAL(18,2) NOT NULL DEFAULT 0,

    referred_owner_id BIGINT UNSIGNED NULL,
    referred_outlet_id BIGINT UNSIGNED NULL,
    referred_lead_id BIGINT UNSIGNED NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uq_partner_bonus_snapshot_batch_row (import_batch_id, source_row_index),
    KEY idx_partner_bonus_snapshot_referred_owner_code (referred_owner_code),
    KEY idx_partner_bonus_snapshot_payout_status (payout_status),
    KEY idx_partner_bonus_snapshot_renewal_date (renewal_date),

    CONSTRAINT fk_partner_bonus_snapshot_batch
        FOREIGN KEY (import_batch_id) REFERENCES import_batches(id) ON DELETE CASCADE,
    CONSTRAINT fk_partner_bonus_snapshot_owner
        FOREIGN KEY (referred_owner_id) REFERENCES owners(id) ON DELETE SET NULL,
    CONSTRAINT fk_partner_bonus_snapshot_outlet
        FOREIGN KEY (referred_outlet_id) REFERENCES outlets(id) ON DELETE SET NULL,
    CONSTRAINT fk_partner_bonus_snapshot_lead
        FOREIGN KEY (referred_lead_id) REFERENCES customer_leads(id) ON DELETE SET NULL,
    CONSTRAINT chk_partner_bonus_snapshot_amounts_non_negative
        CHECK (
            unpaid_amount >= 0 AND stage1_amount >= 0 AND stage2_amount >= 0
            AND paid_amount >= 0 AND total_amount >= 0
        )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS partner_bonus_referral_snapshots;
