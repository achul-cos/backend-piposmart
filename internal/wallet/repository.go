package wallet

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Repository struct {
	db *sql.DB
}

type queryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type TopupResult struct {
	Payment     WalletPayment
	Transaction WalletTransaction
	Wallet      WalletAccount
	Idempotent  bool
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListWallets(ctx context.Context, actor identity.User, params ListParams) ([]WalletAccount, int64, error) {
	where, args := walletWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM owners o
		LEFT JOIN wallet_accounts wa ON wa.owner_id = o.id AND wa.deleted_at IS NULL
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := walletOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := walletSelect() + `
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
	return scanWallets(rows, total)
}

func (r *Repository) FindWalletByOwner(ctx context.Context, actor identity.User, ownerID int64) (WalletAccount, error) {
	where, args := walletWhere(actor, ListParams{OwnerID: &ownerID})
	item, err := scanWallet(r.db.QueryRowContext(ctx, walletSelect()+`
		WHERE `+where+`
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return WalletAccount{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ListPayments(ctx context.Context, actor identity.User, params ListParams) ([]WalletPayment, int64, error) {
	where, args := paymentWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM wallet_payments wp
		LEFT JOIN owners o ON o.id = wp.owner_id AND o.deleted_at IS NULL
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := paymentOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := paymentSelect() + `
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
	return scanPayments(rows, total)
}

func (r *Repository) FindPaymentByID(ctx context.Context, actor identity.User, id int64) (WalletPayment, error) {
	where, args := paymentWhere(actor, ListParams{})
	args = append([]any{id}, args...)
	item, err := scanPayment(r.db.QueryRowContext(ctx, paymentSelect()+`
		WHERE wp.id = ? AND `+where+`
		LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return WalletPayment{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) ListTransactions(ctx context.Context, actor identity.User, params ListParams) ([]WalletTransaction, int64, error) {
	where, args := transactionWhere(actor, params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM wallet_transactions wt
		LEFT JOIN owners o ON o.id = wt.owner_id AND o.deleted_at IS NULL
		LEFT JOIN wallet_accounts wa ON wa.id = wt.wallet_account_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy, err := transactionOrderBy(params.Sort)
	if err != nil {
		return nil, 0, err
	}
	query := transactionSelect() + `
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
	return scanTransactions(rows, total)
}

func (r *Repository) CreateTopup(ctx context.Context, actor identity.User, ownerID int64, req CreateTopupRequest) (TopupResult, error) {
	key, err := topupIdempotencyKey(req)
	if err != nil {
		return TopupResult{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TopupResult{}, err
	}
	defer tx.Rollback()

	if existing, found, err := r.findPaymentByIdempotency(ctx, tx, key); err != nil {
		return TopupResult{}, err
	} else if found {
		transaction, err := r.findTransactionByPaymentID(ctx, tx, existing.ID)
		if err != nil {
			return TopupResult{}, err
		}
		wallet, err := r.findWalletByID(ctx, tx, existing.WalletAccountID.Int64)
		if err != nil {
			return TopupResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return TopupResult{}, err
		}
		return TopupResult{Payment: existing, Transaction: transaction, Wallet: wallet, Idempotent: true}, nil
	}

	if err := r.ensureOwnerExists(ctx, tx, ownerID); err != nil {
		return TopupResult{}, err
	}
	amountCents, err := parsePositiveMoneyToCents(req.Amount)
	if err != nil {
		return TopupResult{}, err
	}
	wallet, err := r.lockOrCreateWallet(ctx, tx, ownerID)
	if err != nil {
		return TopupResult{}, err
	}
	balanceBeforeCents, err := parseMoneyToCents(wallet.Balance)
	if err != nil {
		return TopupResult{}, err
	}
	balanceAfterCents, err := applyBalance(balanceBeforeCents, amountCents, DirectionCredit)
	if err != nil {
		return TopupResult{}, err
	}
	paidAt := time.Now().UTC()
	if req.PaidAt != nil {
		paidAt = req.PaidAt.UTC()
	}
	channel := normalizeCode(req.PaymentChannel)
	if channel == "" {
		channel = "MANUAL"
	}
	paymentCode := nextCode("PAY", time.Now().UTC(), ownerID)
	paymentResult, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_payments
			(code, owner_id, wallet_account_id, payment_type, payment_channel, external_reference,
			 idempotency_key, amount, currency, status, paid_at, note, created_by_user_id)
		VALUES (?, ?, ?, 'TOPUP', ?, ?, ?, ?, ?, 'PAID', ?, ?, ?)`,
		paymentCode,
		ownerID,
		wallet.ID,
		channel,
		nullableString(req.ExternalReference),
		key,
		formatCents(amountCents),
		wallet.Currency,
		paidAt,
		nullableString(req.Note),
		actor.ID,
	)
	if err != nil {
		return TopupResult{}, err
	}
	paymentID, err := paymentResult.LastInsertId()
	if err != nil {
		return TopupResult{}, err
	}
	transactionID, err := r.insertLedgerTransaction(ctx, tx, ledgerInput{
		Wallet:            wallet,
		PaymentID:         sql.NullInt64{Int64: paymentID, Valid: true},
		TransactionType:   TransactionTypeCredit,
		Direction:         DirectionCredit,
		AmountCents:       amountCents,
		BalanceBefore:     balanceBeforeCents,
		BalanceAfter:      balanceAfterCents,
		SourceType:        SourceTopup,
		SourceReference:   paymentCode,
		ExternalReference: req.ExternalReference,
		IdempotencyKey:    key,
		OccurredAt:        paidAt,
		Note:              req.Note,
		ActorID:           actor.ID,
	})
	if err != nil {
		return TopupResult{}, err
	}
	if err := r.updateWalletBalance(ctx, tx, wallet.ID, balanceAfterCents); err != nil {
		return TopupResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return TopupResult{}, err
	}
	payment, err := r.findPaymentByIDRaw(ctx, r.db, paymentID)
	if err != nil {
		return TopupResult{}, err
	}
	transaction, err := r.findTransactionByIDRaw(ctx, r.db, transactionID)
	if err != nil {
		return TopupResult{}, err
	}
	updatedWallet, err := r.findWalletByID(ctx, r.db, wallet.ID)
	if err != nil {
		return TopupResult{}, err
	}
	return TopupResult{Payment: payment, Transaction: transaction, Wallet: updatedWallet}, nil
}

func (r *Repository) CreateDebit(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest) (WalletTransaction, error) {
	return r.createManualLedger(ctx, actor, ownerID, req, TransactionTypeDebit, DirectionDebit, SourceManualDebit)
}

func (r *Repository) CreateAdjustment(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest) (WalletTransaction, error) {
	direction := normalizeDirection(req.Direction)
	if direction == "" {
		return WalletTransaction{}, ErrInvalidDirection
	}
	return r.createManualLedger(ctx, actor, ownerID, req, TransactionTypeAdjustment, direction, SourceAdjustment)
}

func (r *Repository) CreateRefund(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest) (WalletTransaction, error) {
	return r.createManualLedger(ctx, actor, ownerID, req, TransactionTypeRefund, DirectionDebit, SourceRefund)
}

func (r *Repository) createManualLedger(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest, transactionType, direction, sourceType string) (WalletTransaction, error) {
	key, err := transactionIdempotencyKey(sourceType, req)
	if err != nil {
		return WalletTransaction{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return WalletTransaction{}, err
	}
	defer tx.Rollback()
	if existing, found, err := r.findTransactionByIdempotency(ctx, tx, key); err != nil {
		return WalletTransaction{}, err
	} else if found {
		if err := tx.Commit(); err != nil {
			return WalletTransaction{}, err
		}
		return existing, nil
	}
	if err := r.ensureOwnerExists(ctx, tx, ownerID); err != nil {
		return WalletTransaction{}, err
	}
	amountCents, err := parsePositiveMoneyToCents(req.Amount)
	if err != nil {
		return WalletTransaction{}, err
	}
	wallet, err := r.lockOrCreateWallet(ctx, tx, ownerID)
	if err != nil {
		return WalletTransaction{}, err
	}
	balanceBeforeCents, err := parseMoneyToCents(wallet.Balance)
	if err != nil {
		return WalletTransaction{}, err
	}
	balanceAfterCents, err := applyBalance(balanceBeforeCents, amountCents, direction)
	if err != nil {
		return WalletTransaction{}, err
	}
	occurredAt := time.Now().UTC()
	if req.OccurredAt != nil {
		occurredAt = req.OccurredAt.UTC()
	}
	transactionID, err := r.insertLedgerTransaction(ctx, tx, ledgerInput{
		Wallet:            wallet,
		TransactionType:   transactionType,
		Direction:         direction,
		AmountCents:       amountCents,
		BalanceBefore:     balanceBeforeCents,
		BalanceAfter:      balanceAfterCents,
		SourceType:        sourceType,
		SourceReference:   req.SourceReference,
		ExternalReference: req.ExternalReference,
		IdempotencyKey:    key,
		OccurredAt:        occurredAt,
		Note:              req.Note,
		ActorID:           actor.ID,
	})
	if err != nil {
		return WalletTransaction{}, err
	}
	if err := r.updateWalletBalance(ctx, tx, wallet.ID, balanceAfterCents); err != nil {
		return WalletTransaction{}, err
	}
	if err := tx.Commit(); err != nil {
		return WalletTransaction{}, err
	}
	return r.findTransactionByIDRaw(ctx, r.db, transactionID)
}

type ledgerInput struct {
	Wallet            WalletAccount
	PaymentID         sql.NullInt64
	TransactionType   string
	Direction         string
	AmountCents       int64
	BalanceBefore     int64
	BalanceAfter      int64
	SourceType        string
	SourceReference   string
	ExternalReference string
	IdempotencyKey    string
	OccurredAt        time.Time
	Note              string
	ActorID           int64
}

func (r *Repository) insertLedgerTransaction(ctx context.Context, tx *sql.Tx, input ledgerInput) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions
			(code, wallet_account_id, owner_id, payment_id, transaction_type, direction, amount,
			 balance_before, balance_after, currency, source_type, source_reference, external_reference,
			 idempotency_key, occurred_at, note, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextCode("WTX", time.Now().UTC(), input.Wallet.ID),
		input.Wallet.ID,
		input.Wallet.OwnerID,
		input.PaymentID,
		input.TransactionType,
		input.Direction,
		formatCents(input.AmountCents),
		formatCents(input.BalanceBefore),
		formatCents(input.BalanceAfter),
		input.Wallet.Currency,
		input.SourceType,
		nullableString(input.SourceReference),
		nullableString(input.ExternalReference),
		input.IdempotencyKey,
		input.OccurredAt,
		nullableString(input.Note),
		input.ActorID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) updateWalletBalance(ctx context.Context, tx *sql.Tx, walletID int64, balanceAfterCents int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE wallet_accounts
		SET balance = ?
		WHERE id = ? AND deleted_at IS NULL`, formatCents(balanceAfterCents), walletID)
	return err
}

func (r *Repository) ensureOwnerExists(ctx context.Context, q queryExecutor, ownerID int64) error {
	var exists int
	err := q.QueryRowContext(ctx, "SELECT 1 FROM owners WHERE id = ? AND deleted_at IS NULL LIMIT 1", ownerID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrOwnerNotFound
	}
	return err
}

func (r *Repository) lockOrCreateWallet(ctx context.Context, tx *sql.Tx, ownerID int64) (WalletAccount, error) {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_accounts (owner_id, account_code, currency, balance, status)
		VALUES (?, ?, 'IDR', 0, 'ACTIVE')
		ON DUPLICATE KEY UPDATE deleted_at = NULL, status = 'ACTIVE'`,
		ownerID, fmt.Sprintf("WALLET-OWNER-%06d", ownerID),
	)
	if err != nil {
		return WalletAccount{}, err
	}
	return r.lockWalletByOwner(ctx, tx, ownerID)
}

func (r *Repository) lockWalletByOwner(ctx context.Context, tx *sql.Tx, ownerID int64) (WalletAccount, error) {
	item, err := scanWallet(tx.QueryRowContext(ctx, `
		SELECT wa.id, wa.owner_id, o.code, o.name, wa.account_code, wa.currency,
			CAST(wa.balance AS CHAR), CAST(wa.balance AS CHAR) AS ledger_balance,
			wa.status, wa.created_at, wa.updated_at
		FROM wallet_accounts wa
		LEFT JOIN owners o ON o.id = wa.owner_id AND o.deleted_at IS NULL
		WHERE wa.owner_id = ? AND wa.deleted_at IS NULL
		FOR UPDATE`, ownerID))
	if err == sql.ErrNoRows {
		return WalletAccount{}, ErrNotFound
	}
	return item, err
}

// findWalletByID has no GROUP BY, matching FindWalletByOwner/ListWallets below: the wtl subquery
// inside walletSelect() already aggregates to at most one row per wallet_account_id before the
// join, so the join itself can never fan out — an explicit GROUP BY here was redundant and, under
// MySQL 8's default ONLY_FULL_GROUP_BY sql_mode, actively broke this query (it can't prove
// wtl.ledger_balance, coming from an ungrouped-at-the-call-site derived table, is functionally
// dependent on wa.id, even though it provably is).
func (r *Repository) findWalletByID(ctx context.Context, q queryExecutor, id int64) (WalletAccount, error) {
	item, err := scanWallet(q.QueryRowContext(ctx, walletSelect()+`
		WHERE wa.id = ? AND wa.deleted_at IS NULL`, id))
	if err == sql.ErrNoRows {
		return WalletAccount{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findPaymentByIDRaw(ctx context.Context, q queryExecutor, id int64) (WalletPayment, error) {
	item, err := scanPayment(q.QueryRowContext(ctx, paymentSelect()+`
		WHERE wp.id = ? AND wp.deleted_at IS NULL
		LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return WalletPayment{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findPaymentByIdempotency(ctx context.Context, q queryExecutor, key string) (WalletPayment, bool, error) {
	item, err := scanPayment(q.QueryRowContext(ctx, paymentSelect()+`
		WHERE wp.idempotency_key = ? AND wp.deleted_at IS NULL
		LIMIT 1`, key))
	if err == sql.ErrNoRows {
		return WalletPayment{}, false, nil
	}
	if err != nil {
		return WalletPayment{}, false, err
	}
	return item, true, nil
}

func (r *Repository) findTransactionByIDRaw(ctx context.Context, q queryExecutor, id int64) (WalletTransaction, error) {
	item, err := scanTransaction(q.QueryRowContext(ctx, transactionSelect()+`
		WHERE wt.id = ? AND wt.deleted_at IS NULL
		LIMIT 1`, id))
	if err == sql.ErrNoRows {
		return WalletTransaction{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findTransactionByPaymentID(ctx context.Context, q queryExecutor, paymentID int64) (WalletTransaction, error) {
	item, err := scanTransaction(q.QueryRowContext(ctx, transactionSelect()+`
		WHERE wt.payment_id = ? AND wt.deleted_at IS NULL
		ORDER BY wt.id
		LIMIT 1`, paymentID))
	if err == sql.ErrNoRows {
		return WalletTransaction{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) findTransactionByIdempotency(ctx context.Context, q queryExecutor, key string) (WalletTransaction, bool, error) {
	item, err := scanTransaction(q.QueryRowContext(ctx, transactionSelect()+`
		WHERE wt.idempotency_key = ? AND wt.deleted_at IS NULL
		LIMIT 1`, key))
	if err == sql.ErrNoRows {
		return WalletTransaction{}, false, nil
	}
	if err != nil {
		return WalletTransaction{}, false, err
	}
	return item, true, nil
}

func walletSelect() string {
	return `
		SELECT
			COALESCE(wa.id, 0) AS wallet_id,
			o.id,
			o.code,
			o.name,
			COALESCE(wa.account_code, CONCAT('WALLET-OWNER-', LPAD(o.id, 6, '0'))) AS account_code,
			COALESCE(wa.currency, 'IDR') AS currency,
			COALESCE(CAST(wa.balance AS CHAR), '0.00') AS balance,
			COALESCE(CAST(wtl.ledger_balance AS CHAR), COALESCE(CAST(wa.balance AS CHAR), '0.00')) AS ledger_balance,
			COALESCE(wa.status, 'ACTIVE') AS status,
			COALESCE(wa.created_at, o.created_at) AS created_at,
			COALESCE(wa.updated_at, o.updated_at) AS updated_at
		FROM owners o
		LEFT JOIN wallet_accounts wa ON wa.owner_id = o.id AND wa.deleted_at IS NULL
		LEFT JOIN (
			SELECT
				wt.wallet_account_id,
				COALESCE(SUM(CASE WHEN wt.direction = 'CREDIT' THEN wt.amount ELSE -wt.amount END), 0) AS ledger_balance
			FROM wallet_transactions wt
			WHERE wt.deleted_at IS NULL
			GROUP BY wt.wallet_account_id
		) wtl ON wtl.wallet_account_id = wa.id`
}

func paymentSelect() string {
	return `
		SELECT wp.id, wp.code, wp.owner_id, o.code, o.name, wp.wallet_account_id,
			wp.payment_type, wp.payment_channel, wp.external_reference, wp.idempotency_key,
			CAST(wp.amount AS CHAR), wp.currency, wp.status, wp.paid_at, wp.note,
			wp.created_by_user_id, u.name, wp.created_at, wp.updated_at
		FROM wallet_payments wp
		LEFT JOIN owners o ON o.id = wp.owner_id AND o.deleted_at IS NULL
		LEFT JOIN users u ON u.id = wp.created_by_user_id`
}

func transactionSelect() string {
	return `
		SELECT wt.id, wt.code, wt.wallet_account_id, wt.owner_id, o.code, o.name, wt.payment_id,
			wt.transaction_type, wt.direction, CAST(wt.amount AS CHAR), CAST(wt.balance_before AS CHAR),
			CAST(wt.balance_after AS CHAR), wt.currency, wt.source_type, wt.source_reference, wt.external_reference,
			wt.idempotency_key, wt.occurred_at, wt.note, wt.created_by_user_id, u.name, wt.created_at
		FROM wallet_transactions wt
		LEFT JOIN owners o ON o.id = wt.owner_id AND o.deleted_at IS NULL
		LEFT JOIN wallet_accounts wa ON wa.id = wt.wallet_account_id
		LEFT JOIN users u ON u.id = wt.created_by_user_id`
}

func walletWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"o.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "o.id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(COALESCE(wa.account_code, CONCAT('WALLET-OWNER-', LPAD(o.id, 6, '0'))) LIKE ? OR o.code LIKE ? OR o.name LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if params.OwnerID != nil {
		where = append(where, "o.id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.Status != "" {
		where = append(where, "COALESCE(wa.status, 'ACTIVE') = ?")
		args = append(args, params.Status)
	}
	return strings.Join(where, " AND "), args
}

func paymentWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"wp.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "wp.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(wp.code LIKE ? OR wp.external_reference LIKE ? OR o.name LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if params.OwnerID != nil {
		where = append(where, "wp.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.PaymentType != "" {
		where = append(where, "wp.payment_type = ?")
		args = append(args, params.PaymentType)
	}
	if params.Channel != "" {
		where = append(where, "wp.payment_channel = ?")
		args = append(args, params.Channel)
	}
	if params.PaidFrom != nil {
		where = append(where, "wp.paid_at >= ?")
		args = append(args, *params.PaidFrom)
	}
	if params.PaidTo != nil {
		where = append(where, "wp.paid_at <= ?")
		args = append(args, *params.PaidTo)
	}
	return strings.Join(where, " AND "), args
}

func transactionWhere(actor identity.User, params ListParams) (string, []any) {
	where := []string{"wt.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "wt.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	if params.Query != "" {
		pattern := like(params.Query)
		where = append(where, "(wt.code LIKE ? OR wt.source_reference LIKE ? OR wt.external_reference LIKE ? OR o.name LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if params.OwnerID != nil {
		where = append(where, "wt.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.Direction != "" {
		where = append(where, "wt.direction = ?")
		args = append(args, params.Direction)
	}
	if params.Type != "" {
		where = append(where, "wt.transaction_type = ?")
		args = append(args, params.Type)
	}
	if params.OccurredFrom != nil {
		where = append(where, "wt.occurred_at >= ?")
		args = append(args, *params.OccurredFrom)
	}
	if params.OccurredTo != nil {
		where = append(where, "wt.occurred_at <= ?")
		args = append(args, *params.OccurredTo)
	}
	return strings.Join(where, " AND "), args
}

func ownerVisibilityWhere(actor identity.User, ownerColumn string) (string, []any) {
	switch actor.RoleCode {
	case RoleAdmin:
		return "1 = 1", nil
	case RoleSupervisor:
		return `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = ` + ownerColumn + `
				AND cl.deleted_at IS NULL
				AND (cl.current_owner_user_id = ? OR cl.supervisor_id = ?)
		)`, []any{actor.ID, actor.ID}
	case RoleSales:
		return `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = ` + ownerColumn + `
				AND cl.deleted_at IS NULL
				AND cl.current_owner_role = 'SALES'
				AND cl.current_owner_user_id = ?
		)`, []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func walletOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"created_at": "COALESCE(wa.created_at, o.created_at)",
		"updated_at": "COALESCE(wa.updated_at, o.updated_at)",
		"balance":    "COALESCE(wa.balance, 0)",
		"code":       "COALESCE(wa.account_code, CONCAT('WALLET-OWNER-', LPAD(o.id, 6, '0')))",
	}, "COALESCE(wa.created_at, o.created_at) DESC, o.id DESC")
}

func paymentOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"paid_at":    "wp.paid_at",
		"created_at": "wp.created_at",
		"amount":     "wp.amount",
		"code":       "wp.code",
	}, "wp.paid_at DESC, wp.id DESC")
}

func transactionOrderBy(sort string) (string, error) {
	return orderBy(sort, map[string]string{
		"occurred_at": "wt.occurred_at",
		"created_at":  "wt.created_at",
		"amount":      "wt.amount",
		"code":        "wt.code",
	}, "wt.occurred_at DESC, wt.id DESC")
}

func scanWallets(rows *sql.Rows, total int64) ([]WalletAccount, int64, error) {
	items := []WalletAccount{}
	for rows.Next() {
		item, err := scanWallet(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func scanPayments(rows *sql.Rows, total int64) ([]WalletPayment, int64, error) {
	items := []WalletPayment{}
	for rows.Next() {
		item, err := scanPayment(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func scanTransactions(rows *sql.Rows, total int64) ([]WalletTransaction, int64, error) {
	items := []WalletTransaction{}
	for rows.Next() {
		item, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanWallet(row scanner) (WalletAccount, error) {
	var item WalletAccount
	err := row.Scan(&item.ID, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.AccountCode, &item.Currency, &item.Balance, &item.LedgerBalance, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanPayment(row scanner) (WalletPayment, error) {
	var item WalletPayment
	err := row.Scan(&item.ID, &item.Code, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.WalletAccountID, &item.PaymentType, &item.PaymentChannel, &item.ExternalReference, &item.IdempotencyKey, &item.Amount, &item.Currency, &item.Status, &item.PaidAt, &item.Note, &item.CreatedByUserID, &item.CreatedByName, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanTransaction(row scanner) (WalletTransaction, error) {
	var item WalletTransaction
	err := row.Scan(&item.ID, &item.Code, &item.WalletAccountID, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.PaymentID, &item.TransactionType, &item.Direction, &item.Amount, &item.BalanceBefore, &item.BalanceAfter, &item.Currency, &item.SourceType, &item.SourceReference, &item.ExternalReference, &item.IdempotencyKey, &item.OccurredAt, &item.Note, &item.CreatedByUserID, &item.CreatedByName, &item.CreatedAt)
	return item, err
}

func topupIdempotencyKey(req CreateTopupRequest) (string, error) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key != "" {
		return "topup:" + key, nil
	}
	externalReference := strings.TrimSpace(req.ExternalReference)
	if externalReference == "" {
		return "", ErrIdempotencyRequired
	}
	return "topup:external:" + externalReference, nil
}

func transactionIdempotencyKey(sourceType string, req CreateWalletTransactionRequest) (string, error) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key != "" {
		return strings.ToLower(sourceType) + ":" + key, nil
	}
	externalReference := strings.TrimSpace(req.ExternalReference)
	if externalReference == "" {
		return "", ErrIdempotencyRequired
	}
	return strings.ToLower(sourceType) + ":external:" + externalReference, nil
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeDirection(value string) string {
	value = normalizeCode(value)
	switch value {
	case DirectionCredit, DirectionDebit:
		return value
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

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func like(value string) string { return "%" + strings.TrimSpace(value) + "%" }

func nextCode(prefix string, at time.Time, seed int64) string {
	return fmt.Sprintf("%s-%s-%06d-%06d", prefix, at.Format("20060102150405"), seed, at.Nanosecond()/1000)
}
