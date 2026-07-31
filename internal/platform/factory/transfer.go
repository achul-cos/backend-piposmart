package factory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Transfer demo-seeds owner_transfers (Sprint 15a §3) — bank-transfer proof from an owner,
// tracked separately from wallet_payments (top-up) until matched against a PENDING top-up.
type Transfer struct {
	Amount                 string
	TransferDate           time.Time
	ProofURL               string
	Note                   string
	MatchStatus            string // UNMATCHED | SUGGESTED | MATCHED | REJECTED_MATCH
	MatchedWalletPaymentID sql.NullInt64
	ExternalReference      string
}

func (f *Factory) BuildTransfer(index int, amount, matchStatus string) Transfer {
	return Transfer{
		Amount:            amount,
		TransferDate:      f.asOf.AddDate(0, 0, index).Add(9 * time.Hour),
		Note:              fmt.Sprintf("Demo Sprint 15a transfer %02d", index),
		MatchStatus:       matchStatus,
		ExternalReference: fmt.Sprintf("DEMO-TRF-%02d-%s", index, f.asOf.Format("20060102")),
	}
}

func (f *Factory) CreateTransfer(ctx context.Context, tx *sql.Tx, ownerID int64, item Transfer) (int64, error) {
	if item.TransferDate.IsZero() {
		item.TransferDate = f.asOf
	}
	matchStatus := strings.ToUpper(strings.TrimSpace(item.MatchStatus))
	if matchStatus == "" {
		matchStatus = "UNMATCHED"
	}

	var existingID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM owner_transfers
		WHERE owner_id = ? AND external_reference = ? AND deleted_at IS NULL
		LIMIT 1`, ownerID, item.ExternalReference).Scan(&existingID)
	if err == nil {
		return existingID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup existing seed transfer owner=%d: %w", ownerID, err)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO owner_transfers
			(owner_id, amount, transfer_date, proof_url, note, matched_wallet_payment_id, match_status, source, external_reference)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'MANUAL_ENTRY', ?)`,
		ownerID,
		item.Amount,
		item.TransferDate,
		nullableSeedString(item.ProofURL),
		nullableSeedString(item.Note),
		item.MatchedWalletPaymentID,
		matchStatus,
		nullableSeedString(item.ExternalReference),
	)
	if err != nil {
		return 0, fmt.Errorf("seed transfer owner=%d: %w", ownerID, err)
	}
	return result.LastInsertId()
}
