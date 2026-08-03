package transfer

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin = "ADMIN"

	MatchStatusUnmatched     = "UNMATCHED"
	MatchStatusSuggested     = "SUGGESTED"
	MatchStatusMatched       = "MATCHED"
	MatchStatusRejectedMatch = "REJECTED_MATCH"

	SourceManualEntry    = "MANUAL_ENTRY"
	SourceAdminDashboard = "ADMIN_DASHBOARD"

	// uniqueCodeMaxRupiah bounds how large a transfer-vs-topup amount difference can be and
	// still be treated as a manual-transfer unique code (e.g. request Rp 34.000, transfer
	// Rp 34.123) rather than a genuine amount mismatch that needs a human look.
	uniqueCodeMaxRupiah = 999
)

// Transfer is a bukti-transfer record: an owner's bank transfer to the company, tracked
// separately from the Top Up request it's meant to settle (internal/wallet). source/
// external_reference are generic so a future automatic admin-dashboard integration (still an
// open architecture decision) can populate this table without another migration.
type Transfer struct {
	ID                     int64
	OwnerID                int64
	OwnerCode              sql.NullString
	OwnerName              sql.NullString
	Amount                 string
	TransferDate           time.Time
	ProofURL               sql.NullString
	Note                   sql.NullString
	MatchedWalletPaymentID sql.NullInt64
	MatchStatus            string
	Source                 string
	ExternalReference      sql.NullString
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type OwnerRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

type TransferResponse struct {
	ID                     int64     `json:"id"`
	Owner                  OwnerRef  `json:"owner"`
	Amount                 string    `json:"amount"`
	TransferDate           time.Time `json:"transfer_date"`
	ProofURL               string    `json:"proof_url,omitempty"`
	Note                   string    `json:"note,omitempty"`
	MatchedWalletPaymentID *int64    `json:"matched_wallet_payment_id,omitempty"`
	MatchStatus            string    `json:"match_status"`
	Source                 string    `json:"source"`
	ExternalReference      string    `json:"external_reference,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// MatchSuggestion pairs an UNMATCHED transfer with a PENDING top-up that plausibly settles it —
// read-only, computed on demand (SuggestMatches never writes to the DB). AmountMismatch flags a
// difference too large to be a manual-transfer unique code; it's still surfaced (never silently
// dropped) but needs a human decision, not an auto-suggestion an admin can casually accept.
type MatchSuggestion struct {
	Transfer            TransferResponse `json:"transfer"`
	WalletPaymentID     int64            `json:"wallet_payment_id"`
	WalletPaymentCode   string           `json:"wallet_payment_code"`
	WalletPaymentAmount string           `json:"wallet_payment_amount"`
	UniqueCode          string           `json:"unique_code,omitempty"`
	AmountMismatch      bool             `json:"amount_mismatch"`
}

type CreateTransferRequest struct {
	Amount            string    `json:"amount" binding:"required"`
	TransferDate      time.Time `json:"transfer_date" binding:"required"`
	ProofURL          string    `json:"proof_url"`
	Note              string    `json:"note"`
	ExternalReference string    `json:"external_reference"`
}

type ConfirmMatchRequest struct {
	WalletPaymentID int64  `json:"wallet_payment_id" binding:"required"`
	UniqueCode      string `json:"unique_code,omitempty"`
}

type RejectMatchRequest struct {
	Note string `json:"note"`
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type ListParams struct {
	OwnerID     *int64
	MatchStatus string
	All         bool
	Page        int
	Limit       int
	Sort        string
}

type TransferListResponse struct {
	Items      []TransferResponse `json:"items"`
	Pagination PaginationMeta     `json:"pagination"`
}

func NewTransferResponse(item Transfer) TransferResponse {
	var matchedID *int64
	if item.MatchedWalletPaymentID.Valid {
		v := item.MatchedWalletPaymentID.Int64
		matchedID = &v
	}
	return TransferResponse{
		ID: item.ID,
		Owner: OwnerRef{
			ID:   item.OwnerID,
			Code: item.OwnerCode.String,
			Name: item.OwnerName.String,
		},
		Amount:                 item.Amount,
		TransferDate:           item.TransferDate,
		ProofURL:               item.ProofURL.String,
		Note:                   item.Note.String,
		MatchedWalletPaymentID: matchedID,
		MatchStatus:            item.MatchStatus,
		Source:                 item.Source,
		ExternalReference:      item.ExternalReference.String,
		CreatedAt:              item.CreatedAt,
		UpdatedAt:              item.UpdatedAt,
	}
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
