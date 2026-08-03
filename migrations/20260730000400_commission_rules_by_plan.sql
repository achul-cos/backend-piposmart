-- +goose Up
-- Sprint 15a §5: commission is priced per subscription PLAN (package + tenure + price), not per
-- package — two plans of the same package (e.g. Business 12 vs 24 bulan) pay different commission
-- per the mitra MOU (data_admin/Ringkasan_Komisi_Piposmart.pdf). Rescope commission_rules from
-- package_id to plan_id. No production data depends on the old column (commission_rules was an
-- unused overlay before Sprint 15a — see docs/sprint-12/ADDENDUM_roadmap_audit.md), so this is a
-- straight rename + FK retarget rather than a data migration.
ALTER TABLE commission_rules
    DROP FOREIGN KEY fk_commission_rules_package;

ALTER TABLE commission_rules
    CHANGE COLUMN package_id plan_id BIGINT UNSIGNED NULL;

ALTER TABLE commission_rules
    ADD CONSTRAINT fk_commission_rules_plan
        FOREIGN KEY (plan_id) REFERENCES subscription_plans(id);

-- The old idx_commission_rules_package index (now sitting on the renamed plan_id column) is left
-- as-is above to satisfy the FK constraint requirement; rename it here now that both FKs briefly
-- coexist, so the index name matches the column it actually covers.
ALTER TABLE commission_rules
    RENAME INDEX idx_commission_rules_package TO idx_commission_rules_plan;

-- +goose Down
ALTER TABLE commission_rules
    DROP FOREIGN KEY fk_commission_rules_plan;

ALTER TABLE commission_rules
    CHANGE COLUMN plan_id package_id BIGINT UNSIGNED NULL;

ALTER TABLE commission_rules
    ADD CONSTRAINT fk_commission_rules_package
        FOREIGN KEY (package_id) REFERENCES subscription_packages(id);

ALTER TABLE commission_rules
    RENAME INDEX idx_commission_rules_plan TO idx_commission_rules_package;
