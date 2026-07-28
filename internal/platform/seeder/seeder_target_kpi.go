package seeder

import (
	"context"
	"database/sql"
	"fmt"

	"backend_crm_piposmart/internal/kpi"
	"backend_crm_piposmart/internal/platform/factory"
)

// seedTargetKpiRankingScenario demonstrates Sprint 13 (Sales Target, KPI, Ranking) with 3
// dedicated Sales reps whose CONFIRMED closing counts for options.AsOf's calendar month are
// engineered to land in each of the three classification tiers against a target of 5:
//   - Sales A: 5 closings (100% of target)  -> ACHIEVED
//   - Sales B: 4 closings (80% of target)   -> NEAR_ACHIEVED (== threshold_near)
//   - Sales C: 1 closing  (20% of target)   -> NOT_ACHIEVED
//
// Recompute is called directly (not through the job_queue) so the demo data is materialized
// synchronously as part of `seed demo` — the seeder has no running worker to process a job.
func seedTargetKpiRankingScenario(ctx context.Context, tx *sql.Tx, options Options) error {
	fake := factory.New(options.Seed, options.AsOf)

	scenarios := []struct {
		userIndex      int
		closingCount   int
		ownerIndexBase int
	}{
		{userIndex: 91, closingCount: 5, ownerIndexBase: 900},
		{userIndex: 92, closingCount: 4, ownerIndexBase: 910},
		{userIndex: 93, closingCount: 1, ownerIndexBase: 920},
	}

	var salesIDs []int64
	for _, scenario := range scenarios {
		salesUser := fake.BuildUser("SALES", scenario.userIndex)
		salesID, err := fake.CreateUser(ctx, tx, salesUser)
		if err != nil {
			return fmt.Errorf("seed target/kpi sales user: %w", err)
		}
		salesIDs = append(salesIDs, salesID)

		for i := 1; i <= scenario.closingCount; i++ {
			ownerIndex := scenario.ownerIndexBase + i
			owner := fake.BuildOwner(ownerIndex)
			ownerID, err := fake.CreateOwner(ctx, tx, owner)
			if err != nil {
				return fmt.Errorf("seed target/kpi owner: %w", err)
			}
			outlet := fake.BuildOutlet(owner.Code, 1, owner)
			outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
			if err != nil {
				return fmt.Errorf("seed target/kpi outlet: %w", err)
			}
			lead := fake.BuildLead(owner.Code, 1, salesUser.Email)
			leadID, err := fake.CreateLead(ctx, tx, ownerID, outletID, lead)
			if err != nil {
				return fmt.Errorf("seed target/kpi lead: %w", err)
			}
			// Small day offsets (1..5) so every closing stays within options.AsOf's month
			// regardless of which day of the month AsOf falls on.
			closing := fake.BuildClosing(i, "BASIC_01_MONTHS", "", "CONFIRMED")
			if _, err := fake.CreateClosing(ctx, tx, leadID, closing); err != nil {
				return fmt.Errorf("seed target/kpi closing: %w", err)
			}
		}
	}

	var metricCodeID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM metric_codes WHERE code = 'CONFIRMED_CLOSING_COUNT'`).Scan(&metricCodeID); err != nil {
		return fmt.Errorf("seed target/kpi: lookup CONFIRMED_CLOSING_COUNT: %w", err)
	}

	periodYear := options.AsOf.Year()
	periodMonth := int(options.AsOf.Month())

	for _, salesID := range salesIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sales_targets (sales_id, metric_code_id, period_year, period_month, target_value, source)
			VALUES (?, ?, ?, ?, '5.00', 'BULK')
			ON DUPLICATE KEY UPDATE
				target_value = VALUES(target_value),
				source = VALUES(source),
				updated_at = NOW()`,
			salesID, metricCodeID, periodYear, periodMonth,
		); err != nil {
			return fmt.Errorf("seed target/kpi: insert sales_target: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO kpi_definitions (metric_code_id, period_year, period_month, weight, threshold_achieved, threshold_near)
		VALUES (?, ?, ?, '100.00', '100.00', '80.00')
		ON DUPLICATE KEY UPDATE
			weight = VALUES(weight),
			threshold_achieved = VALUES(threshold_achieved),
			threshold_near = VALUES(threshold_near),
			active = TRUE,
			updated_at = NOW()`,
		metricCodeID, periodYear, periodMonth,
	); err != nil {
		return fmt.Errorf("seed target/kpi: insert kpi_definition: %w", err)
	}

	// Recompute doesn't touch Repository.db (it only ever operates on the tx passed in), so a
	// nil-backed Repository is safe here — see internal/kpi/repository.go.
	kpiRepo := kpi.NewRepository(nil)
	if err := kpiRepo.Recompute(ctx, tx, periodYear, periodMonth, nil); err != nil {
		return fmt.Errorf("seed target/kpi: recompute: %w", err)
	}

	return nil
}
