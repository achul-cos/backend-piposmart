-- +goose Up
-- Sprint 11 DoD requires "satu mitra hanya memiliki satu PIC aktif" (one partner has
-- only one active PIC), analogous to the same rule for customer_leads solved in
-- migration 20260723000400 via a generated column + UNIQUE KEY. partner_assignments
-- was missing this DB-level guard: the service layer only did a check-then-deactivate-
-- then-insert without a transaction/lock, which is racy under concurrent AssignPIC calls.
-- Uses VIRTUAL rather than STORED: adding a STORED generated column via ALTER TABLE
-- on this MySQL 8.0.46 instance raises "Error 1215 Cannot add foreign key constraint"
-- for tables with outgoing FKs (partner_assignments -> partners/users), regardless of
-- ALGORITHM=COPY/INPLACE or foreign_key_checks. VIRTUAL avoids the rebuild path that
-- triggers it, and MySQL supports secondary/unique indexes on VIRTUAL generated
-- columns natively, so the constraint guarantee is identical. This differs from
-- lead_assignments (migration 20260723000400) only because that STORED column was
-- defined at CREATE TABLE time, not added via ALTER TABLE afterward.
ALTER TABLE partner_assignments
    ADD COLUMN active_partner_key BIGINT UNSIGNED GENERATED ALWAYS AS (
        CASE WHEN active THEN partner_id ELSE NULL END
    ) VIRTUAL AFTER active;

ALTER TABLE partner_assignments
    ADD UNIQUE KEY uq_partner_assignments_one_active (active_partner_key);

-- +goose Down
ALTER TABLE partner_assignments
    DROP INDEX uq_partner_assignments_one_active,
    DROP COLUMN active_partner_key;
