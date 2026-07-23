-- +goose Up
CREATE TABLE wallet_accounts (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_id BIGINT UNSIGNED NULL,
    account_code VARCHAR(80) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    balance DECIMAL(18,2) NOT NULL DEFAULT 0,
    status VARCHAR(40) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_wallet_accounts_owner (owner_id),
    UNIQUE KEY uq_wallet_accounts_code (account_code),
    KEY idx_wallet_accounts_status (status),
    KEY idx_wallet_accounts_deleted_at (deleted_at),
    CONSTRAINT fk_wallet_accounts_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_wallet_accounts_balance_non_negative
        CHECK (balance >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE wallet_payments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    owner_id BIGINT UNSIGNED NULL,
    wallet_account_id BIGINT UNSIGNED NULL,
    payment_type VARCHAR(40) NOT NULL,
    payment_channel VARCHAR(40) NOT NULL,
    external_reference VARCHAR(120) NULL,
    idempotency_key VARCHAR(160) NOT NULL,
    amount DECIMAL(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(40) NOT NULL,
    paid_at DATETIME(6) NULL,
    note TEXT NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_wallet_payments_code (code),
    UNIQUE KEY uq_wallet_payments_idempotency (idempotency_key),
    UNIQUE KEY uq_wallet_payments_external_reference (external_reference),
    KEY idx_wallet_payments_owner_paid (owner_id, paid_at),
    KEY idx_wallet_payments_status_paid (status, paid_at),
    KEY idx_wallet_payments_wallet (wallet_account_id),
    KEY idx_wallet_payments_deleted_at (deleted_at),
    CONSTRAINT fk_wallet_payments_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_wallet_payments_wallet
        FOREIGN KEY (wallet_account_id) REFERENCES wallet_accounts(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_wallet_payments_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_wallet_payments_amount_positive
        CHECK (amount > 0),
    CONSTRAINT chk_wallet_payments_type
        CHECK (payment_type IN ('TOPUP')),
    CONSTRAINT chk_wallet_payments_status
        CHECK (status IN ('PAID', 'PENDING', 'FAILED', 'CANCELED', 'REFUNDED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE wallet_transactions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(80) NOT NULL,
    wallet_account_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    payment_id BIGINT UNSIGNED NULL,
    transaction_type VARCHAR(40) NOT NULL,
    direction VARCHAR(10) NOT NULL,
    amount DECIMAL(18,2) NOT NULL,
    balance_before DECIMAL(18,2) NOT NULL,
    balance_after DECIMAL(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    source_type VARCHAR(60) NOT NULL,
    source_reference VARCHAR(160) NULL,
    external_reference VARCHAR(120) NULL,
    idempotency_key VARCHAR(160) NOT NULL,
    occurred_at DATETIME(6) NOT NULL,
    note TEXT NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_wallet_transactions_code (code),
    UNIQUE KEY uq_wallet_transactions_idempotency (idempotency_key),
    KEY idx_wallet_transactions_wallet_occurred (wallet_account_id, occurred_at),
    KEY idx_wallet_transactions_owner_occurred (owner_id, occurred_at),
    KEY idx_wallet_transactions_payment (payment_id),
    KEY idx_wallet_transactions_type_occurred (transaction_type, occurred_at),
    KEY idx_wallet_transactions_deleted_at (deleted_at),
    CONSTRAINT fk_wallet_transactions_wallet
        FOREIGN KEY (wallet_account_id) REFERENCES wallet_accounts(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_wallet_transactions_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_wallet_transactions_payment
        FOREIGN KEY (payment_id) REFERENCES wallet_payments(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_wallet_transactions_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT chk_wallet_transactions_amount_positive
        CHECK (amount > 0),
    CONSTRAINT chk_wallet_transactions_direction
        CHECK (direction IN ('CREDIT', 'DEBIT')),
    CONSTRAINT chk_wallet_transactions_type
        CHECK (transaction_type IN ('CREDIT', 'DEBIT', 'ADJUSTMENT', 'REFUND')),
    CONSTRAINT chk_wallet_transactions_balance_non_negative
        CHECK (balance_before >= 0 AND balance_after >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS wallet_payments;
DROP TABLE IF EXISTS wallet_accounts;