-- +goose Up
ALTER TABLE outlets
    DROP FOREIGN KEY fk_outlets_owner;

ALTER TABLE outlets
    MODIFY owner_id BIGINT UNSIGNED NULL;

ALTER TABLE outlets
    ADD CONSTRAINT fk_outlets_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id)
        ON DELETE SET NULL;

-- +goose Down
INSERT INTO owners (code, name, status, deleted_at)
VALUES ('__ORPHAN_OWNER__', 'Placeholder Owner untuk rollback orphan outlet', 'DELETED', CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id);

UPDATE outlets
SET owner_id = LAST_INSERT_ID()
WHERE owner_id IS NULL;

ALTER TABLE outlets
    DROP FOREIGN KEY fk_outlets_owner;

ALTER TABLE outlets
    MODIFY owner_id BIGINT UNSIGNED NOT NULL;

ALTER TABLE outlets
    ADD CONSTRAINT fk_outlets_owner
        FOREIGN KEY (owner_id) REFERENCES owners(id);
