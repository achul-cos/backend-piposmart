package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Closing struct {
	PlanCode      string
	PromotionCode string
	// Sprint 15a §4b — PromotionCodes lets a demo closing stack more than one promotion for the
	// same plan (sales_closing_promotions). When set, it takes precedence over the singular
	// PromotionCode; the first entry still populates the legacy promotion_id/promotion_snapshot_json
	// columns, matching production's insertClosing behavior.
	PromotionCodes     []string
	DiscountAmount     string
	UniqueTransferCode int
	Status             string
	Note               string
	ClosedAt           time.Time
}

func (f *Factory) BuildClosing(index int, planCode, promotionCode, status string) Closing {
	return Closing{
		PlanCode:           planCode,
		PromotionCode:      promotionCode,
		DiscountAmount:     "0.00",
		UniqueTransferCode: 100 + index,
		Status:             status,
		Note:               fmt.Sprintf("Demo Sprint 08 closing %02d", index),
		ClosedAt:           f.asOf.AddDate(0, 0, index).Add(15 * time.Hour),
	}
}

func (f *Factory) CreateClosing(ctx context.Context, tx *sql.Tx, leadID int64, closing Closing) (int64, error) {
	state, err := lookupLeadSeedState(ctx, tx, leadID)
	if err != nil {
		return 0, err
	}
	plan, pkg, err := lookupPlanSnapshot(ctx, tx, closing.PlanCode)
	if err != nil {
		return 0, err
	}
	promotionCodes := closing.PromotionCodes
	if len(promotionCodes) == 0 && strings.TrimSpace(closing.PromotionCode) != "" {
		promotionCodes = []string{closing.PromotionCode}
	}
	additionalCharge := "0.00"
	var promotionID sql.NullInt64
	var promotionSnapshot sql.NullString
	var promotionSnapshots []promotionSeedSnapshot
	if len(promotionCodes) > 0 {
		additionalCents := int64(0)
		for _, promotionCode := range promotionCodes {
			promo, err := lookupPromotionSnapshot(ctx, tx, promotionCode)
			if err != nil {
				return 0, err
			}
			cents, err := parseSeedMoneyToCents(promo.AdditionalCharge)
			if err != nil {
				return 0, err
			}
			additionalCents += cents
			promotionSnapshots = append(promotionSnapshots, promo)
		}
		additionalCharge = formatSeedCents(additionalCents)
		// The legacy singular promotion_id/promotion_snapshot_json columns keep only the FIRST
		// promotion, matching production's insertClosing — sales_closing_promotions (inserted
		// below, once closingID exists) is the authoritative full list.
		first := promotionSnapshots[0]
		promotionID = sql.NullInt64{Int64: first.ID, Valid: true}
		bytes, err := json.Marshal(first)
		if err != nil {
			return 0, err
		}
		promotionSnapshot = sql.NullString{String: string(bytes), Valid: true}
	}
	baseCents, err := parseSeedMoneyToCents(plan.Price)
	if err != nil {
		return 0, err
	}
	discountCents, err := parseSeedMoneyToCents(closing.DiscountAmount)
	if err != nil {
		return 0, err
	}
	additionalCents, err := parseSeedMoneyToCents(additionalCharge)
	if err != nil {
		return 0, err
	}
	finalCents := baseCents - discountCents + additionalCents + int64(closing.UniqueTransferCode*100)
	if finalCents < 0 {
		return 0, fmt.Errorf("seed closing final amount negatif lead=%d", leadID)
	}
	packageBytes, err := json.Marshal(pkg)
	if err != nil {
		return 0, err
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return 0, err
	}
	status := strings.ToUpper(strings.TrimSpace(closing.Status))
	if status == "" {
		status = "PENDING_RECONCILIATION"
	}
	confirmedAt := sql.NullTime{}
	rejectedAt := sql.NullTime{}
	if status == "CONFIRMED" {
		confirmedAt = sql.NullTime{Time: closing.ClosedAt.Add(2 * time.Hour), Valid: true}
	}
	if status == "REJECTED" {
		rejectedAt = sql.NullTime{Time: closing.ClosedAt.Add(2 * time.Hour), Valid: true}
	}
	code := fmt.Sprintf("DEMO-CLS-%06d-%s", leadID, plan.Code)

	if _, err := tx.ExecContext(ctx, "DELETE FROM sales_closings WHERE code = ?", code); err != nil {
		return 0, fmt.Errorf("reset demo closing %s: %w", code, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM customer_interactions WHERE lead_id = ? AND note = ?", leadID, closing.Note); err != nil {
		return 0, fmt.Errorf("reset demo closing interaction lead=%d: %w", leadID, err)
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO sales_closings
			(code, lead_id, owner_id, outlet_id, sales_id, supervisor_id, package_id, plan_id, promotion_id,
			 package_snapshot_json, plan_snapshot_json, promotion_snapshot_json, tenure_months, duration_days,
			 base_price, discount_amount, additional_charge, unique_transfer_code, final_amount, currency,
			 status, note, closed_at, confirmed_at, rejected_at, created_by_user_id, updated_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		code,
		leadID,
		state.ownerID,
		state.outletID,
		state.salesID,
		state.supervisorID,
		pkg.ID,
		plan.ID,
		promotionID,
		string(packageBytes),
		string(planBytes),
		promotionSnapshot,
		plan.TenureMonths,
		plan.DurationDays,
		plan.Price,
		formatSeedCents(discountCents),
		additionalCharge,
		closing.UniqueTransferCode,
		formatSeedCents(finalCents),
		plan.Currency,
		status,
		closing.Note,
		closing.ClosedAt,
		confirmedAt,
		rejectedAt,
		state.salesID,
		state.salesID,
	)
	if err != nil {
		return 0, fmt.Errorf("seed closing %s: %w", code, err)
	}
	closingID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, promo := range promotionSnapshots {
		promoBytes, err := json.Marshal(promo)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sales_closing_promotions (closing_id, promotion_id, promotion_snapshot_json, additional_charge)
			VALUES (?, ?, ?, ?)`,
			closingID, promo.ID, string(promoBytes), promo.AdditionalCharge,
		); err != nil {
			return 0, fmt.Errorf("seed closing promotion %s: %w", code, err)
		}
	}
	reasonID, reasonCode, reasonLabel, err := lookupRemarkReasonByScore(ctx, tx, 3)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO customer_interactions
			(lead_id, owner_id, outlet_id, sales_id, supervisor_id, interaction_type, call_status, chat_status, interaction_at,
			 remark_reason_id, remark_score, remark_code, remark_label, note, customer_response,
			 stage_before, stage_after, status_before, status_after, score_before, score_after, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, 'CALL', 'TERHUBUNG', NULL, ?, ?, 3, ?, ?, ?, ?, ?, 'CLOSING', ?, 'OPEN', ?, 3, ?)`,
		leadID,
		state.ownerID,
		state.outletID,
		state.salesID,
		state.supervisorID,
		closing.ClosedAt,
		reasonID,
		reasonCode,
		reasonLabel,
		closing.Note,
		"Customer setuju membeli paket Piposmart.",
		state.stage,
		state.status,
		state.score,
		state.salesID,
	); err != nil {
		return 0, fmt.Errorf("seed closing interaction %s: %w", code, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO lead_stage_histories
			(lead_id, owner_id, from_stage, to_stage, from_status, to_status, from_score, to_score,
			 changed_by_user_id, source_type, source_id, reason)
		VALUES (?, ?, ?, 'CLOSING', ?, 'OPEN', ?, 3, ?, 'DEMO_CLOSING', ?, ?)`,
		leadID,
		state.ownerID,
		state.stage,
		state.status,
		state.score,
		state.salesID,
		closingID,
		closing.Note,
	); err != nil {
		return 0, fmt.Errorf("seed closing stage history %s: %w", code, err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE customer_leads
		SET stage = 'CLOSING', status = 'OPEN', current_score = 3, last_interaction_at = ?, next_follow_up_at = NULL,
			invalidated_at = NULL, invalidated_by_sales_id = NULL
		WHERE id = ?`, closing.ClosedAt, leadID); err != nil {
		return 0, fmt.Errorf("update seed lead closing lead=%d: %w", leadID, err)
	}
	return closingID, nil
}

type planSeedSnapshot struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	TenureMonths  int    `json:"tenure_months"`
	DurationDays  int    `json:"duration_days"`
	Price         string `json:"price"`
	Currency      string `json:"currency"`
	EffectiveFrom string `json:"effective_from"`
}

type packageSeedSnapshot struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	LevelOrder int    `json:"level_order"`
}

type promotionSeedSnapshot struct {
	ID               int64  `json:"id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	PromotionType    string `json:"promotion_type"`
	ChargeType       string `json:"charge_type"`
	AdditionalCharge string `json:"additional_charge"`
	Priority         int    `json:"priority"`
	EffectiveFrom    string `json:"effective_from"`
}

func lookupPlanSnapshot(ctx context.Context, tx *sql.Tx, planCode string) (planSeedSnapshot, packageSeedSnapshot, error) {
	var plan planSeedSnapshot
	var pkg packageSeedSnapshot
	var effectiveFrom time.Time
	err := tx.QueryRowContext(ctx, `
		SELECT spl.id, spl.code, spl.name, spl.tenure_months, spl.duration_days, CAST(spl.price AS CHAR), spl.currency,
			spl.effective_from, sp.id, sp.code, sp.name, sp.level_order
		FROM subscription_plans spl
		JOIN subscription_packages sp ON sp.id = spl.package_id
		WHERE spl.code = ? AND spl.deleted_at IS NULL AND sp.deleted_at IS NULL
		LIMIT 1`, planCode).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.TenureMonths,
		&plan.DurationDays,
		&plan.Price,
		&plan.Currency,
		&effectiveFrom,
		&pkg.ID,
		&pkg.Code,
		&pkg.Name,
		&pkg.LevelOrder,
	)
	if err != nil {
		return planSeedSnapshot{}, packageSeedSnapshot{}, fmt.Errorf("lookup plan snapshot %s: %w", planCode, err)
	}
	plan.EffectiveFrom = effectiveFrom.Format("2006-01-02")
	return plan, pkg, nil
}

func lookupPromotionSnapshot(ctx context.Context, tx *sql.Tx, promotionCode string) (promotionSeedSnapshot, error) {
	var promo promotionSeedSnapshot
	var effectiveFrom time.Time
	err := tx.QueryRowContext(ctx, `
		SELECT id, code, name, promotion_type, charge_type, CAST(additional_charge AS CHAR), priority, effective_from
		FROM promotions
		WHERE code = ? AND deleted_at IS NULL
		LIMIT 1`, promotionCode).Scan(
		&promo.ID,
		&promo.Code,
		&promo.Name,
		&promo.PromotionType,
		&promo.ChargeType,
		&promo.AdditionalCharge,
		&promo.Priority,
		&effectiveFrom,
	)
	if err != nil {
		return promotionSeedSnapshot{}, fmt.Errorf("lookup promotion snapshot %s: %w", promotionCode, err)
	}
	promo.EffectiveFrom = effectiveFrom.Format("2006-01-02")
	return promo, nil
}

func parseSeedMoneyToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("decimal seed tidak valid: %s", value)
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	cents := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > 2 {
			return 0, fmt.Errorf("decimal seed tidak valid: %s", value)
		}
		if len(fraction) == 1 {
			fraction += "0"
		}
		cents, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return whole*100 + cents, nil
}

func formatSeedCents(value int64) string {
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}
