SET FOREIGN_KEY_CHECKS = 0;

-- 1. Create a temporary table to store the IDs of dummy owners
CREATE TEMPORARY TABLE temp_dummy_owners AS
SELECT id FROM owners 
WHERE is_testing_account = 1 
   OR name LIKE '%test%' 
   OR name LIKE '%dummy%' 
   OR name LIKE '%coba%' 
   OR name LIKE '%demo%';

-- 2. Create a temporary table to store the IDs of dummy outlets
CREATE TEMPORARY TABLE temp_dummy_outlets AS
SELECT id FROM outlets 
WHERE owner_id IN (SELECT id FROM temp_dummy_owners)
   OR name LIKE '%test%' 
   OR name LIKE '%dummy%' 
   OR name LIKE '%coba%' 
   OR name LIKE '%demo%';

-- 3. Delete Wallets & Payments
DELETE FROM wallet_transactions WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM wallet_payments WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM wallet_accounts WHERE owner_id IN (SELECT id FROM temp_dummy_owners);

-- 4. Delete Subscriptions
DELETE FROM subscription_reconciliations WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM subscription_orders WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM subscription_periods WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM subscriptions WHERE owner_id IN (SELECT id FROM temp_dummy_owners);

-- 5. Delete Leads & Interactions
DELETE FROM lead_stage_histories WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM customer_interactions WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM customer_leads WHERE owner_id IN (SELECT id FROM temp_dummy_owners);

-- 6. Delete other relations
DELETE FROM sales_closings WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM training_reports WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM owner_transfers WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM partner_bonus_referral_snapshots WHERE referred_owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM import_rows WHERE owner_id IN (SELECT id FROM temp_dummy_owners);
DELETE FROM reconciliation_issues WHERE owner_id IN (SELECT id FROM temp_dummy_owners);

-- 7. Delete the dummy Outlets
DELETE FROM outlets WHERE id IN (SELECT id FROM temp_dummy_outlets);

-- 8. Delete the dummy Owners
DELETE FROM owners WHERE id IN (SELECT id FROM temp_dummy_owners);

-- 9. Delete dummy Users (Sales)
DELETE FROM users WHERE id IN (12, 13);

-- Clean up
DROP TABLE temp_dummy_owners;
DROP TABLE temp_dummy_outlets;

SET FOREIGN_KEY_CHECKS = 1;
