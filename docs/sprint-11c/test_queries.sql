-- Sprint 11c - Validation Queries for `seed demo --preset=large`
-- Run: mysql -u root -p crm_piposmart < test_queries.sql

-- 1. Entity counts
SELECT 'Owners' AS entity, COUNT(*) AS count FROM owners
UNION ALL SELECT 'Outlets', COUNT(*) FROM outlets
UNION ALL SELECT 'Leads', COUNT(*) FROM customer_leads
UNION ALL SELECT 'Interactions', COUNT(*) FROM customer_interactions
UNION ALL SELECT 'Closings', COUNT(*) FROM sales_closings
UNION ALL SELECT 'Sales users', COUNT(*) FROM users u JOIN roles r ON r.id = u.role_id WHERE r.code = 'SALES'
UNION ALL SELECT 'Supervisors', COUNT(*) FROM users u JOIN roles r ON r.id = u.role_id WHERE r.code = 'SUPERVISOR';

-- 2. Temporal distribution (growth curve check)
-- Expected: owner_count meningkat dari 2020 ke 2024-2025, lalu turun di tahun as-of (plateau parsial)
SELECT
  YEAR(created_at) AS year,
  COUNT(*) AS owner_count,
  ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM owners), 1) AS pct
FROM owners
GROUP BY YEAR(created_at)
ORDER BY year;

-- 3. Constraint integrity: setiap owner harus punya tepat 1 lead (uq_customer_leads_owner_id)
-- Expected: 0 baris
SELECT owner_id, COUNT(*) AS lead_count
FROM customer_leads
GROUP BY owner_id
HAVING COUNT(*) > 1;

-- 4. Orphan check: lead tanpa owner valid
-- Expected: 0
SELECT COUNT(*) AS orphan_leads
FROM customer_leads cl
LEFT JOIN owners o ON o.id = cl.owner_id
WHERE o.id IS NULL;

-- 5. Interaction spread per lead
SELECT
  MIN(cnt) AS min_interactions,
  MAX(cnt) AS max_interactions,
  ROUND(AVG(cnt), 1) AS avg_interactions
FROM (
  SELECT lead_id, COUNT(*) AS cnt
  FROM customer_interactions
  GROUP BY lead_id
) x;

-- 6. Closing rate check (target ~12%)
SELECT
  (SELECT COUNT(*) FROM sales_closings) AS total_closings,
  (SELECT COUNT(*) FROM customer_leads) AS total_leads,
  ROUND(100.0 * (SELECT COUNT(*) FROM sales_closings) / (SELECT COUNT(*) FROM customer_leads), 1) AS closing_rate_pct;

-- 7. No future-dated events beyond as-of (clampToAsOf check)
-- Ganti '2026-07-24' dengan nilai --as-of yang dipakai saat seeding
SELECT COUNT(*) AS future_interactions
FROM customer_interactions
WHERE interaction_at > '2026-07-24 23:59:59';

SELECT COUNT(*) AS future_closings
FROM sales_closings
WHERE closed_at > '2026-07-24 23:59:59';

-- 8. Geographic spread
SELECT COUNT(DISTINCT province) AS unique_provinces, COUNT(DISTINCT city) AS unique_cities
FROM owners;
