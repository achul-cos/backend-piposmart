-- +goose Up
ALTER TABLE customer_leads
    ADD COLUMN current_owner_user_id BIGINT UNSIGNED NULL AFTER active_sales_id,
    ADD COLUMN current_owner_role VARCHAR(30) NOT NULL DEFAULT 'ADMIN' AFTER current_owner_user_id,
    ADD COLUMN supervisor_id BIGINT UNSIGNED NULL AFTER current_owner_role,
    ADD COLUMN current_score TINYINT UNSIGNED NULL AFTER stage,
    ADD COLUMN invalidated_at DATETIME(6) NULL AFTER next_follow_up_at,
    ADD COLUMN invalidated_by_sales_id BIGINT UNSIGNED NULL AFTER invalidated_at,
    ADD UNIQUE KEY uq_customer_leads_owner_id (owner_id),
    ADD KEY idx_customer_leads_current_owner (current_owner_role, current_owner_user_id),
    ADD KEY idx_customer_leads_supervisor (supervisor_id),
    ADD KEY idx_customer_leads_score (current_score),
    ADD CONSTRAINT fk_customer_leads_current_owner
        FOREIGN KEY (current_owner_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    ADD CONSTRAINT fk_customer_leads_supervisor
        FOREIGN KEY (supervisor_id) REFERENCES users(id)
        ON DELETE SET NULL,
    ADD CONSTRAINT fk_customer_leads_invalidated_by_sales
        FOREIGN KEY (invalidated_by_sales_id) REFERENCES users(id)
        ON DELETE SET NULL;

UPDATE customer_leads cl
LEFT JOIN users s ON s.id = cl.active_sales_id
LEFT JOIN roles sr ON sr.id = s.role_id
SET
    cl.current_owner_user_id = cl.active_sales_id,
    cl.current_owner_role = 'SALES',
    cl.supervisor_id = (
        SELECT su.id
        FROM users su
        JOIN roles sur ON sur.id = su.role_id
        WHERE sur.code = 'SUPERVISOR' AND su.deleted_at IS NULL
        ORDER BY su.id
        LIMIT 1
    ),
    cl.current_score = CASE
        WHEN cl.stage = 'INVALID' THEN 0
        WHEN cl.stage = 'POSSIBLE' THEN 1
        WHEN cl.stage = 'POTENTIAL' THEN 2
        WHEN cl.stage = 'CLOSING' THEN 3
        ELSE 1
    END
WHERE cl.active_sales_id IS NOT NULL AND sr.code = 'SALES';

UPDATE customer_leads cl
SET
    cl.current_owner_user_id = (
        SELECT au.id
        FROM users au
        JOIN roles ar ON ar.id = au.role_id
        WHERE ar.code = 'ADMIN' AND au.deleted_at IS NULL
        ORDER BY au.id
        LIMIT 1
    ),
    cl.current_owner_role = 'ADMIN',
    cl.current_score = COALESCE(cl.current_score, 1)
WHERE cl.current_owner_user_id IS NULL;

INSERT INTO customer_leads
    (code, owner_id, source_type, source_reference, stage, status, current_score, current_owner_user_id, current_owner_role)
SELECT
    CONCAT('LEAD-', LPAD(o.id, 6, '0')) AS code,
    o.id AS owner_id,
    'MIGRATION' AS source_type,
    o.code AS source_reference,
    'NEW' AS stage,
    'OPEN' AS status,
    1 AS current_score,
    (
        SELECT au.id
        FROM users au
        JOIN roles ar ON ar.id = au.role_id
        WHERE ar.code = 'ADMIN' AND au.deleted_at IS NULL
        ORDER BY au.id
        LIMIT 1
    ) AS current_owner_user_id,
    'ADMIN' AS current_owner_role
FROM owners o
LEFT JOIN customer_leads cl ON cl.owner_id = o.id
WHERE o.deleted_at IS NULL AND cl.id IS NULL;

CREATE TABLE lead_assignments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    lead_id BIGINT UNSIGNED NOT NULL,
    owner_id BIGINT UNSIGNED NULL,
    from_user_id BIGINT UNSIGNED NULL,
    from_role VARCHAR(30) NULL,
    to_user_id BIGINT UNSIGNED NULL,
    to_role VARCHAR(30) NOT NULL,
    supervisor_id BIGINT UNSIGNED NULL,
    assigned_by_user_id BIGINT UNSIGNED NULL,
    action VARCHAR(60) NOT NULL,
    reason TEXT NULL,
    score TINYINT UNSIGNED NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    active_lead_key BIGINT UNSIGNED GENERATED ALWAYS AS (
        CASE WHEN active THEN lead_id ELSE NULL END
    ) STORED,
    started_at DATETIME(6) NOT NULL,
    ended_at DATETIME(6) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY uq_lead_assignments_one_active (active_lead_key),
    KEY idx_lead_assignments_lead_created (lead_id, created_at),
    KEY idx_lead_assignments_owner (owner_id),
    KEY idx_lead_assignments_to_user_active (to_user_id, active),
    KEY idx_lead_assignments_supervisor_active (supervisor_id, active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO lead_assignments
    (lead_id, owner_id, to_user_id, to_role, supervisor_id, assigned_by_user_id, action, score, active, started_at)
SELECT
    cl.id,
    cl.owner_id,
    cl.current_owner_user_id,
    cl.current_owner_role,
    cl.supervisor_id,
    cl.current_owner_user_id,
    'MIGRATED',
    cl.current_score,
    TRUE,
    COALESCE(cl.created_at, CURRENT_TIMESTAMP)
FROM customer_leads cl
WHERE cl.deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS lead_assignments;

ALTER TABLE customer_leads
    DROP FOREIGN KEY fk_customer_leads_invalidated_by_sales,
    DROP FOREIGN KEY fk_customer_leads_supervisor,
    DROP FOREIGN KEY fk_customer_leads_current_owner;

ALTER TABLE customer_leads
    DROP INDEX uq_customer_leads_owner_id,
    DROP INDEX idx_customer_leads_current_owner,
    DROP INDEX idx_customer_leads_supervisor,
    DROP INDEX idx_customer_leads_score;

ALTER TABLE customer_leads
    DROP COLUMN invalidated_by_sales_id,
    DROP COLUMN invalidated_at,
    DROP COLUMN current_score,
    DROP COLUMN supervisor_id,
    DROP COLUMN current_owner_role,
    DROP COLUMN current_owner_user_id;
