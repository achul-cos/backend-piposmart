package wallet

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListWallets(ctx context.Context, actor identity.User, params ListParams) (WalletListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListWallets(ctx, actor, params)
	if err != nil {
		return WalletListResponse{}, err
	}
	responses, err := walletResponses(items)
	if err != nil {
		return WalletListResponse{}, err
	}
	return WalletListResponse{Items: responses, Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(responses), total), Total: total}}, nil
}

func (s *Service) GetOwnerWallet(ctx context.Context, actor identity.User, ownerID int64) (WalletAccountResponse, error) {
	item, err := s.repo.FindWalletByOwner(ctx, actor, ownerID)
	if err != nil {
		return WalletAccountResponse{}, err
	}
	response := NewWalletAccountResponse(item)
	if response.Balance != response.LedgerBalance {
		return WalletAccountResponse{}, ErrLedgerOutOfSync
	}
	return response, nil
}

func (s *Service) ListPayments(ctx context.Context, actor identity.User, params ListParams) (PaymentListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListPayments(ctx, actor, params)
	if err != nil {
		return PaymentListResponse{}, err
	}
	return PaymentListResponse{Items: paymentResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) GetPayment(ctx context.Context, actor identity.User, id int64) (WalletPaymentResponse, error) {
	item, err := s.repo.FindPaymentByID(ctx, actor, id)
	if err != nil {
		return WalletPaymentResponse{}, err
	}
	return NewWalletPaymentResponse(item), nil
}

func (s *Service) ListTransactions(ctx context.Context, actor identity.User, params ListParams) (TransactionListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListTransactions(ctx, actor, params)
	if err != nil {
		return TransactionListResponse{}, err
	}
	return TransactionListResponse{Items: transactionResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) ListOwnerTransactions(ctx context.Context, actor identity.User, ownerID int64, params ListParams) (TransactionListResponse, error) {
	params.OwnerID = &ownerID
	return s.ListTransactions(ctx, actor, params)
}

func (s *Service) CreateTopup(ctx context.Context, actor identity.User, ownerID int64, req CreateTopupRequest) (TopupResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return TopupResponse{}, ErrForbidden
	}
	if err := validateTopupRequest(req); err != nil {
		return TopupResponse{}, err
	}
	result, err := s.repo.CreateTopup(ctx, actor, ownerID, req)
	if err != nil {
		return TopupResponse{}, err
	}
	return newTopupResponse(result), nil
}

// AcceptTopup confirms a PENDING top-up as a genuine, matched transfer and credits the wallet.
func (s *Service) AcceptTopup(ctx context.Context, actor identity.User, paymentID int64, req AcceptTopupRequest) (TopupResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return TopupResponse{}, ErrForbidden
	}
	result, err := s.repo.AcceptTopup(ctx, actor, paymentID, req.UniqueCode, req.TransferDateOverride)
	if err != nil {
		return TopupResponse{}, err
	}
	if result.Wallet.Balance != result.Wallet.LedgerBalance {
		return TopupResponse{}, ErrLedgerOutOfSync
	}
	return newTopupResponse(result), nil
}

// RejectTopup cancels a PENDING top-up — it never touched balance, so there's nothing to reverse.
func (s *Service) RejectTopup(ctx context.Context, actor identity.User, paymentID int64, req RejectTopupRequest) (WalletPaymentResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return WalletPaymentResponse{}, ErrForbidden
	}
	payment, err := s.repo.RejectTopup(ctx, actor, paymentID, req.Note)
	if err != nil {
		return WalletPaymentResponse{}, err
	}
	return NewWalletPaymentResponse(payment), nil
}

// SetTransferDateOverride lets admin correct a top-up's effective transfer date from the payment
// proof/receipt, regardless of the top-up's current status.
func (s *Service) SetTransferDateOverride(ctx context.Context, actor identity.User, paymentID int64, req SetTransferDateOverrideRequest) (WalletPaymentResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return WalletPaymentResponse{}, ErrForbidden
	}
	payment, err := s.repo.SetTransferDateOverride(ctx, paymentID, req.TransferDate)
	if err != nil {
		return WalletPaymentResponse{}, err
	}
	return NewWalletPaymentResponse(payment), nil
}

func newTopupResponse(result TopupResult) TopupResponse {
	var transaction *WalletTransactionResponse
	if result.Transaction != nil {
		resp := NewWalletTransactionResponse(*result.Transaction)
		transaction = &resp
	}
	return TopupResponse{
		Payment:     NewWalletPaymentResponse(result.Payment),
		Transaction: transaction,
		Wallet:      NewWalletAccountResponse(result.Wallet),
		Idempotent:  result.Idempotent,
	}
}

func (s *Service) CreateDebit(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest) (WalletTransactionResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return WalletTransactionResponse{}, ErrForbidden
	}
	if err := validateManualTransactionRequest(req, false); err != nil {
		return WalletTransactionResponse{}, err
	}
	item, err := s.repo.CreateDebit(ctx, actor, ownerID, req)
	if err != nil {
		return WalletTransactionResponse{}, err
	}
	return NewWalletTransactionResponse(item), nil
}

func (s *Service) CreateAdjustment(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest) (WalletTransactionResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return WalletTransactionResponse{}, ErrForbidden
	}
	if err := validateManualTransactionRequest(req, true); err != nil {
		return WalletTransactionResponse{}, err
	}
	item, err := s.repo.CreateAdjustment(ctx, actor, ownerID, req)
	if err != nil {
		return WalletTransactionResponse{}, err
	}
	return NewWalletTransactionResponse(item), nil
}

func (s *Service) CreateRefund(ctx context.Context, actor identity.User, ownerID int64, req CreateWalletTransactionRequest) (WalletTransactionResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return WalletTransactionResponse{}, ErrForbidden
	}
	if err := validateManualTransactionRequest(req, false); err != nil {
		return WalletTransactionResponse{}, err
	}
	item, err := s.repo.CreateRefund(ctx, actor, ownerID, req)
	if err != nil {
		return WalletTransactionResponse{}, err
	}
	return NewWalletTransactionResponse(item), nil
}

func validateTopupRequest(req CreateTopupRequest) error {
	if _, err := parsePositiveMoneyToCents(req.Amount); err != nil {
		return err
	}
	if _, err := topupIdempotencyKey(req); err != nil {
		return err
	}
	return nil
}

func validateManualTransactionRequest(req CreateWalletTransactionRequest, requireDirection bool) error {
	if _, err := parsePositiveMoneyToCents(req.Amount); err != nil {
		return err
	}
	if requireDirection && normalizeDirection(req.Direction) == "" {
		return ErrInvalidDirection
	}
	if _, err := transactionIdempotencyKey("manual", req); err != nil {
		return err
	}
	return nil
}

func normalizeListParams(params ListParams) ListParams {
	if params.All {
		params.Page = 1
		params.Limit = 0
	} else {
		if params.Page < 1 {
			params.Page = 1
		}
		if params.Limit < 1 {
			params.Limit = 10
		}
		if params.Limit > 100 {
			params.Limit = 100
		}
	}
	params.Query = strings.TrimSpace(params.Query)
	params.Status = normalizeCode(params.Status)
	params.PaymentType = normalizeCode(params.PaymentType)
	params.Channel = normalizeCode(params.Channel)
	params.Direction = normalizeDirection(params.Direction)
	params.Type = normalizeCode(params.Type)
	params.Sort = strings.TrimSpace(params.Sort)
	return params
}

func resolveReturnedLimit(all bool, limit int, itemCount int, total int64) int {
	if !all {
		return limit
	}
	if total == 0 {
		return 0
	}
	return itemCount
}

func walletResponses(items []WalletAccount) ([]WalletAccountResponse, error) {
	responses := make([]WalletAccountResponse, 0, len(items))
	for _, item := range items {
		response := NewWalletAccountResponse(item)
		if response.Balance != response.LedgerBalance {
			return nil, ErrLedgerOutOfSync
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func paymentResponses(items []WalletPayment) []WalletPaymentResponse {
	responses := make([]WalletPaymentResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewWalletPaymentResponse(item))
	}
	return responses
}

func transactionResponses(items []WalletTransaction) []WalletTransactionResponse {
	responses := make([]WalletTransactionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewWalletTransactionResponse(item))
	}
	return responses
}

func parseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func nullableID(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}
