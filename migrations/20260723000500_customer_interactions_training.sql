-- +goose Up
CREATE TABLE customer_interactions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    lead_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    outlet_id BIGINT UNSIGNED NULL,
    sales_id BIGINT UNSIGNED NULL,
    supervisor_id BIGINT UNSIGNED NULL,
    interaction_type VARCHAR(30) NOT NULL,
    interaction_at DATETIME(6) NOT NULL,
    contact_name VARCHAR(160) NULL,
    contact_phone VARCHAR(40) NULL,
    duration_seconds INT NULL,
    remark_reason_id BIGINT UNSIGNED NULL,
    remark_score TINYINT UNSIGNED NULL,
    remark_code VARCHAR(60) NULL,
    remark_label VARCHAR(160) NULL,
    note TEXT NULL,
    customer_response TEXT NULL,
    follow_up_at DATETIME(6) NULL,
    follow_up_note TEXT NULL,
    stage_before VARCHAR(40) NULL,
    stage_after VARCHAR(40) NULL,
    status_before VARCHAR(40) NULL,
    status_after VARCHAR(40) NULL,
    score_before TINYINT UNSIGNED NULL,
    score_after TINYINT UNSIGNED NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    KEY idx_customer_interactions_lead_at (lead_id, interaction_at),
    KEY idx_customer_interactions_owner_at (owner_id, interaction_at),
    KEY idx_customer_interactions_sales_at (sales_id, interaction_at),
    KEY idx_customer_interactions_supervisor_at (supervisor_id, interaction_at),
    KEY idx_customer_interactions_type_at (interaction_type, interaction_at),
    KEY idx_customer_interactions_score_at (remark_score, interaction_at),
    KEY idx_customer_interactions_follow_up (follow_up_at),
    KEY idx_customer_interactions_deleted_at (deleted_at),
    CONSTRAINT fk_customer_interactions_lead
        FOREIGN KEY (lead_id) REFERENCES customer_leads(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_customer_interactions_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_customer_interactions_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_customer_interactions_sales
        FOREIGN KEY (sales_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_customer_interactions_supervisor
        FOREIGN KEY (supervisor_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_customer_interactions_remark_reason
        FOREIGN KEY (remark_reason_id) REFERENCES remark_reasons(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_customer_interactions_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE lead_stage_histories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    lead_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    from_stage VARCHAR(40) NULL,
    to_stage VARCHAR(40) NOT NULL,
    from_status VARCHAR(40) NULL,
    to_status VARCHAR(40) NOT NULL,
    from_score TINYINT UNSIGNED NULL,
    to_score TINYINT UNSIGNED NULL,
    changed_by_user_id BIGINT UNSIGNED NULL,
    source_type VARCHAR(40) NOT NULL,
    source_id BIGINT UNSIGNED NULL,
    reason TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_lead_stage_histories_lead_created (lead_id, created_at),
    KEY idx_lead_stage_histories_owner_created (owner_id, created_at),
    KEY idx_lead_stage_histories_changed_by (changed_by_user_id, created_at),
    KEY idx_lead_stage_histories_source (source_type, source_id),
    CONSTRAINT fk_lead_stage_histories_lead
        FOREIGN KEY (lead_id) REFERENCES customer_leads(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_lead_stage_histories_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_lead_stage_histories_changed_by
        FOREIGN KEY (changed_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE training_reports (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    lead_id BIGINT UNSIGNED NULL,
    owner_id BIGINT UNSIGNED NULL,
    outlet_id BIGINT UNSIGNED NULL,
    sales_id BIGINT UNSIGNED NULL,
    supervisor_id BIGINT UNSIGNED NULL,
    training_type VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    scheduled_at DATETIME(6) NOT NULL,
    completed_at DATETIME(6) NULL,
    canceled_at DATETIME(6) NULL,
    rescheduled_at DATETIME(6) NULL,
    location VARCHAR(255) NULL,
    meeting_url VARCHAR(255) NULL,
    trainer_name VARCHAR(160) NULL,
    participant_name VARCHAR(160) NULL,
    note TEXT NULL,
    result_note TEXT NULL,
    cancel_reason TEXT NULL,
    created_by_user_id BIGINT UNSIGNED NULL,
    updated_by_user_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    KEY idx_training_reports_lead_scheduled (lead_id, scheduled_at),
    KEY idx_training_reports_owner_scheduled (owner_id, scheduled_at),
    KEY idx_training_reports_sales_scheduled (sales_id, scheduled_at),
    KEY idx_training_reports_supervisor_scheduled (supervisor_id, scheduled_at),
    KEY idx_training_reports_status_scheduled (status, scheduled_at),
    KEY idx_training_reports_type_scheduled (training_type, scheduled_at),
    KEY idx_training_reports_deleted_at (deleted_at),
    CONSTRAINT fk_training_reports_lead
        FOREIGN KEY (lead_id) REFERENCES customer_leads(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_training_reports_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_training_reports_outlet
        FOREIGN KEY (outlet_id) REFERENCES outlets(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_training_reports_sales
        FOREIGN KEY (sales_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_training_reports_supervisor
        FOREIGN KEY (supervisor_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_training_reports_created_by
        FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_training_reports_updated_by
        FOREIGN KEY (updated_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE customer_leads
    ADD KEY idx_customer_leads_last_interaction (last_interaction_at);

-- +goose Down
ALTER TABLE customer_leads
    DROP INDEX idx_customer_leads_last_interaction;

DROP TABLE IF EXISTS training_reports;
DROP TABLE IF EXISTS lead_stage_histories;
DROP TABLE IF EXISTS customer_interactions;
