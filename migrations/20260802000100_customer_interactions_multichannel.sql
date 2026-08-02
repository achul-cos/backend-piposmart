-- +goose Up
ALTER TABLE customer_interactions
    ADD COLUMN call_status VARCHAR(60) NULL AFTER interaction_type,
    ADD COLUMN chat_status VARCHAR(60) NULL AFTER call_status;

ALTER TABLE customer_interactions
    ADD KEY idx_customer_interactions_call_status_at (call_status, interaction_at),
    ADD KEY idx_customer_interactions_chat_status_at (chat_status, interaction_at);

-- +goose Down
ALTER TABLE customer_interactions
    DROP INDEX idx_customer_interactions_chat_status_at,
    DROP INDEX idx_customer_interactions_call_status_at,
    DROP COLUMN chat_status,
    DROP COLUMN call_status;
