package wallet

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	WalletStatusActive = "ACTIVE"

	PaymentTypeTopup = "TOPUP"

	// Sprint 15a §2: a top-up starts PENDING (no balance/ledger effect yet — the transfer hasn't
	// been verified), auto-EXPIREs after its 24h session window if left untouched, and only
	// credits the wallet once ACCEPTED (by admin, or via Transfer suggest/confirm matching).
	PaymentStatusPending  = "PENDING"
	PaymentStatusRejected = "REJECTED"
	PaymentStatusExpired  = "EXPIRED"
	PaymentStatusAccepted = "ACCEPTED"

	// topupSessionDuration is the 24h window a PENDING top-up has before ExpireStaleTopups moves
	// it to EXPIRED.
	topupSessionDuration = 24 * time.Hour

	DirectionCredit = "CREDIT"
	DirectionDebit  = "DEBIT"

	TransactionTypeCredit     = "CREDIT"
	TransactionTypeDebit      = "DEBIT"
	TransactionTypeAdjustment = "ADJUSTMENT"
	TransactionTypeRefund     = "REFUND"

	SourceTopup       = "TOPUP"
	SourceManualDebit = "MANUAL_DEBIT"
	SourceAdjustment  = "ADJUSTMENT"
	SourceRefund      = "REFUND"
)

type WalletAccount struct {
	ID            int64
	OwnerID       sql.NullInt64
	OwnerCode     sql.NullString
	OwnerName     sql.NullString
	AccountCode   string
	Currency      string
	Balance       string
	LedgerBalance string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type WalletPayment struct {
	ID                   int64
	Code                 string
	OwnerID              sql.NullInt64
	OwnerCode            sql.NullString
	OwnerName            sql.NullString
	WalletAccountID      sql.NullInt64
	PaymentType          string
	PaymentChannel       string
	ExternalReference    sql.NullString
	IdempotencyKey       string
	Amount               string
	Currency             string
	Status               string
	PaidAt               sql.NullTime
	SessionExpiresAt     sql.NullTime
	TransferDateOverride sql.NullTime
	UniqueCode           sql.NullString
	Note                 sql.NullString
	CreatedByUserID      sql.NullInt64
	CreatedByName        sql.NullString
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type WalletTransaction struct {
	ID                int64
	Code              string
	WalletAccountID   sql.NullInt64
	OwnerID           sql.NullInt64
	OwnerCode         sql.NullString
	OwnerName         sql.NullString
	PaymentID         sql.NullInt64
	TransactionType   string
	Direction         string
	Amount            string
	BalanceBefore     string
	BalanceAfter      string
	Currency          string
	SourceType        string
	SourceReference   sql.NullString
	ExternalReference sql.NullString
	IdempotencyKey    string
	OccurredAt        time.Time
	Note              sql.NullString
	CreatedByUserID   sql.NullInt64
	CreatedByName     sql.NullString
	CreatedAt         time.Time
}

type OwnerRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
}

type UserBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type WalletAccountResponse struct {
	ID            int64     `json:"id"`
	Owner         *OwnerRef `json:"owner,omitempty"`
	AccountCode   string    `json:"account_code"`
	Currency      string    `json:"currency"`
	Balance       string    `json:"balance"`
	LedgerBalance string    `json:"ledger_balance"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WalletPaymentResponse struct {
	ID                   int64      `json:"id"`
	Code                 string     `json:"code"`
	Owner                *OwnerRef  `json:"owner,omitempty"`
	WalletAccountID      *int64     `json:"wallet_account_id,omitempty"`
	PaymentType          string     `json:"payment_type"`
	PaymentChannel       string     `json:"payment_channel"`
	ExternalReference    string     `json:"external_reference,omitempty"`
	IdempotencyKey       string     `json:"idempotency_key"`
	Amount               string     `json:"amount"`
	Currency             string     `json:"currency"`
	Status               string     `json:"status"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	SessionExpiresAt     *time.Time `json:"session_expires_at,omitempty"`
	TransferDateOverride *time.Time `json:"transfer_date_override,omitempty"`
	// EffectiveTransferDate is TransferDateOverride when the admin has corrected it (e.g. the
	// receipt shows yesterday but the system recorded it in real time), falling back to PaidAt.
	EffectiveTransferDate *time.Time `json:"effective_transfer_date,omitempty"`
	UniqueCode            string     `json:"unique_code,omitempty"`
	Note                  string     `json:"note,omitempty"`
	CreatedBy             *UserBrief `json:"created_by,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type WalletTransactionResponse struct {
	ID                int64      `json:"id"`
	Code              string     `json:"code"`
	WalletAccountID   *int64     `json:"wallet_account_id,omitempty"`
	Owner             *OwnerRef  `json:"owner,omitempty"`
	PaymentID         *int64     `json:"payment_id,omitempty"`
	TransactionType   string     `json:"transaction_type"`
	Direction         string     `json:"direction"`
	Amount            string     `json:"amount"`
	BalanceBefore     string     `json:"balance_before"`
	BalanceAfter      string     `json:"balance_after"`
	Currency          string     `json:"currency"`
	SourceType        string     `json:"source_type"`
	SourceReference   string     `json:"source_reference,omitempty"`
	ExternalReference string     `json:"external_reference,omitempty"`
	IdempotencyKey    string     `json:"idempotency_key"`
	OccurredAt        time.Time  `json:"occurred_at"`
	Note              string     `json:"note,omitempty"`
	CreatedBy         *UserBrief `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Transaction is nil while the top-up is PENDING — no ledger entry exists yet, since the wallet
// is only credited once the top-up is ACCEPTED (see AcceptTopup).
type TopupResponse struct {
	Payment     WalletPaymentResponse      `json:"payment"`
	Transaction *WalletTransactionResponse `json:"transaction,omitempty"`
	Wallet      WalletAccountResponse      `json:"wallet"`
	Idempotent  bool                       `json:"idempotent"`
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type WalletListResponse struct {
	Items      []WalletAccountResponse `json:"items"`
	Pagination PaginationMeta          `json:"pagination"`
}

type PaymentListResponse struct {
	Items      []WalletPaymentResponse `json:"items"`
	Pagination PaginationMeta          `json:"pagination"`
}

type TransactionListResponse struct {
	Items      []WalletTransactionResponse `json:"items"`
	Pagination PaginationMeta              `json:"pagination"`
}

type CreateTopupRequest struct {
	Amount            string     `json:"amount" binding:"required"`
	PaymentChannel    string     `json:"payment_channel"`
	ExternalReference string     `json:"external_reference"`
	IdempotencyKey    string     `json:"idempotency_key"`
	PaidAt            *time.Time `json:"paid_at"`
	Note              string     `json:"note"`
}

// AcceptTopupRequest confirms a PENDING top-up as a genuine, matched transfer. UniqueCode
// records the trailing digits a manual-transfer amount carries beyond the round requested
// amount (e.g. request Rp 34.000, owner transfers Rp 34.123 — the 123 is recorded here but never
// counted as revenue; the wallet is always credited req.Amount, the round number).
// TransferDateOverride lets admin correct the transfer date from the payment proof/receipt when
// it differs from when the system recorded the top-up.
type AcceptTopupRequest struct {
	UniqueCode           string     `json:"unique_code"`
	TransferDateOverride *time.Time `json:"transfer_date_override"`
}

type RejectTopupRequest struct {
	Note string `json:"note"`
}

type SetTransferDateOverrideRequest struct {
	TransferDate time.Time `json:"transfer_date" binding:"required"`
}

type CreateWalletTransactionRequest struct {
	Amount            string     `json:"amount" binding:"required"`
	Direction         string     `json:"direction"`
	SourceReference   string     `json:"source_reference"`
	ExternalReference string     `json:"external_reference"`
	IdempotencyKey    string     `json:"idempotency_key"`
	OccurredAt        *time.Time `json:"occurred_at"`
	Note              string     `json:"note"`
}

type BulkIDRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,dive,min=1"`
}

type ListParams struct {
	Query        string
	OwnerID      *int64
	Status       string
	PaymentType  string
	Channel      string
	Direction    string
	Type         string
	PaidFrom     *time.Time
	PaidTo       *time.Time
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	All          bool
	Page         int
	Limit        int
	Sort         string
}

func NewWalletAccountResponse(item WalletAccount) WalletAccountResponse {
	return WalletAccountResponse{
		ID:            item.ID,
		Owner:         nullableOwner(item.OwnerID, item.OwnerCode, item.OwnerName),
		AccountCode:   item.AccountCode,
		Currency:      item.Currency,
		Balance:       item.Balance,
		LedgerBalance: item.LedgerBalance,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func NewWalletPaymentResponse(item WalletPayment) WalletPaymentResponse {
	effective := nullableTimePtr(item.TransferDateOverride)
	if effective == nil {
		effective = nullableTimePtr(item.PaidAt)
	}
	return WalletPaymentResponse{
		ID:                    item.ID,
		Code:                  item.Code,
		Owner:                 nullableOwner(item.OwnerID, item.OwnerCode, item.OwnerName),
		WalletAccountID:       nullableInt64Ptr(item.WalletAccountID),
		PaymentType:           item.PaymentType,
		PaymentChannel:        item.PaymentChannel,
		ExternalReference:     item.ExternalReference.String,
		IdempotencyKey:        item.IdempotencyKey,
		Amount:                item.Amount,
		Currency:              item.Currency,
		Status:                item.Status,
		PaidAt:                nullableTimePtr(item.PaidAt),
		SessionExpiresAt:      nullableTimePtr(item.SessionExpiresAt),
		TransferDateOverride:  nullableTimePtr(item.TransferDateOverride),
		EffectiveTransferDate: effective,
		UniqueCode:            item.UniqueCode.String,
		Note:                  item.Note.String,
		CreatedBy:             nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func NewWalletTransactionResponse(item WalletTransaction) WalletTransactionResponse {
	return WalletTransactionResponse{
		ID:                item.ID,
		Code:              item.Code,
		WalletAccountID:   nullableInt64Ptr(item.WalletAccountID),
		Owner:             nullableOwner(item.OwnerID, item.OwnerCode, item.OwnerName),
		PaymentID:         nullableInt64Ptr(item.PaymentID),
		TransactionType:   item.TransactionType,
		Direction:         item.Direction,
		Amount:            item.Amount,
		BalanceBefore:     item.BalanceBefore,
		BalanceAfter:      item.BalanceAfter,
		Currency:          item.Currency,
		SourceType:        item.SourceType,
		SourceReference:   item.SourceReference.String,
		ExternalReference: item.ExternalReference.String,
		IdempotencyKey:    item.IdempotencyKey,
		OccurredAt:        item.OccurredAt,
		Note:              item.Note.String,
		CreatedBy:         nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		CreatedAt:         item.CreatedAt,
	}
}

func nullableOwner(id sql.NullInt64, code sql.NullString, name sql.NullString) *OwnerRef {
	if !id.Valid {
		return nil
	}
	return &OwnerRef{ID: id.Int64, Code: code.String, Name: name.String}
}

func nullableUser(id sql.NullInt64, name sql.NullString, role string) *UserBrief {
	if !id.Valid {
		return nil
	}
	return &UserBrief{ID: id.Int64, Name: name.String, Role: role}
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time
	return &out
}
