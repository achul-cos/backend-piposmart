-- +goose Up
-- Sprint 15a §3: Transfer — historical bank-transfer proof from an owner to the company, tracked
-- separately from wallet_payments (top-up). Created early (ahead of the full Transfer module) so
-- the new Owner overview rollup (§1, total_transferred) has a real table to sum instead of a stub.
--
-- source/external_reference are generic on purpose: the admin-dashboard integration path (API vs
-- direct DB vs a scraper helper project) is still an open architecture decision, not chosen yet —
-- this schema must not need a second migration once that's picked.
CREATE TABLE owner_transfers (
    id                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    owner_id                 BIGINT UNSIGNED NOT NULL,
    amount                   DECIMAL(18,2) NOT NULL,
    transfer_date            DATETIME NOT NULL,
    proof_url                VARCHAR(500) NULL,
    note                     VARCHAR(500) NULL,
    matched_wallet_payment_id BIGINT UNSIGNED NULL,
    match_status             VARCHAR(20) NOT NULL DEFAULT 'UNMATCHED',
    source                   VARCHAR(20) NOT NULL DEFAULT 'MANUAL_ENTRY',
    external_reference       VARCHAR(255) NULL,
    created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at               DATETIME NULL,
    PRIMARY KEY (id),
    KEY idx_owner_transfers_owner (owner_id),
    KEY idx_owner_transfers_match_status (match_status),
    KEY idx_owner_transfers_matched_payment (matched_wallet_payment_id),
    CONSTRAINT fk_owner_transfers_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id),
    CONSTRAINT fk_owner_transfers_wallet_payment
        FOREIGN KEY (matched_wallet_payment_id) REFERENCES wallet_payments(id) ON DELETE SET NULL,
    CONSTRAINT chk_owner_transfers_match_status
        CHECK (match_status IN ('UNMATCHED', 'SUGGESTED', 'MATCHED', 'REJECTED_MATCH')),
    CONSTRAINT chk_owner_transfers_source
        CHECK (source IN ('MANUAL_ENTRY', 'ADMIN_DASHBOARD'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE IF EXISTS owner_transfers;
