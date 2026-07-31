package transfer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func transferSelect() string {
	return `
		SELECT ot.id, ot.owner_id, o.code, o.name, CAST(ot.amount AS CHAR), ot.transfer_date,
			ot.proof_url, ot.note, ot.matched_wallet_payment_id, ot.match_status, ot.source,
			ot.external_reference, ot.created_at, ot.updated_at
		FROM owner_transfers ot
		LEFT JOIN owners o ON o.id = ot.owner_id
		WHERE ot.deleted_at IS NULL`
}

func scanTransfer(row interface{ Scan(dest ...any) error }) (Transfer, error) {
	var item Transfer
	err := row.Scan(&item.ID, &item.OwnerID, &item.OwnerCode, &item.OwnerName, &item.Amount,
		&item.TransferDate, &item.ProofURL, &item.Note, &item.MatchedWalletPaymentID,
		&item.MatchStatus, &item.Source, &item.ExternalReference, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *Repository) ensureOwnerExists(ctx context.Context, ownerID int64) error {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM owners WHERE id = ? AND deleted_at IS NULL`, ownerID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrOwnerNotFound
	}
	return err
}

func (r *Repository) CreateTransfer(ctx context.Context, ownerID int64, req CreateTransferRequest) (Transfer, error) {
	if _, err := parseMoneyToCents(req.Amount); err != nil {
		return Transfer{}, err
	}
	if err := r.ensureOwnerExists(ctx, ownerID); err != nil {
		return Transfer{}, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO owner_transfers
			(owner_id, amount, transfer_date, proof_url, note, match_status, source, external_reference)
		VALUES (?, ?, ?, ?, ?, 'UNMATCHED', 'MANUAL_ENTRY', ?)`,
		ownerID, req.Amount, req.TransferDate, nullableString(req.ProofURL), nullableString(req.Note),
		nullableString(req.ExternalReference),
	)
	if err != nil {
		return Transfer{}, fmt.Errorf("database transfer: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Transfer{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Transfer, error) {
	item, err := scanTransfer(r.db.QueryRowContext(ctx, transferSelect()+" AND ot.id = ?", id))
	if err == sql.ErrNoRows {
		return Transfer{}, ErrNotFound
	}
	return item, err
}

// getForUpdate locks a transfer row within tx — used by ConfirmMatch/RejectMatch so two
// concurrent decisions on the same transfer can't both succeed.
func (r *Repository) getForUpdate(ctx context.Context, tx *sql.Tx, id int64) (Transfer, error) {
	item, err := scanTransfer(tx.QueryRowContext(ctx, transferSelect()+" AND ot.id = ? FOR UPDATE", id))
	if err == sql.ErrNoRows {
		return Transfer{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) List(ctx context.Context, params ListParams) ([]Transfer, int64, error) {
	where := []string{"ot.deleted_at IS NULL"}
	args := []any{}
	if params.OwnerID != nil {
		where = append(where, "ot.owner_id = ?")
		args = append(args, *params.OwnerID)
	}
	if params.MatchStatus != "" {
		where = append(where, "ot.match_status = ?")
		args = append(args, params.MatchStatus)
	}
	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM owner_transfers ot WHERE `+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT ot.id, ot.owner_id, o.code, o.name, CAST(ot.amount AS CHAR), ot.transfer_date,
			ot.proof_url, ot.note, ot.matched_wallet_payment_id, ot.match_status, ot.source,
			ot.external_reference, ot.created_at, ot.updated_at
		FROM owner_transfers ot
		LEFT JOIN owners o ON o.id = ot.owner_id
		WHERE ` + whereClause + `
		ORDER BY ot.transfer_date DESC, ot.id DESC`
	if !params.All {
		page, limit := params.Page, params.Limit
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 200 {
			limit = 50
		}
		args = append(args, limit, (page-1)*limit)
		query += ` LIMIT ? OFFSET ?`
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Transfer{}
	for rows.Next() {
		item, err := scanTransfer(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListUnresolvedByOwner returns UNMATCHED and SUGGESTED transfers for ownerID — the candidate
// pool SuggestMatches considers. REJECTED_MATCH/MATCHED transfers are excluded: a rejected match
// needs an explicit admin decision (ConfirmMatch against a different payment) rather than being
// re-suggested automatically forever.
func (r *Repository) ListUnresolvedByOwner(ctx context.Context, ownerID int64) ([]Transfer, error) {
	rows, err := r.db.QueryContext(ctx, transferSelect()+`
		AND ot.owner_id = ? AND ot.match_status IN ('UNMATCHED', 'SUGGESTED')
		ORDER BY ot.transfer_date ASC, ot.id ASC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Transfer{}
	for rows.Next() {
		item, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// MarkSuggested flags a transfer as SUGGESTED (a candidate top-up match has been computed and
// shown to admin) without touching matched_wallet_payment_id — that's only ever set by
// ConfirmMatch, an explicit admin decision.
func (r *Repository) MarkSuggested(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE owner_transfers SET match_status = 'SUGGESTED', updated_at = NOW()
		WHERE id = ? AND match_status = 'UNMATCHED'`, id)
	return err
}

// ConfirmMatch locks the transfer, verifies it's not already MATCHED, and records the match.
// The caller (Service.ConfirmMatch) is responsible for calling wallet.Service.AcceptTopup in the
// same logical operation — that's a cross-package call the repository layer can't own, so this
// method only persists the owner_transfers side; Service sequences both.
func (r *Repository) ConfirmMatch(ctx context.Context, id int64, walletPaymentID int64) (Transfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer tx.Rollback()

	transfer, err := r.getForUpdate(ctx, tx, id)
	if err != nil {
		return Transfer{}, err
	}
	if transfer.MatchStatus == MatchStatusMatched {
		return Transfer{}, ErrAlreadyMatched
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE owner_transfers
		SET match_status = 'MATCHED', matched_wallet_payment_id = ?, updated_at = NOW()
		WHERE id = ?`, walletPaymentID, id,
	); err != nil {
		return Transfer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) RejectMatch(ctx context.Context, id int64, note string) (Transfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer tx.Rollback()

	transfer, err := r.getForUpdate(ctx, tx, id)
	if err != nil {
		return Transfer{}, err
	}
	if transfer.MatchStatus == MatchStatusMatched {
		return Transfer{}, ErrAlreadyMatched
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE owner_transfers SET match_status = 'REJECTED_MATCH', note = ?, updated_at = NOW()
		WHERE id = ?`, nullableString(note), id,
	); err != nil {
		return Transfer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}
	return r.GetByID(ctx, id)
}
