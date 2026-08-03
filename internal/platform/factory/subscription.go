package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SubscriptionOrder struct {
	PlanCode      string
	PromotionCode string
	// Sprint 15a §4b — PromotionCodes lets a demo order stack more than one promotion for the
	// same plan (subscription_order_promotions), same convention as Closing.PromotionCodes.
	PromotionCodes         []string
	ClosingID              sql.NullInt64
	ExternalReference      string
	IdempotencyKey         string
	PurchasedAt            time.Time
	SubscriptionStartDate  time.Time
	Note                   string
	// Sprint 15a §4 — AllowBalanceShortfall demos an admin-manual order allowed to exceed the
	// owner's balance: the wallet clamps to 0 instead of going negative, and the deficit is
	// recorded on balance_shortfall_amount, per subscription_partial_confirm.sql. Without this
	// flag, a seed scenario landing short on balance still hard-fails as before (a real seed bug).
	AllowBalanceShortfall bool
}

func (f *Factory) BuildSubscriptionOrder(index int, planCode, promotionCode string, closingID sql.NullInt64) SubscriptionOrder {
	purchasedAt := f.asOf.AddDate(0, 0, index).Add(13 * time.Hour)
	return SubscriptionOrder{
		PlanCode:              planCode,
		PromotionCode:         promotionCode,
		ClosingID:             closingID,
		ExternalReference:     fmt.Sprintf("DEMO-SUB-ORDER-%02d-%s", index, f.asOf.Format("20060102")),
		IdempotencyKey:        fmt.Sprintf("demo:subscription-order:%02d:%s", index, f.asOf.Format("20060102")),
		PurchasedAt:           purchasedAt,
		SubscriptionStartDate: dateOnlySeed(purchasedAt),
		Note:                  fmt.Sprintf("Demo Sprint 10 subscription order %02d", index),
	}
}

func (f *Factory) ResolveSubscriptionOrderTopupAmount(ctx context.Context, tx *sql.Tx, ownerID int64, item SubscriptionOrder, desiredAmount string) (string, error) {
	data, err := buildSeedSubscriptionData(ctx, tx, ownerID, item)
	if err != nil {
		return "", err
	}
	finalAmountCents, err := parseSeedMoneyToCents(data.finalAmount)
	if err != nil {
		return "", err
	}
	if finalAmountCents <= 0 {
		return "", fmt.Errorf("seed subscription amount harus positif owner=%d", ownerID)
	}

	wallet, err := lockOrCreateSeedWallet(ctx, tx, ownerID)
	if err != nil {
		return "", err
	}
	currentBalanceCents, err := parseSeedMoneyToCents(wallet.balance)
	if err != nil {
		return "", err
	}

	desiredAmountCents, err := parseSeedMoneyToCents(desiredAmount)
	if err != nil {
		return "", err
	}
	if desiredAmountCents < 0 {
		return "", fmt.Errorf("seed wallet top-up amount tidak boleh negatif owner=%d", ownerID)
	}

	deficitCents := finalAmountCents - currentBalanceCents
	if deficitCents <= 0 || desiredAmountCents >= deficitCents {
		return formatSeedCents(desiredAmountCents), nil
	}
	return formatSeedCents(deficitCents), nil
}

func (f *Factory) CreateSubscriptionOrder(ctx context.Context, tx *sql.Tx, ownerID int64, item SubscriptionOrder) (int64, error) {
	adminID, err := lookupFirstUserIDByRole(ctx, tx, "ADMIN")
	if err != nil {
		return 0, err
	}
	if item.PurchasedAt.IsZero() {
		item.PurchasedAt = f.asOf
	}
	if item.SubscriptionStartDate.IsZero() {
		item.SubscriptionStartDate = dateOnlySeed(item.PurchasedAt)
	}
	idempotencyKey := strings.TrimSpace(item.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("demo:subscription-order:owner:%06d:%s", ownerID, item.ExternalReference)
	}
	if existingID, found, err := lookupExistingSubscriptionOrderByKey(ctx, tx, idempotencyKey); err != nil {
		return 0, err
	} else if found {
		return existingID, nil
	}

	data, err := buildSeedSubscriptionData(ctx, tx, ownerID, item)
	if err != nil {
		return 0, err
	}
	amountCents, err := parseSeedMoneyToCents(data.finalAmount)
	if err != nil {
		return 0, err
	}
	if amountCents <= 0 {
		return 0, fmt.Errorf("seed subscription amount harus positif owner=%d", ownerID)
	}
	wallet, err := lockOrCreateSeedWallet(ctx, tx, ownerID)
	if err != nil {
		return 0, err
	}
	beforeCents, err := parseSeedMoneyToCents(wallet.balance)
	if err != nil {
		return 0, err
	}
	afterCents := beforeCents - amountCents
	var balanceShortfallAmount sql.NullString
	if afterCents < 0 {
		if !item.AllowBalanceShortfall {
			return 0, fmt.Errorf("seed subscription order melebihi saldo owner=%d", ownerID)
		}
		shortfallCents := -afterCents
		balanceShortfallAmount = sql.NullString{String: formatSeedCents(shortfallCents), Valid: true}
		afterCents = 0
	}
	orderCode := fmt.Sprintf("DEMO-ORD-%06d-%s", ownerID, compactSeedKey(idempotencyKey))
	result, err := tx.ExecContext(ctx, `
		INSERT INTO subscription_orders
			(code, owner_id, outlet_id, closing_id, sales_id, supervisor_id, wallet_account_id, package_id, plan_id, promotion_id,
			 package_snapshot_json, plan_snapshot_json, promotion_snapshot_json, tenure_months, duration_days,
			 base_price, discount_amount, additional_charge, final_amount, balance_shortfall_amount, currency, status, idempotency_key, external_reference,
			 purchased_at, subscription_start_date, note, created_by_user_id, updated_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'PAID', ?, ?, ?, ?, ?, ?, ?)`,
		orderCode, ownerID, data.outletID, item.ClosingID, data.salesID, data.supervisorID, wallet.id,
		data.packageID, data.planID, data.promotionID, data.packageSnapshotJSON, data.planSnapshotJSON, data.promotionSnapshotJSON,
		data.tenureMonths, data.durationDays, data.basePrice, data.discountAmount, data.additionalCharge, data.finalAmount,
		balanceShortfallAmount, data.currency, idempotencyKey, nullableSeedString(item.ExternalReference), item.PurchasedAt, item.SubscriptionStartDate,
		nullableSeedString(item.Note), adminID, adminID,
	)
	if err != nil {
		return 0, fmt.Errorf("seed subscription order owner=%d: %w", ownerID, err)
	}
	orderID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, promo := range data.promotionSnapshots {
		promoBytes, err := json.Marshal(promo)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO subscription_order_promotions (order_id, promotion_id, promotion_snapshot_json, additional_charge)
			VALUES (?, ?, ?, ?)`,
			orderID, promo.ID, string(promoBytes), promo.AdditionalCharge,
		); err != nil {
			return 0, fmt.Errorf("seed subscription order promotion owner=%d: %w", ownerID, err)
		}
	}
	if err := insertSeedWalletTransaction(ctx, tx, seedWalletTransactionInput{WalletID: wallet.id, OwnerID: ownerID, TransactionType: "DEBIT", Direction: "DEBIT", AmountCents: amountCents, BalanceBefore: beforeCents, BalanceAfter: afterCents, Currency: wallet.currency, SourceType: "SUBSCRIPTION_ORDER", SourceReference: orderCode, ExternalReference: item.ExternalReference, IdempotencyKey: idempotencyKey, OccurredAt: item.PurchasedAt, Note: item.Note, CreatedByUserID: adminID}); err != nil {
		return 0, err
	}
	transactionID, err := lookupExistingWalletTransactionID(ctx, tx, idempotencyKey)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE subscription_orders SET wallet_transaction_id = ? WHERE id = ?", transactionID, orderID); err != nil {
		return 0, err
	}
	if err := updateSeedWalletBalance(ctx, tx, wallet.id, afterCents); err != nil {
		return 0, err
	}

	endDate := item.SubscriptionStartDate.AddDate(0, 0, data.durationDays)
	subCode := fmt.Sprintf("DEMO-SUB-%06d", orderID)
	subResult, err := tx.ExecContext(ctx, `
		INSERT INTO subscriptions (code, owner_id, outlet_id, order_id, package_id, plan_id, status, active_from, active_until, total_duration_days, source_type)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?, 'SUBSCRIPTION_ORDER')`, subCode, ownerID, data.outletID, orderID, data.packageID, data.planID, item.SubscriptionStartDate, endDate, data.durationDays)
	if err != nil {
		return 0, fmt.Errorf("seed subscription owner=%d: %w", ownerID, err)
	}
	subscriptionID, err := subResult.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO subscription_periods (subscription_id, order_id, owner_id, period_index, start_date, end_date, duration_days, package_id, plan_id, promotion_id, status)
		VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?, ?, 'ACTIVE')`, subscriptionID, orderID, ownerID, item.SubscriptionStartDate, endDate, data.durationDays, data.packageID, data.planID, data.promotionID); err != nil {
		return 0, fmt.Errorf("seed subscription period owner=%d: %w", ownerID, err)
	}

	if item.ClosingID.Valid {
		recCode := fmt.Sprintf("DEMO-REC-%06d", orderID)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO subscription_reconciliations (code, order_id, closing_id, owner_id, status, match_type, amount_difference, note, confirmed_at, created_by_user_id, updated_by_user_id)
			VALUES (?, ?, ?, ?, 'CONFIRMED', 'AUTO', 0, ?, ?, ?, ?)`, recCode, orderID, item.ClosingID, ownerID, nullableSeedString("Auto reconciliation demo seed"), item.PurchasedAt, adminID, adminID); err != nil {
			return 0, fmt.Errorf("seed reconciliation order=%d: %w", orderID, err)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE subscription_orders SET status = 'RECONCILED' WHERE id = ?", orderID); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE sales_closings SET status = 'CONFIRMED', confirmed_at = COALESCE(confirmed_at, ?), updated_by_user_id = ? WHERE id = ? AND deleted_at IS NULL", item.PurchasedAt, adminID, item.ClosingID); err != nil {
			return 0, err
		}
	} else {
		issueCode := fmt.Sprintf("DEMO-RIS-%06d", orderID)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reconciliation_issues (code, order_id, owner_id, issue_type, status, description, detected_at, created_by_user_id, updated_by_user_id)
			VALUES (?, ?, ?, 'HANGING_ORDER', 'OPEN', ?, ?, ?, ?)`, issueCode, orderID, ownerID, nullableSeedString("Demo hanging order: order subscription belum terhubung dengan closing."), item.PurchasedAt, adminID, adminID); err != nil {
			return 0, fmt.Errorf("seed reconciliation issue order=%d: %w", orderID, err)
		}
	}
	return orderID, nil
}

type seedSubscriptionData struct {
	outletID              sql.NullInt64
	salesID               sql.NullInt64
	supervisorID          sql.NullInt64
	packageID             int64
	planID                int64
	promotionID           sql.NullInt64
	packageSnapshotJSON   string
	planSnapshotJSON      string
	promotionSnapshotJSON sql.NullString
	promotionSnapshots    []promotionSeedSnapshot
	tenureMonths          int
	durationDays          int
	basePrice             string
	discountAmount        string
	additionalCharge      string
	finalAmount           string
	currency              string
}

func buildSeedSubscriptionData(ctx context.Context, tx *sql.Tx, ownerID int64, item SubscriptionOrder) (seedSubscriptionData, error) {
	if item.ClosingID.Valid {
		var data seedSubscriptionData
		err := tx.QueryRowContext(ctx, `
			SELECT outlet_id, sales_id, supervisor_id, package_id, plan_id, promotion_id, CAST(package_snapshot_json AS CHAR), CAST(plan_snapshot_json AS CHAR), CAST(promotion_snapshot_json AS CHAR), tenure_months, duration_days, CAST(base_price AS CHAR), CAST(discount_amount AS CHAR), CAST(additional_charge AS CHAR), CAST(final_amount AS CHAR), currency
			FROM sales_closings
			WHERE id = ? AND owner_id = ? AND deleted_at IS NULL
			LIMIT 1`, item.ClosingID, ownerID).Scan(&data.outletID, &data.salesID, &data.supervisorID, &data.packageID, &data.planID, &data.promotionID, &data.packageSnapshotJSON, &data.planSnapshotJSON, &data.promotionSnapshotJSON, &data.tenureMonths, &data.durationDays, &data.basePrice, &data.discountAmount, &data.additionalCharge, &data.finalAmount, &data.currency)
		if err != nil {
			return seedSubscriptionData{}, fmt.Errorf("lookup closing subscription data: %w", err)
		}
		promotionSnapshots, err := lookupClosingPromotionSnapshots(ctx, tx, item.ClosingID.Int64)
		if err != nil {
			return seedSubscriptionData{}, err
		}
		data.promotionSnapshots = promotionSnapshots
		return data, nil
	}
	plan, pkg, err := lookupPlanSnapshot(ctx, tx, item.PlanCode)
	if err != nil {
		return seedSubscriptionData{}, err
	}
	packageBytes, err := json.Marshal(pkg)
	if err != nil {
		return seedSubscriptionData{}, err
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return seedSubscriptionData{}, err
	}
	promotionCodes := item.PromotionCodes
	if len(promotionCodes) == 0 && strings.TrimSpace(item.PromotionCode) != "" {
		promotionCodes = []string{item.PromotionCode}
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
				return seedSubscriptionData{}, err
			}
			cents, err := parseSeedMoneyToCents(promo.AdditionalCharge)
			if err != nil {
				return seedSubscriptionData{}, err
			}
			additionalCents += cents
			promotionSnapshots = append(promotionSnapshots, promo)
		}
		additionalCharge = formatSeedCents(additionalCents)
		first := promotionSnapshots[0]
		promotionID = sql.NullInt64{Int64: first.ID, Valid: true}
		promoBytes, err := json.Marshal(first)
		if err != nil {
			return seedSubscriptionData{}, err
		}
		promotionSnapshot = sql.NullString{String: string(promoBytes), Valid: true}
	}
	baseCents, err := parseSeedMoneyToCents(plan.Price)
	if err != nil {
		return seedSubscriptionData{}, err
	}
	additionalCents, err := parseSeedMoneyToCents(additionalCharge)
	if err != nil {
		return seedSubscriptionData{}, err
	}
	salesID, supervisorID, err := lookupSeedLeadPIC(ctx, tx, ownerID)
	if err != nil {
		return seedSubscriptionData{}, err
	}
	return seedSubscriptionData{salesID: salesID, supervisorID: supervisorID, packageID: pkg.ID, planID: plan.ID, promotionID: promotionID, packageSnapshotJSON: string(packageBytes), planSnapshotJSON: string(planBytes), promotionSnapshotJSON: promotionSnapshot, promotionSnapshots: promotionSnapshots, tenureMonths: plan.TenureMonths, durationDays: plan.DurationDays, basePrice: plan.Price, discountAmount: "0.00", additionalCharge: additionalCharge, finalAmount: formatSeedCents(baseCents + additionalCents), currency: plan.Currency}, nil
}

func lookupClosingPromotionSnapshots(ctx context.Context, tx *sql.Tx, closingID int64) ([]promotionSeedSnapshot, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT promotion_id, CAST(promotion_snapshot_json AS CHAR)
		FROM sales_closing_promotions
		WHERE closing_id = ?
		ORDER BY id`, closingID)
	if err != nil {
		return nil, fmt.Errorf("lookup closing promotions closing=%d: %w", closingID, err)
	}
	defer rows.Close()

	var snapshots []promotionSeedSnapshot
	for rows.Next() {
		var promotionID int64
		var snapshotJSON string
		if err := rows.Scan(&promotionID, &snapshotJSON); err != nil {
			return nil, err
		}
		var snapshot promotionSeedSnapshot
		if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
			return nil, fmt.Errorf("decode closing promotion snapshot closing=%d promotion=%d: %w", closingID, promotionID, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func lookupSeedLeadPIC(ctx context.Context, tx *sql.Tx, ownerID int64) (sql.NullInt64, sql.NullInt64, error) {
	var salesID sql.NullInt64
	var supervisorID sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT CASE WHEN current_owner_role = 'SALES' THEN current_owner_user_id ELSE active_sales_id END AS sales_id, supervisor_id
		FROM customer_leads
		WHERE owner_id = ? AND deleted_at IS NULL
		ORDER BY updated_at DESC, id DESC
		LIMIT 1`, ownerID).Scan(&salesID, &supervisorID)
	if err == sql.ErrNoRows {
		return sql.NullInt64{}, sql.NullInt64{}, nil
	}
	if err != nil {
		return sql.NullInt64{}, sql.NullInt64{}, err
	}
	return salesID, supervisorID, nil
}

func lookupExistingSubscriptionOrderByKey(ctx context.Context, tx *sql.Tx, key string) (int64, bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, "SELECT id FROM subscription_orders WHERE idempotency_key = ? AND deleted_at IS NULL LIMIT 1", key).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lookup subscription order key=%s: %w", key, err)
	}
	return id, true, nil
}

func dateOnlySeed(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
