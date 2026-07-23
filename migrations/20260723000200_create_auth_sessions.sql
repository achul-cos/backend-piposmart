-- +goose Up
ALTER TABLE users
    ADD COLUMN password_changed_at DATETIME(6) NULL AFTER must_change_password,
    ADD COLUMN deactivated_at DATETIME(6) NULL AFTER last_login_at;

CREATE TABLE auth_sessions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    refresh_token_hash CHAR(64) NOT NULL,
    refresh_token_family CHAR(36) NOT NULL,
    replaced_by_session_id BIGINT UNSIGNED NULL,
    expires_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6) NULL,
    last_used_at DATETIME(6) NULL,
    ip_address VARCHAR(80) NULL,
    user_agent VARCHAR(255) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_auth_sessions_refresh_hash (refresh_token_hash),
    KEY idx_auth_sessions_user_active (user_id, revoked_at, expires_at),
    KEY idx_auth_sessions_family (refresh_token_family),
    CONSTRAINT fk_auth_sessions_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_auth_sessions_replaced_by
        FOREIGN KEY (replaced_by_session_id) REFERENCES auth_sessions(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS auth_sessions;

ALTER TABLE users
    DROP COLUMN deactivated_at,
    DROP COLUMN password_changed_at;
