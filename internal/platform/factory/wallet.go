package factory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type WalletTopup struct {
	Amount            string
	PaymentChannel    string
	ExternalReference string
	IdempotencyKey    string
	PaidAt            time.Time
	Note              string
}

type WalletLedgerTransaction struct {
	TransactionType   string
	Direction         string
	Amount            string
	SourceType        string
	SourceReference   string
	ExternalReference string
	IdempotencyKey    string
	OccurredAt        time.Time
	Note              string
}

func (f *Factory) BuildWalletTopup(index int, amount string) WalletTopup {
	return WalletTopup{
		Amount:            amount,
		PaymentChannel:    "MANUAL_TRANSFER",
		ExternalReference: fmt.Sprintf("DEMO-TOPUP-%02d-%s", index, f.asOf.Format("20060102")),
		IdempotencyKey:    fmt.Sprintf("demo:topup:%02d:%s", index, f.asOf.Format("20060102")),
		PaidAt:            f.asOf.AddDate(0, 0, index).Add(9 * time.Hour),
		Note:              fmt.Sprintf("Demo Sprint 09 top-up %02d", index),
	}
}

func (f *Factory) BuildWalletDebit(index int, amount string) WalletLedgerTransaction {
	return WalletLedgerTransaction{
		TransactionType:   "DEBIT",
		Direction:         "DEBIT",
		Amount:            amount,
		SourceType:        "MANUAL_DEBIT",
		SourceReference:   fmt.Sprintf("DEMO-USAGE-%02d", index),
		ExternalReference: fmt.Sprintf("DEMO-DEBIT-%02d-%s", index, f.asOf.Format("20060102")),
		IdempotencyKey:    fmt.Sprintf("demo:debit:%02d:%s", index, f.asOf.Format("20060102")),
		OccurredAt:        f.asOf.AddDate(0, 0, index).Add(11 * time.Hour),
		Note:              fmt.Sprintf("Demo Sprint 09 saldo terpakai %02d", index),
	}
}

func (f *Factory) CreateWalletTopup(ctx context.Context, tx *sql.Tx, ownerID int64, topup WalletTopup) (int64, error) {
	adminID, err := lookupFirstUserIDByRole(ctx, tx, "ADMIN")
	if err != nil {
		return 0, err
	}
	if topup.PaidAt.IsZero() {
		topup.PaidAt = f.asOf
	}
	amountCents, err := parseSeedMoneyToCents(topup.Amount)
	if err != nil {
		return 0, err
	}
	if amountCents <= 0 {
		return 0, fmt.Errorf("seed wallet top-up amount harus positif owner=%d", ownerID)
	}
	idempotencyKey := strings.TrimSpace(topup.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("demo:topup:owner:%06d:%s", ownerID, topup.ExternalReference)
	}
	if existingID, found, err := lookupExistingWalletPaymentByKey(ctx, tx, idempotencyKey); err != nil {
		return 0, err
	} else if found {
		return existingID, nil
	}
	wallet, err := lockOrCreateSeedWallet(ctx, tx, ownerID)
	if err != nil {
		return 0, err
	}
	beforeCents, err := parseSeedMoneyToCents(wallet.balance)
	if err != nil {
		return 0, err
	}
	afterCents := beforeCents + amountCents
	paymentCode := fmt.Sprintf("DEMO-PAY-%06d-%02d", ownerID, topup.PaidAt.Day())
	result, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_payments
			(code, owner_id, wallet_account_id, payment_type, payment_channel, external_reference,
			 idempotency_key, amount, currency, status, paid_at, note, created_by_user_id)
		VALUES (?, ?, ?, 'TOPUP', ?, ?, ?, ?, ?, 'PAID', ?, ?, ?)`,
		paymentCode,
		ownerID,
		wallet.id,
		normalizeSeedCode(topup.PaymentChannel, "MANUAL"),
		nullableSeedString(topup.ExternalReference),
		idempotencyKey,
		formatSeedCents(amountCents),
		wallet.currency,
		topup.PaidAt,
		nullableSeedString(topup.Note),
		adminID,
	)
	if err != nil {
		return 0, fmt.Errorf("seed wallet top-up owner=%d: %w", ownerID, err)
	}
	paymentID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := insertSeedWalletTransaction(ctx, tx, seedWalletTransactionInput{
		WalletID:          wallet.id,
		OwnerID:           ownerID,
		PaymentID:         sql.NullInt64{Int64: paymentID, Valid: true},
		TransactionType:   "CREDIT",
		Direction:         "CREDIT",
		AmountCents:       amountCents,
		BalanceBefore:     beforeCents,
		BalanceAfter:      afterCents,
		Currency:          wallet.currency,
		SourceType:        "TOPUP",
		SourceReference:   paymentCode,
		ExternalReference: topup.ExternalReference,
		IdempotencyKey:    idempotencyKey,
		OccurredAt:        topup.PaidAt,
		Note:              topup.Note,
		CreatedByUserID:   adminID,
	}); err != nil {
		return 0, err
	}
	if err := updateSeedWalletBalance(ctx, tx, wallet.id, afterCents); err != nil {
		return 0, err
	}
	return paymentID, nil
}

func (f *Factory) CreateWalletDebit(ctx context.Context, tx *sql.Tx, ownerID int64, item WalletLedgerTransaction) (int64, error) {
	adminID, err := lookupFirstUserIDByRole(ctx, tx, "ADMIN")
	if err != nil {
		return 0, err
	}
	if item.OccurredAt.IsZero() {
		item.OccurredAt = f.asOf
	}
	amountCents, err := parseSeedMoneyToCents(item.Amount)
	if err != nil {
		return 0, err
	}
	if amountCents <= 0 {
		return 0, fmt.Errorf("seed wallet debit amount harus positif owner=%d", ownerID)
	}
	idempotencyKey := strings.TrimSpace(item.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("demo:debit:owner:%06d:%s", ownerID, item.ExternalReference)
	}
	if existingID, found, err := lookupExistingWalletTransactionByKey(ctx, tx, idempotencyKey); err != nil {
		return 0, err
	} else if found {
		return existingID, nil
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
	if afterCents < 0 {
		return 0, fmt.Errorf("seed wallet debit melebihi saldo owner=%d", ownerID)
	}
	if err := insertSeedWalletTransaction(ctx, tx, seedWalletTransactionInput{
		WalletID:          wallet.id,
		OwnerID:           ownerID,
		TransactionType:   normalizeSeedCode(item.TransactionType, "DEBIT"),
		Direction:         "DEBIT",
		AmountCents:       amountCents,
		BalanceBefore:     beforeCents,
		BalanceAfter:      afterCents,
		Currency:          wallet.currency,
		SourceType:        normalizeSeedCode(item.SourceType, "MANUAL_DEBIT"),
		SourceReference:   item.SourceReference,
		ExternalReference: item.ExternalReference,
		IdempotencyKey:    idempotencyKey,
		OccurredAt:        item.OccurredAt,
		Note:              item.Note,
		CreatedByUserID:   adminID,
	}); err != nil {
		return 0, err
	}
	if err := updateSeedWalletBalance(ctx, tx, wallet.id, afterCents); err != nil {
		return 0, err
	}
	return lookupExistingWalletTransactionID(ctx, tx, idempotencyKey)
}

type seedWallet struct {
	id       int64
	balance  string
	currency string
}

type seedWalletTransactionInput struct {
	WalletID          int64
	OwnerID           int64
	PaymentID         sql.NullInt64
	TransactionType   string
	Direction         string
	AmountCents       int64
	BalanceBefore     int64
	BalanceAfter      int64
	Currency          string
	SourceType        string
	SourceReference   string
	ExternalReference string
	IdempotencyKey    string
	OccurredAt        time.Time
	Note              string
	CreatedByUserID   int64
}

func lockOrCreateSeedWallet(ctx context.Context, tx *sql.Tx, ownerID int64) (seedWallet, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_accounts (owner_id, account_code, currency, balance, status)
		VALUES (?, ?, 'IDR', 0, 'ACTIVE')
		ON DUPLICATE KEY UPDATE status = 'ACTIVE', deleted_at = NULL`,
		ownerID,
		fmt.Sprintf("WALLET-OWNER-%06d", ownerID),
	); err != nil {
		return seedWallet{}, fmt.Errorf("seed wallet owner=%d: %w", ownerID, err)
	}
	var wallet seedWallet
	err := tx.QueryRowContext(ctx, `
		SELECT id, CAST(balance AS CHAR), currency
		FROM wallet_accounts
		WHERE owner_id = ? AND deleted_at IS NULL
		FOR UPDATE`, ownerID).Scan(&wallet.id, &wallet.balance, &wallet.currency)
	if err != nil {
		return seedWallet{}, fmt.Errorf("lock seed wallet owner=%d: %w", ownerID, err)
	}
	return wallet, nil
}

func insertSeedWalletTransaction(ctx context.Context, tx *sql.Tx, input seedWalletTransactionInput) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions
			(code, wallet_account_id, owner_id, payment_id, transaction_type, direction, amount,
			 balance_before, balance_after, currency, source_type, source_reference, external_reference,
			 idempotency_key, occurred_at, note, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("DEMO-WTX-%06d-%s", input.OwnerID, compactSeedKey(input.IdempotencyKey)),
		input.WalletID,
		input.OwnerID,
		input.PaymentID,
		input.TransactionType,
		input.Direction,
		formatSeedCents(input.AmountCents),
		formatSeedCents(input.BalanceBefore),
		formatSeedCents(input.BalanceAfter),
		input.Currency,
		input.SourceType,
		nullableSeedString(input.SourceReference),
		nullableSeedString(input.ExternalReference),
		input.IdempotencyKey,
		input.OccurredAt,
		nullableSeedString(input.Note),
		input.CreatedByUserID,
	)
	if err != nil {
		return fmt.Errorf("seed wallet transaction %s: %w", input.IdempotencyKey, err)
	}
	return nil
}

func updateSeedWalletBalance(ctx context.Context, tx *sql.Tx, walletID int64, balanceCents int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE wallet_accounts
		SET balance = ?
		WHERE id = ?`, formatSeedCents(balanceCents), walletID)
	if err != nil {
		return fmt.Errorf("update seed wallet balance id=%d: %w", walletID, err)
	}
	return nil
}

func lookupExistingWalletPaymentByKey(ctx context.Context, tx *sql.Tx, key string) (int64, bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM wallet_payments
		WHERE idempotency_key = ? AND deleted_at IS NULL
		LIMIT 1`, key).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lookup wallet payment key=%s: %w", key, err)
	}
	return id, true, nil
}

func lookupExistingWalletTransactionByKey(ctx context.Context, tx *sql.Tx, key string) (int64, bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM wallet_transactions
		WHERE idempotency_key = ? AND deleted_at IS NULL
		LIMIT 1`, key).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lookup wallet transaction key=%s: %w", key, err)
	}
	return id, true, nil
}

func lookupExistingWalletTransactionID(ctx context.Context, tx *sql.Tx, key string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM wallet_transactions
		WHERE idempotency_key = ? AND deleted_at IS NULL
		LIMIT 1`, key).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("lookup wallet transaction key=%s: %w", key, err)
	}
	return id, nil
}

func normalizeSeedCode(value, fallback string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func nullableSeedString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func compactSeedKey(value string) string {
	value = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(value, ":", "-"), "_", "-"))
	if len(value) > 40 {
		return value[len(value)-40:]
	}
	return value
}
