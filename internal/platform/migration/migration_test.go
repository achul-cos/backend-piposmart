package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBaselineMigrationContainsSprintTwoTables(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "migrations", "20260723000100_baseline_crm_schema.sql"))
	if err != nil {
		t.Fatalf("read baseline migration: %v", err)
	}

	sql := string(content)
	required := []string{
		"CREATE TABLE roles",
		"CREATE TABLE users",
		"CREATE TABLE seed_runs",
		"CREATE TABLE remark_reasons",
		"CREATE TABLE partner_types",
		"CREATE TABLE metric_codes",
		"CREATE TABLE subscription_packages",
		"CREATE TABLE subscription_plans",
		"CREATE TABLE promotions",
		"CREATE TABLE owners",
		"CREATE TABLE outlets",
		"CREATE TABLE customer_leads",
		"-- +goose Down",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Errorf("baseline migration tidak memiliki %q", item)
		}
	}
}

func TestExpectedSQLMigrationsArePresent(t *testing.T) {
	root := repositoryRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	sqlFiles := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles[entry.Name()] = true
		}
	}

	required := []string{
		"20260723000100_baseline_crm_schema.sql",
		"20260723000200_create_auth_sessions.sql",
		"20260723000300_owner_outlet_orphan_delete_policy.sql",
		"20260723000400_lead_ownership_assignment.sql",
		"20260723000500_customer_interactions_training.sql",
		"20260723000600_sales_closings.sql",
		"20260723000700_wallet_payment_ledger.sql",
		"20260723000800_subscription_order_reconciliation.sql",
	}
	for _, item := range required {
		if !sqlFiles[item] {
			t.Fatalf("migration SQL %q tidak ditemukan; files=%v", item, sqlFiles)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(workingDirectory, "..", "..", ".."))
}
