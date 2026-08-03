package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

const (
	closingStatusPending   = "PENDING_RECONCILIATION"
	closingStatusConfirmed = "CONFIRMED"
	closingStatusRejected  = "REJECTED"

	benefitTypeFreeDuration = "FREE_DURATION"
)

type Repository struct {
	db *sql.DB
}

type queryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type OrderResult struct {
	Order          SubscriptionOrder
	Subscription   Subscription
	Period         SubscriptionPeriod
	Reconciliation *Reconciliation
	Issue          *ReconciliationIssue
	Idempotent     bool
}

type ReconciliationResult struct {
	Order          SubscriptionOrder
	Reconciliation Reconciliation
	Issue          *ReconciliationIssue
}

type walletAccount struct {
	ID       int64
	OwnerID  sql.NullInt64
	Currency string
	Balance  string
}

type closingSnapshot struct {
	ID                    int64
	Code                  string
	OwnerID               sql.NullInt64
	OutletID              sql.NullInt64
	SalesID               sql.NullInt64
	SupervisorID          sql.NullInt64
	PackageID             sql.NullInt64
	PlanID                sql.NullInt64
	PromotionID           sql.NullInt64
	PackageSnapshotJSON   string
	PlanSnapshotJSON      string
	PromotionSnapshotJSON sql.NullString
	TenureMonths          int
	DurationDays          int
	BasePrice             string
	DiscountAmount        string
	AdditionalCharge      string
	FinalAmount           string
	Currency              string
	Status                string
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListOrders(ctx context.Context, actor identity.User, params ListParams) ([]SubscriptionOrder, int64, error) {
	where, args := orderWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM subscription_orders so
LEFT JOIN owners o ON o.id = so.owner_id AND o.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.id = so.closing_id AND sc.deleted_at IS NULL
WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := orderOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := orderSelect() + `
WHERE ` + where + `
ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanOrders(rows, total)
}

func (r *Repository) FindOrderByID(ctx context.Context, actor identity.User, id int64) (SubscriptionOrder, error) {
	where, args := orderWhere(actor, ListParams{})
	args = append([]any{id}, args...)
	item, err := scanOrder(r.db.QueryRowContext(ctx, orderSelect()+`
WHERE so.id = ? AND `+where+`
LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return SubscriptionOrder{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ListSubscriptions(ctx context.Context, actor identity.User, params ListParams) ([]Subscription, int64, error) {
	where, args := subscriptionWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM subscriptions s
LEFT JOIN owners o ON o.id = s.owner_id AND o.deleted_at IS NULL
WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := subscriptionOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := subscriptionSelect() + `
WHERE ` + where + `
ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanSubscriptions(rows, total)
}

func (r *Repository) FindSubscriptionByID(ctx context.Context, actor identity.User, id int64) (Subscription, error) {
	where, args := subscriptionWhere(actor, ListParams{})
	args = append([]any{id}, args...)
	item, err := scanSubscription(r.db.QueryRowContext(ctx, subscriptionSelect()+`
WHERE s.id = ? AND `+where+`
LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return Subscription{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ListReconciliations(ctx context.Context, actor identity.User, params ListParams) ([]Reconciliation, int64, error) {
	where, args := reconciliationWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM subscription_reconciliations sr
LEFT JOIN owners o ON o.id = sr.owner_id AND o.deleted_at IS NULL
LEFT JOIN subscription_orders so ON so.id = sr.order_id AND so.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.id = sr.closing_id AND sc.deleted_at IS NULL
WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := reconciliationOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := reconciliationSelect() + `
WHERE ` + where + `
ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReconciliations(rows, total)
}

func (r *Repository) ListIssues(ctx context.Context, actor identity.User, params ListParams) ([]ReconciliationIssue, int64, error) {
	where, args := issueWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM reconciliation_issues ri
LEFT JOIN owners o ON o.id = ri.owner_id AND o.deleted_at IS NULL
LEFT JOIN subscription_orders so ON so.id = ri.order_id AND so.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.id = ri.closing_id AND sc.deleted_at IS NULL
WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := issueOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := issueSelect() + `
WHERE ` + where + `
ORDER BY ` + orderBy
	if !params.All {
		offset := (params.Page - 1) * params.Limit
		args = append(args, params.Limit, offset)
		query += `
LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanIssues(rows, total)
}

func (r *Repository) CreateOrder(ctx context.Context, actor identity.User, ownerID int64, req CreateOrderRequest) (OrderResult, error) {
	key, err := orderIdempotencyKey(req)
	if err != nil {
		return OrderResult{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResult{}, err
	}
	defer tx.Rollback()

	if existing, found, err := r.findOrderByIdempotency(ctx, tx, key); err != nil {
		return OrderResult{}, err
	} else if found {
		result, err := r.orderResultFromOrder(ctx, tx, existing, true)
		if err != nil {
			return OrderResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return OrderResult{}, err
		}
		return result, nil
	}

	if err := r.ensureOwnerExists(ctx, tx, ownerID); err != nil {
		return OrderResult{}, err
	}
	purchasedAt := time.Now().UTC()
	if req.PurchasedAt != nil {
		purchasedAt = req.PurchasedAt.UTC()
	}
	startDate, err := subscriptionStartDate(req.SubscriptionStartDate, purchasedAt)
	if err != nil {
		return OrderResult{}, err
	}

	input, err := r.buildOrderInput(ctx, tx, ownerID, req, purchasedAt)
	if err != nil {
		return OrderResult{}, err
	}
	if input.OutletID.Valid {
		if err := r.ensureOutletBelongsToOwner(ctx, tx, ownerID, input.OutletID.Int64); err != nil {
			return OrderResult{}, err
		}
	}

	finalAmountCents, err := parsePositiveMoneyToCents(input.FinalAmount)
	if err != nil {
		return OrderResult{}, err
	}
	wallet, err := r.lockOrCreateWallet(ctx, tx, ownerID)
	if err != nil {
		return OrderResult{}, err
	}
	balanceBeforeCents, err := parseMoneyToCents(wallet.Balance)
	if err != nil {
		return OrderResult{}, err
	}
	balanceAfterCents := balanceBeforeCents - finalAmountCents
	var shortfallCents int64
	if balanceAfterCents < 0 {
		// Non-Admin callers hit a real, live debit (Sales/Supervisor creating an order tied to an
		// actual closing) — insufficient balance still hard-blocks. Admin creating a manual order
		// is backfilling/correcting a purchase that already happened in the real Piposmart app
		// (where balance was presumably sufficient) — proceed, clamp the wallet to 0 (never
		// negative: wallet_accounts.balance has a CHECK >= 0), and flag the difference instead.
		if actor.RoleCode != RoleAdmin {
			return OrderResult{}, ErrInsufficientBalance
		}
		shortfallCents = -balanceAfterCents
		balanceAfterCents = 0
	}

	orderCode := nextCode("ORD", time.Now().UTC(), ownerID)
	var balanceShortfallAmount sql.NullString
	if shortfallCents > 0 {
		balanceShortfallAmount = sql.NullString{String: formatCents(shortfallCents), Valid: true}
	}
	orderID, err := r.insertOrder(ctx, tx, actor, orderCode, key, ownerID, wallet.ID, input, req, purchasedAt, startDate, balanceShortfallAmount)
	if err != nil {
		return OrderResult{}, err
	}
	for _, snapshot := range input.PromotionSnapshots {
		snapshotBytes, err := json.Marshal(snapshot)
		if err != nil {
			return OrderResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO subscription_order_promotions (order_id, promotion_id, promotion_snapshot_json, additional_charge)
			VALUES (?, ?, ?, ?)`,
			orderID, snapshot.ID, string(snapshotBytes), snapshot.AdditionalCharge,
		); err != nil {
			return OrderResult{}, err
		}
	}
	transactionID, err := r.insertLedgerTransaction(ctx, tx, ledgerInput{
		Wallet:            wallet,
		OrderCode:         orderCode,
		AmountCents:       finalAmountCents,
		BalanceBefore:     balanceBeforeCents,
		BalanceAfter:      balanceAfterCents,
		ExternalReference: req.ExternalReference,
		IdempotencyKey:    key,
		OccurredAt:        purchasedAt,
		Note:              req.Note,
		ActorID:           actor.ID,
	})
	if err != nil {
		return OrderResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE subscription_orders
SET wallet_transaction_id = ?, updated_by_user_id = ?
WHERE id = ?`, transactionID, actor.ID, orderID); err != nil {
		return OrderResult{}, err
	}
	if err := r.updateWalletBalance(ctx, tx, wallet.ID, balanceAfterCents); err != nil {
		return OrderResult{}, err
	}

	subscriptionID, periodID, err := r.activateSubscription(ctx, tx, orderID, ownerID, input, startDate)
	if err != nil {
		return OrderResult{}, err
	}

	var reconciliationID int64
	var issueID int64
	if input.ClosingID.Valid {
		if err := r.ensureClosingNotReconciledByOtherOrder(ctx, tx, input.ClosingID.Int64, orderID); err != nil {
			return OrderResult{}, err
		}
		reconciliationID, err = r.insertReconciliation(ctx, tx, reconciliationInput{
			OrderID:          sql.NullInt64{Int64: orderID, Valid: true},
			ClosingID:        input.ClosingID,
			OwnerID:          sql.NullInt64{Int64: ownerID, Valid: true},
			Status:           ReconciliationStatusConfirmed,
			MatchType:        ReconciliationMatchAuto,
			AmountDifference: "0.00",
			Note:             nullableString("Auto reconciliation dari closing dan pembelian subscription"),
			ConfirmedAt:      sql.NullTime{Time: time.Now().UTC(), Valid: true},
			ActorID:          actor.ID,
		})
		if err != nil {
			return OrderResult{}, err
		}
		if err := r.confirmOrderAndClosing(ctx, tx, actor.ID, orderID, input.ClosingID.Int64); err != nil {
			return OrderResult{}, err
		}
	} else {
		issueID, err = r.insertIssue(ctx, tx, issueInput{
			OrderID:     sql.NullInt64{Int64: orderID, Valid: true},
			OwnerID:     sql.NullInt64{Int64: ownerID, Valid: true},
			IssueType:   IssueHangingOrder,
			Description: nullableString("Subscription order belum terhubung dengan laporan closing sales."),
			DetectedAt:  purchasedAt,
			ActorID:     actor.ID,
		})
		if err != nil {
			return OrderResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return OrderResult{}, err
	}
	return r.orderResultByIDs(ctx, r.db, orderID, subscriptionID, periodID, reconciliationID, issueID, false)
}

func (r *Repository) CreateUpgrade(ctx context.Context, actor identity.User, subscriptionID int64, req CreateUpgradeRequest) (OrderResult, error) {
	key, err := upgradeIdempotencyKey(req)
	if err != nil {
		return OrderResult{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResult{}, err
	}
	defer tx.Rollback()

	if existing, found, err := r.findOrderByIdempotency(ctx, tx, key); err != nil {
		return OrderResult{}, err
	} else if found {
		result, err := r.orderResultFromOrder(ctx, tx, existing, true)
		if err != nil {
			return OrderResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return OrderResult{}, err
		}
		return result, nil
	}

	sourceSubscription, err := r.lockSubscriptionByID(ctx, tx, subscriptionID)
	if err != nil {
		return OrderResult{}, err
	}
	if sourceSubscription.Status != SubscriptionStatusActive {
		return OrderResult{}, ErrSubscriptionNotActive
	}
	if !sourceSubscription.OwnerID.Valid || !sourceSubscription.OrderID.Valid {
		return OrderResult{}, ErrUpgradeNotAllowed
	}

	sourceOrder, err := r.findOrderByIDRaw(ctx, tx, sourceSubscription.OrderID.Int64)
	if err != nil {
		return OrderResult{}, err
	}
	sourcePeriod, err := r.findPeriodByOrderID(ctx, tx, sourceOrder.ID)
	if err != nil {
		return OrderResult{}, err
	}

	purchasedAt := time.Now().UTC()
	if req.PurchasedAt != nil {
		purchasedAt = req.PurchasedAt.UTC()
	}
	effectiveStart, err := effectiveUpgradeStartDate(req.EffectiveStartDate, purchasedAt)
	if err != nil {
		return OrderResult{}, err
	}
	if effectiveStart.Before(sourceSubscription.ActiveFrom) {
		return OrderResult{}, fmt.Errorf("%w: tanggal efektif upgrade (%s) tidak boleh sebelum tanggal mulai berlangganan (%s)", ErrUpgradeNotAllowed, effectiveStart.Format("2006-01-02"), sourceSubscription.ActiveFrom.Format("2006-01-02"))
	}
	if !effectiveStart.Before(sourceSubscription.ActiveUntil) {
		return OrderResult{}, fmt.Errorf("%w: tanggal efektif upgrade (%s) harus sebelum tanggal berakhir berlangganan (%s)", ErrUpgradeNotAllowed, effectiveStart.Format("2006-01-02"), sourceSubscription.ActiveUntil.Format("2006-01-02"))
	}

	remainingDays := businessDateDiff(effectiveStart, sourceSubscription.ActiveUntil)
	if remainingDays < 1 {
		return OrderResult{}, fmt.Errorf("%w: sisa hari berlangganan kurang dari 1 hari (%d hari)", ErrUpgradeNotAllowed, remainingDays)
	}
	usedDays := businessDateDiff(sourceSubscription.ActiveFrom, effectiveStart)
	if usedDays < 0 {
		return OrderResult{}, fmt.Errorf("%w: tanggal efektif upgrade sebelum tanggal mulai berlangganan", ErrUpgradeNotAllowed)
	}

	previousPackage, previousPlan, err := snapshotsFromOrder(sourceOrder)
	if err != nil {
		return OrderResult{}, err
	}
	targetPackage, targetPlan, err := r.findPlanSnapshot(ctx, tx, req.PlanID, purchasedAt)
	if err != nil {
		return OrderResult{}, err
	}
	if targetPackage.LevelOrder <= previousPackage.LevelOrder {
		return OrderResult{}, fmt.Errorf("%w: level paket tujuan (%s: level %d) harus lebih tinggi dari paket saat ini (%s: level %d)", ErrUpgradeNotAllowed, targetPackage.Name, targetPackage.LevelOrder, previousPackage.Name, previousPackage.LevelOrder)
	}

	var closing *closingSnapshot
	if req.ClosingID != nil {
		item, err := r.lockClosing(ctx, tx, *req.ClosingID)
		if err != nil {
			return OrderResult{}, err
		}
		if !item.OwnerID.Valid || item.OwnerID.Int64 != sourceSubscription.OwnerID.Int64 || item.Status == closingStatusRejected {
			return OrderResult{}, ErrClosingMismatch
		}
		if sourceSubscription.OutletID.Valid && item.OutletID.Valid && sourceSubscription.OutletID.Int64 != item.OutletID.Int64 {
			return OrderResult{}, ErrClosingMismatch
		}
		if !item.PlanID.Valid || item.PlanID.Int64 != req.PlanID {
			return OrderResult{}, ErrClosingMismatch
		}
		closing = &item
	}

	proratedFinalCents, dailyCents, err := proratedPlanAmount(targetPlan.Price, targetPlan.DurationDays, remainingDays)
	if err != nil {
		return OrderResult{}, err
	}
	targetPackageJSON, err := json.Marshal(targetPackage)
	if err != nil {
		return OrderResult{}, err
	}
	targetPlanJSON, err := json.Marshal(targetPlan)
	if err != nil {
		return OrderResult{}, err
	}
	previousPackageJSON, err := json.Marshal(previousPackage)
	if err != nil {
		return OrderResult{}, err
	}
	previousPlanJSON, err := json.Marshal(previousPlan)
	if err != nil {
		return OrderResult{}, err
	}

	input := orderInput{
		OutletID:                    sourceSubscription.OutletID,
		SalesID:                     sourceOrder.SalesID,
		SupervisorID:                sourceOrder.SupervisorID,
		PackageID:                   sql.NullInt64{Int64: targetPackage.ID, Valid: true},
		PlanID:                      sql.NullInt64{Int64: targetPlan.ID, Valid: true},
		PackageSnapshotJSON:         string(targetPackageJSON),
		PlanSnapshotJSON:            string(targetPlanJSON),
		OrderType:                   OrderTypeUpgrade,
		SourceSubscriptionID:        sql.NullInt64{Int64: sourceSubscription.ID, Valid: true},
		UpgradeEffectiveStartDate:   sql.NullTime{Time: effectiveStart, Valid: true},
		UpgradeOriginalEndDate:      sql.NullTime{Time: sourceSubscription.ActiveUntil, Valid: true},
		UpgradeRemainingDays:        sql.NullInt64{Int64: int64(remainingDays), Valid: true},
		UpgradeDailyPrice:           sql.NullString{String: formatCents(dailyCents), Valid: true},
		PreviousPackageSnapshotJSON: sql.NullString{String: string(previousPackageJSON), Valid: true},
		PreviousPlanSnapshotJSON:    sql.NullString{String: string(previousPlanJSON), Valid: true},
		TenureMonths:                targetPlan.TenureMonths,
		DurationDays:                remainingDays,
		BasePrice:                   formatCents(proratedFinalCents),
		DiscountAmount:              "0.00",
		AdditionalCharge:            "0.00",
		FinalAmount:                 formatCents(proratedFinalCents),
		Currency:                    targetPlan.Currency,
	}
	if closing != nil {
		input.ClosingID = sql.NullInt64{Int64: closing.ID, Valid: true}
		input.SalesID = closing.SalesID
		input.SupervisorID = closing.SupervisorID
	}

	wallet, err := r.lockOrCreateWallet(ctx, tx, sourceSubscription.OwnerID.Int64)
	if err != nil {
		return OrderResult{}, err
	}
	balanceBeforeCents, err := parseMoneyToCents(wallet.Balance)
	if err != nil {
		return OrderResult{}, err
	}
	balanceAfterCents := balanceBeforeCents - proratedFinalCents
	var shortfallCents int64
	if balanceAfterCents < 0 {
		if actor.RoleCode != RoleAdmin {
			return OrderResult{}, ErrInsufficientBalance
		}
		shortfallCents = -balanceAfterCents
		balanceAfterCents = 0
	}

	orderCode := nextCode("ORD", time.Now().UTC(), sourceSubscription.OwnerID.Int64)
	var balanceShortfallAmount sql.NullString
	if shortfallCents > 0 {
		balanceShortfallAmount = sql.NullString{String: formatCents(shortfallCents), Valid: true}
	}
	orderReq := CreateOrderRequest{
		ExternalReference: req.ExternalReference,
		Note:              req.Note,
	}
	orderID, err := r.insertOrder(ctx, tx, actor, orderCode, key, sourceSubscription.OwnerID.Int64, wallet.ID, input, orderReq, purchasedAt, effectiveStart, balanceShortfallAmount)
	if err != nil {
		return OrderResult{}, err
	}

	transactionID, err := r.insertLedgerTransaction(ctx, tx, ledgerInput{
		Wallet:            wallet,
		OrderCode:         orderCode,
		AmountCents:       proratedFinalCents,
		BalanceBefore:     balanceBeforeCents,
		BalanceAfter:      balanceAfterCents,
		ExternalReference: req.ExternalReference,
		IdempotencyKey:    key,
		OccurredAt:        purchasedAt,
		Note:              req.Note,
		SourceType:        WalletSourceSubscriptionUpgrade,
		ActorID:           actor.ID,
	})
	if err != nil {
		return OrderResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE subscription_orders
SET wallet_transaction_id = ?, updated_by_user_id = ?
WHERE id = ?`, transactionID, actor.ID, orderID); err != nil {
		return OrderResult{}, err
	}
	if err := r.updateWalletBalance(ctx, tx, wallet.ID, balanceAfterCents); err != nil {
		return OrderResult{}, err
	}
	if err := r.trimSubscriptionForUpgrade(ctx, tx, sourceSubscription, sourcePeriod, effectiveStart, usedDays); err != nil {
		return OrderResult{}, err
	}
	subscriptionIDNew, periodID, err := r.activateSubscriptionForRange(ctx, tx, orderID, sourceSubscription.OwnerID.Int64, input, effectiveStart, sourceSubscription.ActiveUntil, WalletSourceSubscriptionUpgrade)
	if err != nil {
		return OrderResult{}, err
	}
	var reconciliationID int64
	var issueID int64
	if closing != nil {
		if err := r.ensureClosingNotReconciledByOtherOrder(ctx, tx, closing.ID, orderID); err != nil {
			return OrderResult{}, err
		}
		amountDifference, err := moneyDifference(input.FinalAmount, closing.FinalAmount)
		if err != nil {
			return OrderResult{}, err
		}
		now := time.Now().UTC()
		if amountDifference == "0.00" || amountDifference == "-0.00" {
			reconciliationID, err = r.insertReconciliation(ctx, tx, reconciliationInput{
				OrderID:          sql.NullInt64{Int64: orderID, Valid: true},
				ClosingID:        sql.NullInt64{Int64: closing.ID, Valid: true},
				OwnerID:          sql.NullInt64{Int64: sourceSubscription.OwnerID.Int64, Valid: true},
				Status:           ReconciliationStatusConfirmed,
				MatchType:        ReconciliationMatchAuto,
				AmountDifference: "0.00",
				Note:             nullableString("Auto reconciliation dari subscription upgrade dan closing sales."),
				ConfirmedAt:      sql.NullTime{Time: now, Valid: true},
				ActorID:          actor.ID,
			})
			if err != nil {
				return OrderResult{}, err
			}
			if err := r.confirmOrderAndClosing(ctx, tx, actor.ID, orderID, closing.ID); err != nil {
				return OrderResult{}, err
			}
		} else {
			reconciliationID, err = r.insertReconciliation(ctx, tx, reconciliationInput{
				OrderID:          sql.NullInt64{Int64: orderID, Valid: true},
				ClosingID:        sql.NullInt64{Int64: closing.ID, Valid: true},
				OwnerID:          sql.NullInt64{Int64: sourceSubscription.OwnerID.Int64, Valid: true},
				Status:           ReconciliationStatusPartialConfirm,
				MatchType:        ReconciliationMatchAuto,
				AmountDifference: amountDifference,
				AdminFinalAmount: sql.NullString{String: input.FinalAmount, Valid: true},
				Note:             nullableString("Auto partial reconciliation dari subscription upgrade; closing dipin ke nominal prorata upgrade."),
				ConfirmedAt:      sql.NullTime{Time: now, Valid: true},
				ActorID:          actor.ID,
			})
			if err != nil {
				return OrderResult{}, err
			}
			if err := r.confirmOrderAndClosingWithAmount(ctx, tx, actor.ID, orderID, closing.ID, input.FinalAmount); err != nil {
				return OrderResult{}, err
			}
		}
		if err := r.resolveIssues(ctx, tx, actor.ID, orderID, closing.ID, now); err != nil {
			return OrderResult{}, err
		}
	} else {
		issueID, err = r.insertIssue(ctx, tx, issueInput{
			OrderID:     sql.NullInt64{Int64: orderID, Valid: true},
			OwnerID:     sql.NullInt64{Int64: sourceSubscription.OwnerID.Int64, Valid: true},
			IssueType:   IssueHangingOrder,
			Description: nullableString("Subscription upgrade belum terhubung dengan laporan closing sales."),
			DetectedAt:  purchasedAt,
			ActorID:     actor.ID,
		})
		if err != nil {
			return OrderResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return OrderResult{}, err
	}
	return r.orderResultByIDs(ctx, r.db, orderID, subscriptionIDNew, periodID, reconciliationID, issueID, false)
}

func (r *Repository) ReconcileOrder(ctx context.Context, actor identity.User, orderID int64, req ManualReconcileRequest) (ReconciliationResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReconciliationResult{}, err
	}
	defer tx.Rollback()

	order, err := r.lockOrder(ctx, tx, orderID)
	if err != nil {
		return ReconciliationResult{}, err
	}
	if !canManageOrder(actor, order) {
		return ReconciliationResult{}, ErrForbidden
	}
	if order.Status == OrderStatusReconciled {
		return ReconciliationResult{}, ErrOrderAlreadyReconciled
	}
	closing, err := r.lockClosing(ctx, tx, req.ClosingID)
	if err != nil {
		return ReconciliationResult{}, err
	}
	if closing.Status == closingStatusConfirmed {
		if err := r.ensureClosingNotReconciledByOtherOrder(ctx, tx, closing.ID, orderID); err != nil {
			return ReconciliationResult{}, err
		}
	}
	amountDifference, err := moneyDifference(order.FinalAmount, closing.FinalAmount)
	if err != nil {
		return ReconciliationResult{}, err
	}
	issueCode := sql.NullString{}
	if !sameNullableID(order.OwnerID, closing.OwnerID) {
		issueCode = sql.NullString{String: IssueOwnerMismatch, Valid: true}
		normalizedAction := normalizeAction(req.Action)
		if normalizedAction == ReconciliationStatusConfirmed || normalizedAction == ReconciliationStatusPartialConfirm {
			return ReconciliationResult{}, ErrClosingMismatch
		}
	} else if amountDifference != "0.00" && amountDifference != "-0.00" {
		issueCode = sql.NullString{String: IssueAmountMismatch, Valid: true}
	}

	now := time.Now().UTC()
	action := normalizeAction(req.Action)
	var reconciliationID int64
	var issueID int64
	switch action {
	case ReconciliationStatusConfirmed:
		reconciliationID, err = r.upsertReconciliation(ctx, tx, reconciliationInput{
			OrderID:          sql.NullInt64{Int64: orderID, Valid: true},
			ClosingID:        sql.NullInt64{Int64: closing.ID, Valid: true},
			OwnerID:          order.OwnerID,
			Status:           ReconciliationStatusConfirmed,
			MatchType:        ReconciliationMatchManual,
			IssueCode:        issueCode,
			AmountDifference: amountDifference,
			Note:             nullableString(req.Note),
			Reason:           nullableString(req.Reason),
			ConfirmedAt:      sql.NullTime{Time: now, Valid: true},
			ActorID:          actor.ID,
		})
		if err != nil {
			return ReconciliationResult{}, err
		}
		if err := r.confirmOrderAndClosing(ctx, tx, actor.ID, orderID, closing.ID); err != nil {
			return ReconciliationResult{}, err
		}
		if err := r.resolveIssues(ctx, tx, actor.ID, orderID, closing.ID, now); err != nil {
			return ReconciliationResult{}, err
		}
	case ReconciliationStatusPartialConfirm:
		if req.AdminFinalAmount == nil || strings.TrimSpace(*req.AdminFinalAmount) == "" {
			return ReconciliationResult{}, ErrInvalidAction
		}
		adminFinalAmountCents, err := parsePositiveMoneyToCents(*req.AdminFinalAmount)
		if err != nil {
			return ReconciliationResult{}, err
		}
		adminFinalAmount := formatCents(adminFinalAmountCents)
		var adminTenureMonths sql.NullInt64
		if req.AdminTenureMonths != nil {
			adminTenureMonths = sql.NullInt64{Int64: int64(*req.AdminTenureMonths), Valid: true}
		}
		reconciliationID, err = r.upsertReconciliation(ctx, tx, reconciliationInput{
			OrderID:           sql.NullInt64{Int64: orderID, Valid: true},
			ClosingID:         sql.NullInt64{Int64: closing.ID, Valid: true},
			OwnerID:           order.OwnerID,
			Status:            ReconciliationStatusPartialConfirm,
			MatchType:         ReconciliationMatchManual,
			IssueCode:         issueCode,
			AmountDifference:  amountDifference,
			AdminTenureMonths: adminTenureMonths,
			AdminFinalAmount:  sql.NullString{String: adminFinalAmount, Valid: true},
			Note:              nullableString(req.Note),
			Reason:            nullableString(req.Reason),
			ConfirmedAt:       sql.NullTime{Time: now, Valid: true},
			ActorID:           actor.ID,
		})
		if err != nil {
			return ReconciliationResult{}, err
		}
		if err := r.confirmOrderAndClosingWithAmount(ctx, tx, actor.ID, orderID, closing.ID, adminFinalAmount); err != nil {
			return ReconciliationResult{}, err
		}
		if err := r.resolveIssues(ctx, tx, actor.ID, orderID, closing.ID, now); err != nil {
			return ReconciliationResult{}, err
		}
	case ReconciliationStatusRejected:
		reconciliationID, err = r.upsertReconciliation(ctx, tx, reconciliationInput{
			OrderID:          sql.NullInt64{Int64: orderID, Valid: true},
			ClosingID:        sql.NullInt64{Int64: closing.ID, Valid: true},
			OwnerID:          order.OwnerID,
			Status:           ReconciliationStatusRejected,
			MatchType:        ReconciliationMatchManual,
			IssueCode:        issueCode,
			AmountDifference: amountDifference,
			Note:             nullableString(req.Note),
			Reason:           nullableString(req.Reason),
			RejectedAt:       sql.NullTime{Time: now, Valid: true},
			ActorID:          actor.ID,
		})
		if err != nil {
			return ReconciliationResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE subscription_orders
SET status = ?, updated_by_user_id = ?
WHERE id = ? AND deleted_at IS NULL`, OrderStatusRejected, actor.ID, orderID); err != nil {
			return ReconciliationResult{}, err
		}
		if closing.Status == closingStatusPending {
			if _, err := tx.ExecContext(ctx, `
UPDATE sales_closings
SET status = ?, rejection_reason = ?, rejected_at = ?, updated_by_user_id = ?
WHERE id = ? AND deleted_at IS NULL`, closingStatusRejected, nullableString(req.Reason), now, actor.ID, closing.ID); err != nil {
				return ReconciliationResult{}, err
			}
		}
		issueType := IssueAmountMismatch
		if issueCode.Valid {
			issueType = issueCode.String
		}
		issueID, err = r.insertIssue(ctx, tx, issueInput{
			OrderID:     sql.NullInt64{Int64: orderID, Valid: true},
			ClosingID:   sql.NullInt64{Int64: closing.ID, Valid: true},
			OwnerID:     order.OwnerID,
			IssueType:   issueType,
			Description: nullableString(firstNonEmpty(req.Reason, "Reconciliation ditolak dan perlu review ulang.")),
			DetectedAt:  now,
			ActorID:     actor.ID,
		})
		if err != nil {
			return ReconciliationResult{}, err
		}
	default:
		return ReconciliationResult{}, ErrInvalidAction
	}

	if err := tx.Commit(); err != nil {
		return ReconciliationResult{}, err
	}
	order, err = r.findOrderByIDRaw(ctx, r.db, orderID)
	if err != nil {
		return ReconciliationResult{}, err
	}
	reconciliation, err := r.findReconciliationByIDRaw(ctx, r.db, reconciliationID)
	if err != nil {
		return ReconciliationResult{}, err
	}
	var issue *ReconciliationIssue
	if issueID > 0 {
		item, err := r.findIssueByIDRaw(ctx, r.db, issueID)
		if err != nil {
			return ReconciliationResult{}, err
		}
		issue = &item
	}
	return ReconciliationResult{Order: order, Reconciliation: reconciliation, Issue: issue}, nil
}

type orderInput struct {
	OutletID              sql.NullInt64
	ClosingID             sql.NullInt64
	SalesID               sql.NullInt64
	SupervisorID          sql.NullInt64
	PackageID             sql.NullInt64
	PlanID                sql.NullInt64
	PromotionID           sql.NullInt64
	PackageSnapshotJSON   string
	PlanSnapshotJSON      string
	PromotionSnapshotJSON sql.NullString
	// PromotionSnapshots is the full stacked promotion list (Sprint 15a §4b) — PromotionID/
	// PromotionSnapshotJSON above only ever hold the first, for backward compat.
	PromotionSnapshots []PromotionSnapshot
	TenureMonths       int
	DurationDays       int
	BasePrice          string
	DiscountAmount     string
	AdditionalCharge   string
	FinalAmount        string
	Currency           string
}

func (r *Repository) buildOrderInput(ctx context.Context, tx *sql.Tx, ownerID int64, req CreateOrderRequest, purchasedAt time.Time) (orderInput, error) {
	if req.ClosingID != nil {
		closing, err := r.lockClosing(ctx, tx, *req.ClosingID)
		if err != nil {
			return orderInput{}, err
		}
		if !closing.OwnerID.Valid || closing.OwnerID.Int64 != ownerID || closing.Status == closingStatusRejected {
			return orderInput{}, ErrClosingMismatch
		}
		promotionSnapshots, err := r.findClosingPromotionSnapshots(ctx, tx, closing.ID)
		if err != nil {
			return orderInput{}, err
		}
		durationDays := totalDurationDaysMulti(closing.DurationDays, promotionSnapshots)
		return orderInput{
			OutletID:              closing.OutletID,
			ClosingID:             sql.NullInt64{Int64: closing.ID, Valid: true},
			SalesID:               closing.SalesID,
			SupervisorID:          closing.SupervisorID,
			PackageID:             closing.PackageID,
			PlanID:                closing.PlanID,
			PromotionID:           closing.PromotionID,
			PackageSnapshotJSON:   closing.PackageSnapshotJSON,
			PlanSnapshotJSON:      closing.PlanSnapshotJSON,
			PromotionSnapshotJSON: closing.PromotionSnapshotJSON,
			PromotionSnapshots:    promotionSnapshots,
			TenureMonths:          closing.TenureMonths,
			DurationDays:          durationDays,
			BasePrice:             closing.BasePrice,
			DiscountAmount:        closing.DiscountAmount,
			AdditionalCharge:      closing.AdditionalCharge,
			FinalAmount:           closing.FinalAmount,
			Currency:              closing.Currency,
		}, nil
	}
	if req.PlanID < 1 {
		return orderInput{}, ErrInvalidRequest
	}
	pkgSnapshot, planSnapshot, err := r.findPlanSnapshot(ctx, tx, req.PlanID, purchasedAt)
	if err != nil {
		return orderInput{}, err
	}
	promotionIDs := req.PromotionIDs
	if len(promotionIDs) == 0 && req.PromotionID != nil {
		promotionIDs = []int64{*req.PromotionID}
	}
	additionalCharge := "0.00"
	var promotionID sql.NullInt64
	var promotionSnapshotJSON sql.NullString
	var promotionSnapshots []PromotionSnapshot
	if len(promotionIDs) > 0 {
		additionalChargeCents := int64(0)
		for _, id := range promotionIDs {
			snapshot, err := r.findEligiblePromotionSnapshot(ctx, tx, id, req.PlanID, purchasedAt)
			if err != nil {
				return orderInput{}, err
			}
			cents, err := parseMoneyToCents(snapshot.AdditionalCharge)
			if err != nil {
				return orderInput{}, err
			}
			additionalChargeCents += cents
			promotionSnapshots = append(promotionSnapshots, snapshot)
		}
		additionalCharge = formatCents(additionalChargeCents)
		first := promotionSnapshots[0]
		promotionID = sql.NullInt64{Int64: first.ID, Valid: true}
		bytes, err := json.Marshal(first)
		if err != nil {
			return orderInput{}, err
		}
		promotionSnapshotJSON = sql.NullString{String: string(bytes), Valid: true}
	}
	calc, err := calculateFinalAmount(planSnapshot.Price, req.DiscountAmount, additionalCharge)
	if err != nil {
		return orderInput{}, err
	}
	packageBytes, err := json.Marshal(pkgSnapshot)
	if err != nil {
		return orderInput{}, err
	}
	planBytes, err := json.Marshal(planSnapshot)
	if err != nil {
		return orderInput{}, err
	}
	salesID, supervisorID, err := r.findCurrentLeadPIC(ctx, tx, ownerID)
	if err != nil {
		return orderInput{}, err
	}
	return orderInput{
		OutletID:              nullableID(req.OutletID),
		SalesID:               salesID,
		SupervisorID:          supervisorID,
		PackageID:             sql.NullInt64{Int64: pkgSnapshot.ID, Valid: true},
		PlanID:                sql.NullInt64{Int64: planSnapshot.ID, Valid: true},
		PromotionID:           promotionID,
		PackageSnapshotJSON:   string(packageBytes),
		PlanSnapshotJSON:      string(planBytes),
		PromotionSnapshotJSON: promotionSnapshotJSON,
		PromotionSnapshots:    promotionSnapshots,
		TenureMonths:          planSnapshot.TenureMonths,
		DurationDays:          totalDurationDaysMulti(planSnapshot.DurationDays, promotionSnapshots),
		BasePrice:             calc.BasePrice,
		DiscountAmount:        calc.DiscountAmount,
		AdditionalCharge:      calc.AdditionalCharge,
		FinalAmount:           calc.FinalAmount,
		Currency:              planSnapshot.Currency,
	}, nil
}

func (r *Repository) findCurrentLeadPIC(ctx context.Context, tx *sql.Tx, ownerID int64) (sql.NullInt64, sql.NullInt64, error) {
	var salesID sql.NullInt64
	var supervisorID sql.NullInt64
	err := tx.QueryRowContext(ctx, `
SELECT
	CASE WHEN current_owner_role = 'SALES' THEN current_owner_user_id ELSE active_sales_id END AS sales_id,
	supervisor_id
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

type finalAmountCalculation struct {
	BasePrice        string
	DiscountAmount   string
	AdditionalCharge string
	FinalAmount      string
}

func calculateFinalAmount(basePrice, discountAmount, additionalCharge string) (finalAmountCalculation, error) {
	baseCents, err := parseMoneyToCents(basePrice)
	if err != nil {
		return finalAmountCalculation{}, err
	}
	discountCents, err := parseMoneyToCents(discountAmount)
	if err != nil {
		return finalAmountCalculation{}, err
	}
	additionalCents, err := parseMoneyToCents(additionalCharge)
	if err != nil {
		return finalAmountCalculation{}, err
	}
	finalCents := baseCents - discountCents + additionalCents
	if finalCents <= 0 {
		return finalAmountCalculation{}, ErrInvalidMoney
	}
	return finalAmountCalculation{BasePrice: formatCents(baseCents), DiscountAmount: formatCents(discountCents), AdditionalCharge: formatCents(additionalCents), FinalAmount: formatCents(finalCents)}, nil
}

func (r *Repository) insertOrder(ctx context.Context, tx *sql.Tx, actor identity.User, code string, key string, ownerID, walletID int64, input orderInput, req CreateOrderRequest, purchasedAt time.Time, startDate time.Time, balanceShortfallAmount sql.NullString) (int64, error) {
	result, err := tx.ExecContext(ctx, `
INSERT INTO subscription_orders
(code, owner_id, outlet_id, closing_id, sales_id, supervisor_id, wallet_account_id, package_id, plan_id, promotion_id,
 package_snapshot_json, plan_snapshot_json, promotion_snapshot_json, tenure_months, duration_days,
 base_price, discount_amount, additional_charge, final_amount, balance_shortfall_amount, currency, status, idempotency_key, external_reference,
 purchased_at, subscription_start_date, note, created_by_user_id, updated_by_user_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		code, ownerID, input.OutletID, input.ClosingID, input.SalesID, input.SupervisorID, walletID,
		input.PackageID, input.PlanID, input.PromotionID, input.PackageSnapshotJSON, input.PlanSnapshotJSON,
		input.PromotionSnapshotJSON, input.TenureMonths, input.DurationDays, input.BasePrice, input.DiscountAmount,
		input.AdditionalCharge, input.FinalAmount, balanceShortfallAmount, input.Currency, OrderStatusPaid, key, nullableString(req.ExternalReference),
		purchasedAt, startDate, nullableString(req.Note), actor.ID, actor.ID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

type ledgerInput struct {
	Wallet            walletAccount
	OrderCode         string
	AmountCents       int64
	BalanceBefore     int64
	BalanceAfter      int64
	ExternalReference string
	IdempotencyKey    string
	OccurredAt        time.Time
	Note              string
	ActorID           int64
}

func (r *Repository) insertLedgerTransaction(ctx context.Context, tx *sql.Tx, input ledgerInput) (int64, error) {
	result, err := tx.ExecContext(ctx, `
INSERT INTO wallet_transactions
(code, wallet_account_id, owner_id, transaction_type, direction, amount, balance_before, balance_after,
 currency, source_type, source_reference, external_reference, idempotency_key, occurred_at, note, created_by_user_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextCode("WTX", time.Now().UTC(), input.Wallet.ID), input.Wallet.ID, input.Wallet.OwnerID,
		WalletTransactionDebit, WalletDirectionDebit, formatCents(input.AmountCents), formatCents(input.BalanceBefore),
		formatCents(input.BalanceAfter), input.Wallet.Currency, WalletSourceSubscriptionOrder, input.OrderCode,
		nullableString(input.ExternalReference), input.IdempotencyKey, input.OccurredAt, nullableString(input.Note), input.ActorID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) activateSubscription(ctx context.Context, tx *sql.Tx, orderID int64, ownerID int64, input orderInput, startDate time.Time) (int64, int64, error) {
	endDate := startDate.AddDate(0, 0, input.DurationDays)
	result, err := tx.ExecContext(ctx, `
INSERT INTO subscriptions
(code, owner_id, outlet_id, order_id, package_id, plan_id, status, active_from, active_until, total_duration_days, source_type)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextCode("SUB", time.Now().UTC(), orderID), ownerID, input.OutletID, orderID, input.PackageID, input.PlanID,
		SubscriptionStatusActive, startDate, endDate, input.DurationDays, WalletSourceSubscriptionOrder,
	)
	if err != nil {
		return 0, 0, err
	}
	subscriptionID, err := result.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	periodResult, err := tx.ExecContext(ctx, `
INSERT INTO subscription_periods
(subscription_id, order_id, owner_id, period_index, start_date, end_date, duration_days, package_id, plan_id, promotion_id, status)
VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?)`,
		subscriptionID, orderID, ownerID, startDate, endDate, input.DurationDays, input.PackageID, input.PlanID,
		input.PromotionID, SubscriptionStatusActive,
	)
	if err != nil {
		return 0, 0, err
	}
	periodID, err := periodResult.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	return subscriptionID, periodID, nil
}

func (r *Repository) confirmOrderAndClosing(ctx context.Context, tx *sql.Tx, actorID, orderID, closingID int64) error {
	return r.confirmOrderAndClosingWithAmount(ctx, tx, actorID, orderID, closingID, "")
}

// confirmOrderAndClosingWithAmount is confirmOrderAndClosing plus, when adminFinalAmount is
// non-empty (PARTIAL_CONFIRM), overriding sales_closings.final_amount to the admin dashboard's
// real number — this is what actually flows into partner commission calculation
// (partner.SyncCommissions reads sc.final_amount directly), not just an audit-trail note.
func (r *Repository) confirmOrderAndClosingWithAmount(ctx context.Context, tx *sql.Tx, actorID, orderID, closingID int64, adminFinalAmount string) error {
	if _, err := tx.ExecContext(ctx, `
UPDATE subscription_orders
SET status = ?, updated_by_user_id = ?
WHERE id = ? AND deleted_at IS NULL`, OrderStatusReconciled, actorID, orderID); err != nil {
		return err
	}
	if adminFinalAmount != "" {
		if _, err := tx.ExecContext(ctx, `
UPDATE sales_closings
SET final_amount = ?, status = ?, confirmed_at = COALESCE(confirmed_at, ?), rejected_at = NULL, rejection_reason = NULL, updated_by_user_id = ?
WHERE id = ? AND deleted_at IS NULL AND status <> ?`,
			adminFinalAmount, closingStatusConfirmed, time.Now().UTC(), actorID, closingID, closingStatusRejected,
		); err != nil {
			return err
		}
		return nil
	}
	_, err := tx.ExecContext(ctx, `
UPDATE sales_closings
SET status = ?, confirmed_at = COALESCE(confirmed_at, ?), rejected_at = NULL, rejection_reason = NULL, updated_by_user_id = ?
WHERE id = ? AND deleted_at IS NULL AND status <> ?`,
		closingStatusConfirmed, time.Now().UTC(), actorID, closingID, closingStatusRejected,
	)
	return err
}

type reconciliationInput struct {
	OrderID           sql.NullInt64
	ClosingID         sql.NullInt64
	OwnerID           sql.NullInt64
	Status            string
	MatchType         string
	IssueCode         sql.NullString
	AmountDifference  string
	AdminTenureMonths sql.NullInt64
	AdminFinalAmount  sql.NullString
	Note              sql.NullString
	Reason            sql.NullString
	ConfirmedAt       sql.NullTime
	RejectedAt        sql.NullTime
	ActorID           int64
}

func (r *Repository) insertReconciliation(ctx context.Context, tx *sql.Tx, input reconciliationInput) (int64, error) {
	result, err := tx.ExecContext(ctx, `
INSERT INTO subscription_reconciliations
(code, order_id, closing_id, owner_id, status, match_type, issue_code, amount_difference,
 admin_tenure_months, admin_final_amount, note, reason,
 confirmed_at, rejected_at, created_by_user_id, updated_by_user_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextCode("REC", time.Now().UTC(), input.OrderID.Int64), input.OrderID, input.ClosingID, input.OwnerID,
		input.Status, input.MatchType, input.IssueCode, input.AmountDifference,
		input.AdminTenureMonths, input.AdminFinalAmount, input.Note, input.Reason,
		input.ConfirmedAt, input.RejectedAt, input.ActorID, input.ActorID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) upsertReconciliation(ctx context.Context, tx *sql.Tx, input reconciliationInput) (int64, error) {
	if input.ClosingID.Valid {
		if err := r.ensureClosingNotReconciledByOtherOrder(ctx, tx, input.ClosingID.Int64, input.OrderID.Int64); err != nil {
			return 0, err
		}
	}
	existing, found, err := r.findReconciliationByOrderIDForUpdate(ctx, tx, input.OrderID.Int64)
	if err != nil {
		return 0, err
	}
	if !found {
		return r.insertReconciliation(ctx, tx, input)
	}
	if existing.Status == ReconciliationStatusConfirmed || existing.Status == ReconciliationStatusPartialConfirm {
		return 0, ErrOrderAlreadyReconciled
	}
	_, err = tx.ExecContext(ctx, `
UPDATE subscription_reconciliations
SET closing_id = ?, owner_id = ?, status = ?, match_type = ?, issue_code = ?, amount_difference = ?,
admin_tenure_months = ?, admin_final_amount = ?,
note = ?, reason = ?, confirmed_at = ?, rejected_at = ?, updated_by_user_id = ?
WHERE id = ? AND deleted_at IS NULL`,
		input.ClosingID, input.OwnerID, input.Status, input.MatchType, input.IssueCode, input.AmountDifference,
		input.AdminTenureMonths, input.AdminFinalAmount,
		input.Note, input.Reason, input.ConfirmedAt, input.RejectedAt, input.ActorID, existing.ID,
	)
	if err != nil {
		return 0, err
	}
	return existing.ID, nil
}

type issueInput struct {
	OrderID     sql.NullInt64
	ClosingID   sql.NullInt64
	OwnerID     sql.NullInt64
	IssueType   string
	Description sql.NullString
	DetectedAt  time.Time
	ActorID     int64
}

func (r *Repository) insertIssue(ctx context.Context, tx *sql.Tx, input issueInput) (int64, error) {
	result, err := tx.ExecContext(ctx, `
INSERT INTO reconciliation_issues
(code, order_id, closing_id, owner_id, issue_type, status, description, detected_at, created_by_user_id, updated_by_user_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextCode("RIS", time.Now().UTC(), input.OrderID.Int64+input.ClosingID.Int64), input.OrderID, input.ClosingID,
		input.OwnerID, input.IssueType, IssueStatusOpen, input.Description, input.DetectedAt, input.ActorID, input.ActorID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) resolveIssues(ctx context.Context, tx *sql.Tx, actorID, orderID, closingID int64, resolvedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
UPDATE reconciliation_issues
SET status = ?, resolved_at = ?, updated_by_user_id = ?
WHERE deleted_at IS NULL AND status = ? AND (order_id = ? OR closing_id = ?)`,
		IssueStatusResolved, resolvedAt, actorID, IssueStatusOpen, orderID, closingID,
	)
	return err
}

func (r *Repository) orderResultByIDs(ctx context.Context, q queryExecutor, orderID, subscriptionID, periodID, reconciliationID, issueID int64, idempotent bool) (OrderResult, error) {
	order, err := r.findOrderByIDRaw(ctx, q, orderID)
	if err != nil {
		return OrderResult{}, err
	}
	subscription, err := r.findSubscriptionByIDRaw(ctx, q, subscriptionID)
	if err != nil {
		return OrderResult{}, err
	}
	period, err := r.findPeriodByIDRaw(ctx, q, periodID)
	if err != nil {
		return OrderResult{}, err
	}
	var reconciliation *Reconciliation
	if reconciliationID > 0 {
		item, err := r.findReconciliationByIDRaw(ctx, q, reconciliationID)
		if err != nil {
			return OrderResult{}, err
		}
		reconciliation = &item
	}
	var issue *ReconciliationIssue
	if issueID > 0 {
		item, err := r.findIssueByIDRaw(ctx, q, issueID)
		if err != nil {
			return OrderResult{}, err
		}
		issue = &item
	}
	return OrderResult{Order: order, Subscription: subscription, Period: period, Reconciliation: reconciliation, Issue: issue, Idempotent: idempotent}, nil
}

func (r *Repository) orderResultFromOrder(ctx context.Context, q queryExecutor, order SubscriptionOrder, idempotent bool) (OrderResult, error) {
	subscription, err := r.findSubscriptionByOrderID(ctx, q, order.ID)
	if err != nil {
		return OrderResult{}, err
	}
	period, err := r.findPeriodByOrderID(ctx, q, order.ID)
	if err != nil {
		return OrderResult{}, err
	}
	reconciliation, _, err := r.findReconciliationByOrderID(ctx, q, order.ID)
	if err != nil {
		return OrderResult{}, err
	}
	issue, _, err := r.findOpenIssueByOrderID(ctx, q, order.ID)
	if err != nil {
		return OrderResult{}, err
	}
	return OrderResult{Order: order, Subscription: subscription, Period: period, Reconciliation: reconciliation, Issue: issue, Idempotent: idempotent}, nil
}

func (r *Repository) ensureOwnerExists(ctx context.Context, q queryExecutor, ownerID int64) error {
	var exists int
	err := q.QueryRowContext(ctx, "SELECT 1 FROM owners WHERE id = ? AND deleted_at IS NULL LIMIT 1", ownerID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrOwnerNotFound
	}
	return err
}

func (r *Repository) ensureOutletBelongsToOwner(ctx context.Context, q queryExecutor, ownerID, outletID int64) error {
	var exists int
	err := q.QueryRowContext(ctx, "SELECT 1 FROM outlets WHERE id = ? AND owner_id = ? AND deleted_at IS NULL LIMIT 1", outletID, ownerID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrInvalidRequest
	}
	return err
}

func (r *Repository) lockOrCreateWallet(ctx context.Context, tx *sql.Tx, ownerID int64) (walletAccount, error) {
	_, err := tx.ExecContext(ctx, `
INSERT INTO wallet_accounts (owner_id, account_code, currency, balance, status)
VALUES (?, ?, 'IDR', 0, 'ACTIVE')
ON DUPLICATE KEY UPDATE deleted_at = NULL, status = 'ACTIVE'`, ownerID, fmt.Sprintf("WALLET-OWNER-%06d", ownerID))
	if err != nil {
		return walletAccount{}, err
	}
	return r.lockWalletByOwner(ctx, tx, ownerID)
}

func (r *Repository) lockWalletByOwner(ctx context.Context, tx *sql.Tx, ownerID int64) (walletAccount, error) {
	var item walletAccount
	err := tx.QueryRowContext(ctx, `
SELECT id, owner_id, currency, CAST(balance AS CHAR)
FROM wallet_accounts
WHERE owner_id = ? AND deleted_at IS NULL
FOR UPDATE`, ownerID).Scan(&item.ID, &item.OwnerID, &item.Currency, &item.Balance)
	if err == sql.ErrNoRows {
		return walletAccount{}, ErrWalletNotFound
	}
	return item, err
}

func (r *Repository) updateWalletBalance(ctx context.Context, tx *sql.Tx, walletID int64, balanceAfterCents int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE wallet_accounts SET balance = ? WHERE id = ? AND deleted_at IS NULL`, formatCents(balanceAfterCents), walletID)
	return err
}

func (r *Repository) lockClosing(ctx context.Context, tx *sql.Tx, id int64) (closingSnapshot, error) {
	var item closingSnapshot
	err := tx.QueryRowContext(ctx, `
SELECT id, code, owner_id, outlet_id, sales_id, supervisor_id, package_id, plan_id, promotion_id,
CAST(package_snapshot_json AS CHAR), CAST(plan_snapshot_json AS CHAR), CAST(promotion_snapshot_json AS CHAR),
tenure_months, duration_days, CAST(base_price AS CHAR), CAST(discount_amount AS CHAR),
CAST(additional_charge AS CHAR), CAST(final_amount AS CHAR), currency, status
FROM sales_closings
WHERE id = ? AND deleted_at IS NULL
FOR UPDATE`, id).Scan(
		&item.ID, &item.Code, &item.OwnerID, &item.OutletID, &item.SalesID, &item.SupervisorID, &item.PackageID,
		&item.PlanID, &item.PromotionID, &item.PackageSnapshotJSON, &item.PlanSnapshotJSON, &item.PromotionSnapshotJSON,
		&item.TenureMonths, &item.DurationDays, &item.BasePrice, &item.DiscountAmount, &item.AdditionalCharge,
		&item.FinalAmount, &item.Currency, &item.Status,
	)
	if err == sql.ErrNoRows {
		return closingSnapshot{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) lockOrder(ctx context.Context, tx *sql.Tx, id int64) (SubscriptionOrder, error) {
	item, err := scanOrder(tx.QueryRowContext(ctx, orderSelect()+`
WHERE so.id = ? AND so.deleted_at IS NULL
FOR UPDATE`, id))
	if err == sql.ErrNoRows {
		return SubscriptionOrder{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ensureClosingNotReconciledByOtherOrder(ctx context.Context, q queryExecutor, closingID, orderID int64) error {
	var existingOrderID sql.NullInt64
	err := q.QueryRowContext(ctx, `
SELECT order_id FROM subscription_reconciliations
WHERE closing_id = ? AND deleted_at IS NULL AND status = ?
LIMIT 1`, closingID, ReconciliationStatusConfirmed).Scan(&existingOrderID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if existingOrderID.Valid && existingOrderID.Int64 != orderID {
		return ErrOrderAlreadyReconciled
	}
	return nil
}

func (r *Repository) findPlanSnapshot(ctx context.Context, tx *sql.Tx, planID int64, asOf time.Time) (PackageSnapshot, PlanSnapshot, error) {
	var pkg PackageSnapshot
	var plan PlanSnapshot
	var effectiveFrom time.Time
	var effectiveTo sql.NullTime
	err := tx.QueryRowContext(ctx, `
SELECT spl.id, spl.code, spl.name, spl.tenure_months, spl.duration_days, CAST(spl.price AS CHAR), spl.currency,
spl.effective_from, spl.effective_to, sp.id, sp.code, sp.name, sp.level_order
FROM subscription_plans spl
JOIN subscription_packages sp ON sp.id = spl.package_id
WHERE spl.id = ? AND spl.deleted_at IS NULL AND sp.deleted_at IS NULL AND spl.active = TRUE AND sp.active = TRUE
AND spl.effective_from <= DATE(?) AND (spl.effective_to IS NULL OR spl.effective_to >= DATE(?))
LIMIT 1`, planID, asOf, asOf).Scan(
		&plan.ID, &plan.Code, &plan.Name, &plan.TenureMonths, &plan.DurationDays, &plan.Price, &plan.Currency,
		&effectiveFrom, &effectiveTo, &pkg.ID, &pkg.Code, &pkg.Name, &pkg.LevelOrder,
	)
	if err == sql.ErrNoRows {
		return PackageSnapshot{}, PlanSnapshot{}, ErrNotFound
	}
	if err != nil {
		return PackageSnapshot{}, PlanSnapshot{}, err
	}
	plan.EffectiveFrom = effectiveFrom.Format("2006-01-02")
	if effectiveTo.Valid {
		plan.EffectiveTo = effectiveTo.Time.Format("2006-01-02")
	}
	return pkg, plan, nil
}

// ListOrderPromotions returns every promotion stacked onto orderID (Sprint 15a §4b) — the
// authoritative list; the order row's own promotion_id/promotion_snapshot_json only ever holds
// the first one, kept for backward compat.
func (r *Repository) ListOrderPromotions(ctx context.Context, orderID int64) ([]EntityRef, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.code, p.name
		FROM subscription_order_promotions sop
		JOIN promotions p ON p.id = sop.promotion_id
		WHERE sop.order_id = ?
		ORDER BY sop.id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EntityRef{}
	for rows.Next() {
		var item EntityRef
		if err := rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// findClosingPromotionSnapshots reads the full stacked promotion list a closing carries
// (sales_closing_promotions, populated by internal/closing's own multi-promotion support) so an
// order derived from that closing (req.ClosingID path) can record the same list in
// subscription_order_promotions for traceability — the discount arithmetic itself needs no
// recomputation here since closing.AdditionalCharge/FinalAmount are already the correct sum.
func (r *Repository) findClosingPromotionSnapshots(ctx context.Context, tx *sql.Tx, closingID int64) ([]PromotionSnapshot, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT CAST(promotion_snapshot_json AS CHAR) FROM sales_closing_promotions
		WHERE closing_id = ? ORDER BY id`, closingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snapshots []PromotionSnapshot
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var snapshot PromotionSnapshot
		if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *Repository) findEligiblePromotionSnapshot(ctx context.Context, tx *sql.Tx, promotionID, planID int64, asOf time.Time) (PromotionSnapshot, error) {
	var promo PromotionSnapshot
	var effectiveFrom time.Time
	var effectiveTo sql.NullTime
	err := tx.QueryRowContext(ctx, `
SELECT p.id, p.code, p.name, p.promotion_type, p.charge_type, CAST(p.additional_charge AS CHAR), p.priority, p.effective_from, p.effective_to
FROM promotions p
JOIN promotion_plan_eligibilities ppe ON ppe.promotion_id = p.id
WHERE p.id = ? AND ppe.plan_id = ? AND p.deleted_at IS NULL AND p.active = TRUE
AND p.effective_from <= DATE(?) AND (p.effective_to IS NULL OR p.effective_to >= DATE(?))
LIMIT 1`, promotionID, planID, asOf, asOf).Scan(&promo.ID, &promo.Code, &promo.Name, &promo.PromotionType, &promo.ChargeType, &promo.AdditionalCharge, &promo.Priority, &effectiveFrom, &effectiveTo)
	if err == sql.ErrNoRows {
		return PromotionSnapshot{}, ErrInvalidPromotion
	}
	if err != nil {
		return PromotionSnapshot{}, err
	}
	promo.EffectiveFrom = effectiveFrom.Format("2006-01-02")
	if effectiveTo.Valid {
		promo.EffectiveTo = effectiveTo.Time.Format("2006-01-02")
	}
	benefits, err := r.findPromotionBenefitSnapshots(ctx, tx, promotionID)
	if err != nil {
		return PromotionSnapshot{}, err
	}
	promo.Benefits = benefits
	return promo, nil
}

func (r *Repository) findPromotionBenefitSnapshots(ctx context.Context, tx *sql.Tx, promotionID int64) ([]BenefitSnapshot, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT pb.id, pb.benefit_type, pb.package_id, sp.code, sp.name, pb.duration_days, pb.quantity, pb.description, CAST(pb.metadata_json AS CHAR)
FROM promotion_benefits pb
LEFT JOIN subscription_packages sp ON sp.id = pb.package_id
WHERE pb.promotion_id = ?
ORDER BY pb.id ASC`, promotionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BenefitSnapshot{}
	for rows.Next() {
		var item BenefitSnapshot
		var packageID sql.NullInt64
		var packageCode, packageName, description, metadata sql.NullString
		var durationDays, quantity sql.NullInt64
		if err := rows.Scan(&item.ID, &item.BenefitType, &packageID, &packageCode, &packageName, &durationDays, &quantity, &description, &metadata); err != nil {
			return nil, err
		}
		if packageID.Valid {
			value := packageID.Int64
			item.PackageID = &value
			item.PackageCode = packageCode.String
			item.PackageName = packageName.String
		}
		if durationDays.Valid {
			value := durationDays.Int64
			item.DurationDays = &value
		}
		if quantity.Valid {
			value := quantity.Int64
			item.Quantity = &value
		}
		item.Description = description.String
		if metadata.Valid && metadata.String != "" {
			var parsed any
			if err := json.Unmarshal([]byte(metadata.String), &parsed); err == nil {
				item.MetadataJSON = parsed
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) findOrderByIdempotency(ctx context.Context, q queryExecutor, key string) (SubscriptionOrder, bool, error) {
	item, err := scanOrder(q.QueryRowContext(ctx, orderSelect()+` WHERE so.idempotency_key = ? AND so.deleted_at IS NULL LIMIT 1`, key))
	if err == sql.ErrNoRows {
		return SubscriptionOrder{}, false, nil
	}
	if err != nil {
		return SubscriptionOrder{}, false, err
	}
	return item, true, nil
}

func (r *Repository) findOrderByIDRaw(ctx context.Context, q queryExecutor, id int64) (SubscriptionOrder, error) {
	item, err := scanOrder(q.QueryRowContext(ctx, orderSelect()+` WHERE so.id = ? AND so.deleted_at IS NULL LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return SubscriptionOrder{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findSubscriptionByIDRaw(ctx context.Context, q queryExecutor, id int64) (Subscription, error) {
	item, err := scanSubscription(q.QueryRowContext(ctx, subscriptionSelect()+` WHERE s.id = ? AND s.deleted_at IS NULL LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return Subscription{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findSubscriptionByOrderID(ctx context.Context, q queryExecutor, orderID int64) (Subscription, error) {
	item, err := scanSubscription(q.QueryRowContext(ctx, subscriptionSelect()+` WHERE s.order_id = ? AND s.deleted_at IS NULL LIMIT 1`, orderID))
	if err == sql.ErrNoRows {
		return Subscription{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findPeriodByIDRaw(ctx context.Context, q queryExecutor, id int64) (SubscriptionPeriod, error) {
	item, err := scanPeriod(q.QueryRowContext(ctx, periodSelect()+` WHERE spd.id = ? AND spd.deleted_at IS NULL LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return SubscriptionPeriod{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findPeriodByOrderID(ctx context.Context, q queryExecutor, orderID int64) (SubscriptionPeriod, error) {
	item, err := scanPeriod(q.QueryRowContext(ctx, periodSelect()+` WHERE spd.order_id = ? AND spd.deleted_at IS NULL ORDER BY spd.period_index ASC LIMIT 1`, orderID))
	if err == sql.ErrNoRows {
		return SubscriptionPeriod{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findReconciliationByIDRaw(ctx context.Context, q queryExecutor, id int64) (Reconciliation, error) {
	item, err := scanReconciliation(q.QueryRowContext(ctx, reconciliationSelect()+` WHERE sr.id = ? AND sr.deleted_at IS NULL LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return Reconciliation{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findReconciliationByOrderID(ctx context.Context, q queryExecutor, orderID int64) (*Reconciliation, bool, error) {
	item, err := scanReconciliation(q.QueryRowContext(ctx, reconciliationSelect()+` WHERE sr.order_id = ? AND sr.deleted_at IS NULL LIMIT 1`, orderID))
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &item, true, nil
}

func (r *Repository) findReconciliationByOrderIDForUpdate(ctx context.Context, tx *sql.Tx, orderID int64) (Reconciliation, bool, error) {
	item, err := scanReconciliation(tx.QueryRowContext(ctx, reconciliationSelect()+` WHERE sr.order_id = ? AND sr.deleted_at IS NULL LIMIT 1 FOR UPDATE`, orderID))
	if err == sql.ErrNoRows {
		return Reconciliation{}, false, nil
	}
	if err != nil {
		return Reconciliation{}, false, err
	}
	return item, true, nil
}

func (r *Repository) findIssueByIDRaw(ctx context.Context, q queryExecutor, id int64) (ReconciliationIssue, error) {
	item, err := scanIssue(q.QueryRowContext(ctx, issueSelect()+` WHERE ri.id = ? AND ri.deleted_at IS NULL LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return ReconciliationIssue{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findOpenIssueByOrderID(ctx context.Context, q queryExecutor, orderID int64) (*ReconciliationIssue, bool, error) {
	item, err := scanIssue(q.QueryRowContext(ctx, issueSelect()+` WHERE ri.order_id = ? AND ri.deleted_at IS NULL AND ri.status = ? ORDER BY ri.id DESC LIMIT 1`, orderID, IssueStatusOpen))
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &item, true, nil
}

func orderSelect() string {
	return `
SELECT
so.id, so.code, so.owner_id, o.code, o.name, so.outlet_id, so.closing_id, sc.code,
so.sales_id, su.name, so.supervisor_id, spvu.name, so.wallet_account_id, so.wallet_transaction_id,
so.package_id, sp.code, sp.name, so.plan_id, spl.code, spl.name, so.promotion_id, p.code, p.name,
CAST(so.package_snapshot_json AS CHAR), CAST(so.plan_snapshot_json AS CHAR), CAST(so.promotion_snapshot_json AS CHAR),
so.tenure_months, so.duration_days, CAST(so.base_price AS CHAR), CAST(so.discount_amount AS CHAR),
CAST(so.additional_charge AS CHAR), CAST(so.final_amount AS CHAR), CAST(so.balance_shortfall_amount AS CHAR),
so.currency, so.status, so.idempotency_key,
so.external_reference, so.purchased_at, so.subscription_start_date, so.note,
so.created_by_user_id, cbu.name, so.updated_by_user_id, ubu.name, so.created_at, so.updated_at
FROM subscription_orders so
LEFT JOIN owners o ON o.id = so.owner_id AND o.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.id = so.closing_id AND sc.deleted_at IS NULL
LEFT JOIN users su ON su.id = so.sales_id
LEFT JOIN users spvu ON spvu.id = so.supervisor_id
LEFT JOIN subscription_packages sp ON sp.id = so.package_id
LEFT JOIN subscription_plans spl ON spl.id = so.plan_id
LEFT JOIN promotions p ON p.id = so.promotion_id
LEFT JOIN users cbu ON cbu.id = so.created_by_user_id
LEFT JOIN users ubu ON ubu.id = so.updated_by_user_id`
}

func subscriptionSelect() string {
	return `
SELECT s.id, s.code, s.owner_id, o.code, o.name, s.outlet_id, s.order_id, so.code,
s.package_id, sp.code, sp.name, s.plan_id, spl.code, spl.name, s.status,
s.active_from, s.active_until, s.total_duration_days, s.source_type, s.created_at, s.updated_at
FROM subscriptions s
LEFT JOIN owners o ON o.id = s.owner_id AND o.deleted_at IS NULL
LEFT JOIN subscription_orders so ON so.id = s.order_id AND so.deleted_at IS NULL
LEFT JOIN subscription_packages sp ON sp.id = s.package_id
LEFT JOIN subscription_plans spl ON spl.id = s.plan_id`
}

func periodSelect() string {
	return `
SELECT spd.id, spd.subscription_id, spd.order_id, spd.owner_id, spd.period_index, spd.start_date,
spd.end_date, spd.duration_days, spd.package_id, spd.plan_id, spd.promotion_id, spd.status,
spd.created_at, spd.updated_at
FROM subscription_periods spd`
}

func reconciliationSelect() string {
	return `
SELECT sr.id, sr.code, sr.order_id, so.code, sr.closing_id, sc.code, sr.owner_id, o.code, o.name,
sr.status, sr.match_type, sr.issue_code, CAST(sr.amount_difference AS CHAR),
sr.admin_tenure_months, CAST(sr.admin_final_amount AS CHAR), sr.note, sr.reason,
sr.confirmed_at, sr.rejected_at, sr.created_by_user_id, cbu.name, sr.updated_by_user_id, ubu.name,
sr.created_at, sr.updated_at
FROM subscription_reconciliations sr
LEFT JOIN subscription_orders so ON so.id = sr.order_id AND so.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.id = sr.closing_id AND sc.deleted_at IS NULL
LEFT JOIN owners o ON o.id = sr.owner_id AND o.deleted_at IS NULL
LEFT JOIN users cbu ON cbu.id = sr.created_by_user_id
LEFT JOIN users ubu ON ubu.id = sr.updated_by_user_id`
}

func issueSelect() string {
	return `
SELECT ri.id, ri.code, ri.order_id, so.code, ri.closing_id, sc.code, ri.owner_id, o.code, o.name,
ri.issue_type, ri.status, ri.description, ri.detected_at, ri.resolved_at,
ri.created_by_user_id, cbu.name, ri.updated_by_user_id, ubu.name, ri.created_at, ri.updated_at
FROM reconciliation_issues ri
LEFT JOIN subscription_orders so ON so.id = ri.order_id AND so.deleted_at IS NULL
LEFT JOIN sales_closings sc ON sc.id = ri.closing_id AND sc.deleted_at IS NULL
LEFT JOIN owners o ON o.id = ri.owner_id AND o.deleted_at IS NULL
LEFT JOIN users cbu ON cbu.id = ri.created_by_user_id
LEFT JOIN users ubu ON ubu.id = ri.updated_by_user_id`
}

func orderWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"so.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "so.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(so.code LIKE ? OR so.external_reference LIKE ? OR o.name LIKE ? OR sc.code LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if params.Status != "" {
		where = append(where, "so.status = ?")
		args = append(args, params.Status)
	}
	if params.OwnerID != nil {
		where = append(where, "so.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.ClosingID != nil {
		where = append(where, "so.closing_id = ?")
		args = append(args, *params.ClosingID)
	}
	if params.SalesID != nil {
		where = append(where, "so.sales_id = ?")
		args = append(args, *params.SalesID)
	}
	if params.SupervisorID != nil {
		where = append(where, "so.supervisor_id = ?")
		args = append(args, *params.SupervisorID)
	}
	if params.PlanID != nil {
		where = append(where, "so.plan_id = ?")
		args = append(args, *params.PlanID)
	}
	if params.PurchasedFrom != nil {
		where = append(where, "so.purchased_at >= ?")
		args = append(args, *params.PurchasedFrom)
	}
	if params.PurchasedTo != nil {
		where = append(where, "so.purchased_at <= ?")
		args = append(args, *params.PurchasedTo)
	}
	return strings.Join(where, " AND "), args
}

func subscriptionWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"s.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "s.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(s.code LIKE ? OR so.code LIKE ? OR o.name LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if params.Status != "" {
		where = append(where, "s.status = ?")
		args = append(args, params.Status)
	}
	if params.OwnerID != nil {
		where = append(where, "s.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.OutletID != nil {
		where = append(where, "s.outlet_id = ?")
		args = append(args, *params.OutletID)
	}
	if params.OrderID != nil {
		where = append(where, "s.order_id = ?")
		args = append(args, *params.OrderID)
	}
	if params.PlanID != nil {
		where = append(where, "s.plan_id = ?")
		args = append(args, *params.PlanID)
	}
	if params.ActiveFrom != nil {
		where = append(where, "s.active_from >= ?")
		args = append(args, *params.ActiveFrom)
	}
	if params.ActiveTo != nil {
		where = append(where, "s.active_until <= ?")
		args = append(args, *params.ActiveTo)
	}
	return strings.Join(where, " AND "), args
}

func reconciliationWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"sr.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "sr.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(sr.code LIKE ? OR so.code LIKE ? OR sc.code LIKE ? OR o.name LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if params.Status != "" {
		where = append(where, "sr.status = ?")
		args = append(args, params.Status)
	}
	if params.OwnerID != nil {
		where = append(where, "sr.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.OrderID != nil {
		where = append(where, "sr.order_id = ?")
		args = append(args, *params.OrderID)
	}
	if params.ClosingID != nil {
		where = append(where, "sr.closing_id = ?")
		args = append(args, *params.ClosingID)
	}
	return strings.Join(where, " AND "), args
}

func issueWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"ri.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "ri.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(ri.code LIKE ? OR so.code LIKE ? OR sc.code LIKE ? OR o.name LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if params.Status != "" {
		where = append(where, "ri.status = ?")
		args = append(args, params.Status)
	}
	if params.IssueType != "" {
		where = append(where, "ri.issue_type = ?")
		args = append(args, params.IssueType)
	}
	if params.OwnerID != nil {
		where = append(where, "ri.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.OrderID != nil {
		where = append(where, "ri.order_id = ?")
		args = append(args, *params.OrderID)
	}
	if params.ClosingID != nil {
		where = append(where, "ri.closing_id = ?")
		args = append(args, *params.ClosingID)
	}
	return strings.Join(where, " AND "), args
}

func ownerVisibilityWhere(actor identity.User, ownerColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "1 = 1", nil
	case RoleSupervisor:
		return `EXISTS (SELECT 1 FROM customer_leads cl WHERE cl.owner_id = ` + ownerColumn + ` AND cl.deleted_at IS NULL AND (cl.current_owner_user_id = ? OR cl.supervisor_id = ?))`, []any{actor.ID, actor.ID}
	case RoleSales:
		return `EXISTS (SELECT 1 FROM customer_leads cl WHERE cl.owner_id = ` + ownerColumn + ` AND cl.deleted_at IS NULL AND cl.current_owner_role = 'SALES' AND cl.current_owner_user_id = ?)`, []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func orderOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{"purchased_at": "so.purchased_at", "created_at": "so.created_at", "updated_at": "so.updated_at", "status": "so.status", "final_amount": "so.final_amount", "code": "so.code"}, "so.purchased_at DESC, so.id DESC")
}
func subscriptionOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{"active_from": "s.active_from", "active_until": "s.active_until", "created_at": "s.created_at", "status": "s.status", "code": "s.code"}, "s.active_from DESC, s.id DESC")
}
func reconciliationOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{"created_at": "sr.created_at", "updated_at": "sr.updated_at", "status": "sr.status", "code": "sr.code"}, "sr.created_at DESC, sr.id DESC")
}
func issueOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{"detected_at": "ri.detected_at", "created_at": "ri.created_at", "status": "ri.status", "issue_type": "ri.issue_type", "code": "ri.code"}, "ri.detected_at DESC, ri.id DESC")
}

func scanOrders(rows *sql.Rows, total int64) ([]SubscriptionOrder, int64, error) {
	items := []SubscriptionOrder{}
	for rows.Next() {
		item, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
func scanSubscriptions(rows *sql.Rows, total int64) ([]Subscription, int64, error) {
	items := []Subscription{}
	for rows.Next() {
		item, err := scanSubscription(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
func scanReconciliations(rows *sql.Rows, total int64) ([]Reconciliation, int64, error) {
	items := []Reconciliation{}
	for rows.Next() {
		item, err := scanReconciliation(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
func scanIssues(rows *sql.Rows, total int64) ([]ReconciliationIssue, int64, error) {
	items := []ReconciliationIssue{}
	for rows.Next() {
		item, err := scanIssue(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanOrder(row scanner) (SubscriptionOrder, error) {
	var item SubscriptionOrder
	err := row.Scan(&item.ID, &item.Code, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.OutletID, &item.ClosingID, &item.ClosingCode, &item.SalesID, &item.SalesName, &item.SupervisorID, &item.SupervisorName, &item.WalletAccountID, &item.WalletTransactionID, &item.PackageID, &item.PackageCode, &item.PackageName, &item.PlanID, &item.PlanCode, &item.PlanName, &item.PromotionID, &item.PromotionCode, &item.PromotionName, &item.PackageSnapshotJSON, &item.PlanSnapshotJSON, &item.PromotionSnapshotJSON, &item.TenureMonths, &item.DurationDays, &item.BasePrice, &item.DiscountAmount, &item.AdditionalCharge, &item.FinalAmount, &item.BalanceShortfallAmount, &item.Currency, &item.Status, &item.IdempotencyKey, &item.ExternalReference, &item.PurchasedAt, &item.SubscriptionStartDate, &item.Note, &item.CreatedByUserID, &item.CreatedByName, &item.UpdatedByUserID, &item.UpdatedByName, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanSubscription(row scanner) (Subscription, error) {
	var item Subscription
	err := row.Scan(&item.ID, &item.Code, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.OutletID, &item.OrderID, &item.OrderCode, &item.PackageID, &item.PackageCode, &item.PackageName, &item.PlanID, &item.PlanCode, &item.PlanName, &item.Status, &item.ActiveFrom, &item.ActiveUntil, &item.TotalDurationDays, &item.SourceType, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanPeriod(row scanner) (SubscriptionPeriod, error) {
	var item SubscriptionPeriod
	err := row.Scan(&item.ID, &item.SubscriptionID, &item.OrderID, &item.OwnerID, &item.PeriodIndex, &item.StartDate, &item.EndDate, &item.DurationDays, &item.PackageID, &item.PlanID, &item.PromotionID, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanReconciliation(row scanner) (Reconciliation, error) {
	var item Reconciliation
	err := row.Scan(&item.ID, &item.Code, &item.OrderID, &item.OrderCode, &item.ClosingID, &item.ClosingCode, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.Status, &item.MatchType, &item.IssueCode, &item.AmountDifference, &item.AdminTenureMonths, &item.AdminFinalAmount, &item.Note, &item.Reason, &item.ConfirmedAt, &item.RejectedAt, &item.CreatedByUserID, &item.CreatedByName, &item.UpdatedByUserID, &item.UpdatedByName, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanIssue(row scanner) (ReconciliationIssue, error) {
	var item ReconciliationIssue
	err := row.Scan(&item.ID, &item.Code, &item.OrderID, &item.OrderCode, &item.ClosingID, &item.ClosingCode, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.IssueType, &item.Status, &item.Description, &item.DetectedAt, &item.ResolvedAt, &item.CreatedByUserID, &item.CreatedByName, &item.UpdatedByUserID, &item.UpdatedByName, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func orderIdempotencyKey(req CreateOrderRequest) (string, error) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key != "" {
		return "subscription_order:" + key, nil
	}
	externalReference := strings.TrimSpace(req.ExternalReference)
	if externalReference == "" {
		return "", ErrIdempotencyRequired
	}
	return "subscription_order:external:" + externalReference, nil
}

func subscriptionStartDate(value string, purchasedAt time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return dateOnly(purchasedAt), nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return parsed, nil
}

func effectiveUpgradeStartDate(value string, purchasedAt time.Time) (time.Time, error) {
	startDate, err := subscriptionStartDate(value, purchasedAt)
	if err != nil {
		return time.Time{}, err
	}
	if startDate.After(dateOnly(purchasedAt)) {
		return time.Time{}, fmt.Errorf("%w: tanggal efektif upgrade (%s) tidak boleh lebih dari tanggal pembelian (%s)", ErrUpgradeNotAllowed, startDate.Format("2006-01-02"), purchasedAt.Format("2006-01-02"))
	}
	return startDate, nil
}
// totalDurationDaysMulti sums baseDays plus every stacked promotion's FREE_DURATION benefit (Sprint
// 15a §4b) — a single-promotion-JSON lookup would only ever see the FIRST promotion applied and
// silently miss a FREE_DURATION benefit carried by a second or third one.
func totalDurationDaysMulti(baseDays int, promotions []PromotionSnapshot) int {
	total := baseDays
	for _, promotion := range promotions {
		total += freeDurationDays(promotion)
	}
	return total
}

func freeDurationDays(promotion PromotionSnapshot) int {
	total := 0
	for _, benefit := range promotion.Benefits {
		if strings.EqualFold(benefit.BenefitType, benefitTypeFreeDuration) && benefit.DurationDays != nil {
			total += int(*benefit.DurationDays)
		}
	}
	return total
}

func moneyDifference(left, right string) (string, error) {
	leftCents, err := parseMoneyToCents(left)
	if err != nil {
		return "", err
	}
	rightCents, err := parseMoneyToCents(right)
	if err != nil {
		return "", err
	}
	return formatCents(leftCents - rightCents), nil
}

func sameNullableID(left, right sql.NullInt64) bool {
	return left.Valid && right.Valid && left.Int64 == right.Int64
}

func canManageOrder(actor identity.User, order SubscriptionOrder) bool {
	switch actor.RoleCode {
	case RoleAdmin:
		return true
	case RoleSupervisor:
		return order.SupervisorID.Valid && order.SupervisorID.Int64 == actor.ID
	default:
		return false
	}
}

func normalizeAction(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "CONFIRM", ReconciliationStatusConfirmed:
		return ReconciliationStatusConfirmed
	case "REJECT", ReconciliationStatusRejected:
		return ReconciliationStatusRejected
	case ReconciliationStatusPartialConfirm:
		return ReconciliationStatusPartialConfirm
	default:
		return ""
	}
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
func nullableID(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func orderBy(sort string, allowed map[string]string, fallback string) (string, error) {
	sort = strings.TrimSpace(sort)
	if sort == "" {
		return fallback, nil
	}
	direction := "ASC"
	if strings.HasPrefix(sort, "-") {
		direction = "DESC"
		sort = strings.TrimPrefix(sort, "-")
	}
	column, ok := allowed[sort]
	if !ok {
		return "", ErrInvalidSort
	}
	return column + " " + direction, nil
}

func like(value string) string { return "%" + strings.TrimSpace(value) + "%" }
func nextCode(prefix string, at time.Time, seed int64) string {
	return fmt.Sprintf("%s-%s-%06d-%06d", prefix, at.Format("20060102150405"), seed, at.Nanosecond()/1000)
}
func dateOnly(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
