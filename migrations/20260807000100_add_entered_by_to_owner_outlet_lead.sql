-- +goose Up
ALTER TABLE owners
    ADD COLUMN entered_by_user_id BIGINT UNSIGNED NULL AFTER status,
    ADD KEY idx_owners_entered_by (entered_by_user_id),
    ADD CONSTRAINT fk_owners_entered_by
        FOREIGN KEY (entered_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL;

ALTER TABLE outlets
    ADD COLUMN entered_by_user_id BIGINT UNSIGNED NULL AFTER status,
    ADD KEY idx_outlets_entered_by (entered_by_user_id),
    ADD CONSTRAINT fk_outlets_entered_by
        FOREIGN KEY (entered_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL;

ALTER TABLE customer_leads
    ADD COLUMN entered_by_user_id BIGINT UNSIGNED NULL AFTER active_sales_id,
    ADD KEY idx_customer_leads_entered_by (entered_by_user_id),
    ADD CONSTRAINT fk_customer_leads_entered_by
        FOREIGN KEY (entered_by_user_id) REFERENCES users(id)
        ON DELETE SET NULL;

-- +goose Down
ALTER TABLE customer_leads
    DROP FOREIGN KEY fk_customer_leads_entered_by,
    DROP KEY idx_customer_leads_entered_by,
    DROP COLUMN entered_by_user_id;

ALTER TABLE outlets
    DROP FOREIGN KEY fk_outlets_entered_by,
    DROP KEY idx_outlets_entered_by,
    DROP COLUMN entered_by_user_id;

ALTER TABLE owners
    DROP FOREIGN KEY fk_owners_entered_by,
    DROP KEY idx_owners_entered_by,
    DROP COLUMN entered_by_user_id;
