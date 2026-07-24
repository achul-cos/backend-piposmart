-- Sprint 11b: Large Seeder Data Validation Queries
-- Run after seeding: mysql -u crm -p database < test_queries.sql
-- Expected execution time: 2-3 seconds

-- ============================================================================
-- Query 1: Owner Statistics & Geographic Distribution
-- ============================================================================
-- Purpose: Verify owner count, provinces, and cities coverage
-- Expected: ~18,000 owners across 10+ provinces and 40+ cities

SELECT
  COUNT(*) as total_owners,
  COUNT(DISTINCT province) as unique_provinces,
  COUNT(DISTINCT city) as unique_cities,
  COUNT(CASE WHEN status = 'ACTIVE' THEN 1 END) as active_owners,
  COUNT(CASE WHEN status = 'DELETED' THEN 1 END) as deleted_owners,
  MIN(created_at) as first_owner_date,
  MAX(created_at) as last_owner_date
FROM owners
WHERE deleted_at IS NULL;

-- ============================================================================
-- Query 2: User Distribution by Role
-- ============================================================================
-- Purpose: Verify correct number of supervisors and sales staff
-- Expected: 1 admin, 9 supervisors, 45 sales

SELECT
  r.code as role,
  COUNT(u.id) as user_count,
  GROUP_CONCAT(u.code ORDER BY u.code SEPARATOR ', ') as user_codes
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.deleted_at IS NULL
GROUP BY r.code
ORDER BY user_count DESC;

-- ============================================================================
-- Query 3: Lead Statistics & Stage Distribution
-- ============================================================================
-- Purpose: Verify lead count and stage distribution
-- Expected: ~54,000 leads across NEW, POSSIBLE, POTENTIAL, CLOSING stages

SELECT
  COUNT(*) as total_leads,
  COUNT(DISTINCT owner_id) as unique_owners,
  COUNT(CASE WHEN stage = 'NEW' THEN 1 END) as stage_new,
  COUNT(CASE WHEN stage = 'POSSIBLE' THEN 1 END) as stage_possible,
  COUNT(CASE WHEN stage = 'POTENTIAL' THEN 1 END) as stage_potential,
  COUNT(CASE WHEN stage = 'CLOSING' THEN 1 END) as stage_closing,
  COUNT(CASE WHEN stage = 'INVALID' THEN 1 END) as stage_invalid,
  COUNT(CASE WHEN status = 'OPEN' THEN 1 END) as status_open,
  COUNT(CASE WHEN status = 'INVALID' THEN 1 END) as status_invalid
FROM customer_leads
WHERE deleted_at IS NULL;

-- ============================================================================
-- Query 4: Interaction Statistics & Type Distribution
-- ============================================================================
-- Purpose: Verify interaction count and type distribution
-- Expected: ~378,000 interactions with CALL/CHAT ratio ~1:1

SELECT
  COUNT(*) as total_interactions,
  COUNT(DISTINCT lead_id) as unique_leads,
  COUNT(DISTINCT sales_id) as unique_sales,
  COUNT(CASE WHEN interaction_type = 'CALL' THEN 1 END) as call_count,
  COUNT(CASE WHEN interaction_type = 'CHAT' THEN 1 END) as chat_count,
  ROUND(COUNT(CASE WHEN interaction_type = 'CALL' THEN 1 END) * 100.0 / COUNT(*), 1) as call_percentage,
  MIN(interaction_at) as first_interaction,
  MAX(interaction_at) as last_interaction
FROM customer_interactions
WHERE deleted_at IS NULL;

-- ============================================================================
-- Query 5: Closing Statistics & Status Distribution
-- ============================================================================
-- Purpose: Verify closing count and status distribution
-- Expected: ~6,480 closings with mixed CONFIRMED/PENDING status

SELECT
  COUNT(*) as total_closings,
  COUNT(CASE WHEN status = 'CONFIRMED' THEN 1 END) as confirmed_count,
  COUNT(CASE WHEN status = 'PENDING_RECONCILIATION' THEN 1 END) as pending_count,
  COUNT(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected_count,
  COUNT(DISTINCT lead_id) as unique_leads,
  COUNT(DISTINCT sales_id) as unique_sales,
  ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL), 2) as closing_rate_percent,
  MIN(closed_at) as first_closing,
  MAX(closed_at) as last_closing
FROM sales_closings
WHERE deleted_at IS NULL;

-- ============================================================================
-- Query 6: Temporal Distribution by Year
-- ============================================================================
-- Purpose: Verify growth curve temporal distribution (2020-2026)
-- Expected: Growth curve visible with acceleration mid-period

SELECT
  YEAR(created_at) as year,
  COUNT(*) as owner_count,
  ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM owners WHERE deleted_at IS NULL), 1) as percentage,
  MIN(DATE(created_at)) as first_date,
  MAX(DATE(created_at)) as last_date
FROM owners
WHERE deleted_at IS NULL
GROUP BY YEAR(created_at)
ORDER BY year ASC;

-- ============================================================================
-- Query 7: Temporal Distribution by Month (Last 24 months)
-- ============================================================================
-- Purpose: Verify monthly distribution for recent growth pattern
-- Expected: Relatively steady with occasional spikes

SELECT
  YEAR(created_at) as year,
  MONTH(created_at) as month,
  CONCAT(YEAR(created_at), '-', LPAD(MONTH(created_at), 2, '0')) as year_month,
  COUNT(*) as owner_count
FROM owners
WHERE deleted_at IS NULL AND created_at >= DATE_SUB(NOW(), INTERVAL 24 MONTH)
GROUP BY YEAR(created_at), MONTH(created_at)
ORDER BY year ASC, month ASC;

-- ============================================================================
-- Query 8: Lead Distribution Per Owner
-- ============================================================================
-- Purpose: Verify lead count distribution per owner (expected: 1-6, avg ~3)
-- Expected: Most owners have 2-4 leads

SELECT
  lead_count,
  COUNT(*) as owner_count,
  ROUND(COUNT(*) * 100.0 / (SELECT COUNT(DISTINCT owner_id) FROM customer_leads WHERE deleted_at IS NULL), 1) as percentage
FROM (
  SELECT owner_id, COUNT(*) as lead_count
  FROM customer_leads
  WHERE deleted_at IS NULL
  GROUP BY owner_id
) lead_stats
GROUP BY lead_count
ORDER BY lead_count ASC;

-- ============================================================================
-- Query 9: Interaction Distribution Per Lead
-- ============================================================================
-- Purpose: Verify interaction count distribution per lead (expected: 5-15, avg ~7)
-- Expected: Range 5-15 with concentration around 7-8

SELECT
  interaction_count,
  COUNT(*) as lead_count,
  ROUND(COUNT(*) * 100.0 / (SELECT COUNT(DISTINCT lead_id) FROM customer_interactions WHERE deleted_at IS NULL), 1) as percentage
FROM (
  SELECT lead_id, COUNT(*) as interaction_count
  FROM customer_interactions
  WHERE deleted_at IS NULL
  GROUP BY lead_id
) interaction_stats
GROUP BY interaction_count
ORDER BY interaction_count ASC;

-- ============================================================================
-- Query 10: Sales Team Workload & Performance Metrics
-- ============================================================================
-- Purpose: Verify fair distribution of work across sales team
-- Expected: ~1,000-1,200 leads per sales, proportional interactions & closings

SELECT
  u.id,
  u.code,
  u.name,
  COUNT(DISTINCT cl.id) as assigned_leads,
  COUNT(DISTINCT ci.id) as total_interactions,
  COUNT(DISTINCT sc.id) as total_closings,
  ROUND(COUNT(DISTINCT sc.id) * 100.0 / NULLIF(COUNT(DISTINCT cl.id), 0), 1) as closing_rate_percent,
  ROUND(COUNT(DISTINCT ci.id) * 1.0 / NULLIF(COUNT(DISTINCT cl.id), 0), 1) as interactions_per_lead
FROM users u
LEFT JOIN customer_leads cl ON cl.active_sales_id = u.id AND cl.deleted_at IS NULL
LEFT JOIN customer_interactions ci ON ci.sales_id = u.id AND ci.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.sales_id = u.id AND sc.deleted_at IS NULL
WHERE u.deleted_at IS NULL AND u.code LIKE 'SLS-%'
GROUP BY u.id, u.code, u.name
ORDER BY assigned_leads DESC;

-- ============================================================================
-- Summary Statistics (Bonus Query)
-- ============================================================================
-- Purpose: Quick overall statistics summary
-- Expected: Comprehensive overview of dataset

SELECT
  (SELECT COUNT(*) FROM owners WHERE deleted_at IS NULL) as total_owners,
  (SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL) as total_leads,
  (SELECT COUNT(*) FROM customer_interactions WHERE deleted_at IS NULL) as total_interactions,
  (SELECT COUNT(*) FROM sales_closings WHERE deleted_at IS NULL) as total_closings,
  ROUND((SELECT COUNT(*) FROM sales_closings WHERE deleted_at IS NULL) * 100.0 /
        NULLIF((SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL), 0), 2) as overall_closing_rate,
  ROUND((SELECT COUNT(*) FROM customer_interactions WHERE deleted_at IS NULL) * 1.0 /
        NULLIF((SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL), 0), 2) as avg_interactions_per_lead,
  ROUND((SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL) * 1.0 /
        NULLIF((SELECT COUNT(*) FROM owners WHERE deleted_at IS NULL), 0), 2) as avg_leads_per_owner;

-- ============================================================================
-- Execution Summary
-- ============================================================================
-- All queries above should complete within 2-3 seconds
-- If queries are slow, check:
-- 1. Database indexes on created_at, owner_id, lead_id, sales_id, status
-- 2. Available disk I/O
-- 3. Database buffer pool configuration
--
-- Expected row counts summary:
-- Query 1:  1 row (owner stats)
-- Query 2:  3-4 rows (user roles)
-- Query 3:  1 row (lead stats)
-- Query 4:  1 row (interaction stats)
-- Query 5:  1 row (closing stats)
-- Query 6:  7 rows (years 2020-2026)
-- Query 7:  ~24 rows (months, depending on date range)
-- Query 8:  6 rows (lead count distribution 1-6)
-- Query 9: 11 rows (interaction count distribution 5-15)
-- Query 10: 45 rows (sales team SLS-001 to SLS-045)
-- Bonus:   1 row (summary statistics)
--
-- Total expected rows: ~95-105 rows
