package transfer

import (
	"context"
	"fmt"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/wallet"
)

// walletPort is the slice of wallet.Service this package needs — a PENDING top-up to suggest
// matches against, and the ability to accept one once admin confirms a match. Kept as a small
// interface (rather than importing *wallet.Service directly) purely for testability; the real
// wiring (router.go) always passes the concrete *wallet.Service.
type walletPort interface {
	ListPayments(ctx context.Context, actor identity.User, params wallet.ListParams) (wallet.PaymentListResponse, error)
	GetPayment(ctx context.Context, actor identity.User, id int64) (wallet.WalletPaymentResponse, error)
	AcceptTopup(ctx context.Context, actor identity.User, paymentID int64, req wallet.AcceptTopupRequest) (wallet.TopupResponse, error)
}

type Service struct {
	repo   *Repository
	wallet walletPort
}

func NewService(repo *Repository, walletService walletPort) *Service {
	return &Service{repo: repo, wallet: walletService}
}

func isAdmin(actor identity.User) bool { return actor.RoleCode == RoleAdmin }

func (s *Service) CreateTransfer(ctx context.Context, actor identity.User, ownerID int64, req CreateTransferRequest) (TransferResponse, error) {
	if !isAdmin(actor) {
		return TransferResponse{}, ErrForbidden
	}
	item, err := s.repo.CreateTransfer(ctx, ownerID, req)
	if err != nil {
		return TransferResponse{}, err
	}
	return NewTransferResponse(item), nil
}

func (s *Service) GetTransfer(ctx context.Context, actor identity.User, id int64) (TransferResponse, error) {
	if !isAdmin(actor) {
		return TransferResponse{}, ErrForbidden
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return TransferResponse{}, err
	}
	return NewTransferResponse(item), nil
}

func (s *Service) ListTransfers(ctx context.Context, actor identity.User, params ListParams) (TransferListResponse, error) {
	if !isAdmin(actor) {
		return TransferListResponse{}, ErrForbidden
	}
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return TransferListResponse{}, err
	}
	responses := make([]TransferResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewTransferResponse(item))
	}
	limit := params.Limit
	if params.All {
		limit = len(responses)
	}
	return TransferListResponse{Items: responses, Pagination: PaginationMeta{Page: params.Page, Limit: limit, Total: total}}, nil
}

// SuggestMatches pairs ownerID's UNMATCHED/SUGGESTED transfers with their PENDING top-ups —
// pattern: top-up request happens first, the owner's transfer follows. It is read-only except
// for flagging matched transfer candidates as SUGGESTED (never MATCHED, never touching the
// wallet) — the actual match still requires an explicit admin ConfirmMatch call.
func (s *Service) SuggestMatches(ctx context.Context, actor identity.User, ownerID int64) ([]MatchSuggestion, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	transfers, err := s.repo.ListUnresolvedByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if len(transfers) == 0 {
		return []MatchSuggestion{}, nil
	}
	pending, err := s.wallet.ListPayments(ctx, actor, wallet.ListParams{
		OwnerID: &ownerID, Status: wallet.PaymentStatusPending, All: true,
	})
	if err != nil {
		return nil, err
	}
	if len(pending.Items) == 0 {
		return []MatchSuggestion{}, nil
	}

	suggestions := make([]MatchSuggestion, 0, len(transfers))
	for _, t := range transfers {
		transferCents, err := parseMoneyToCents(t.Amount)
		if err != nil {
			continue
		}
		// Pattern: top-up is requested first, then the owner transfers — only consider top-ups
		// created at or before the transfer date, closest first.
		var best *wallet.WalletPaymentResponse
		var bestUniqueCode string
		var bestMismatch bool
		var bestDiff int64 = -1
		for i := range pending.Items {
			payment := pending.Items[i]
			if payment.CreatedAt.After(t.TransferDate) {
				continue
			}
			topupCents, err := parseMoneyToCents(payment.Amount)
			if err != nil {
				continue
			}
			diff := transferCents - topupCents
			if diff < 0 {
				diff = -diff
			}
			if best == nil || diff < bestDiff {
				best = &payment
				bestDiff = diff
				bestUniqueCode, bestMismatch = classifyDiff(transferCents, topupCents)
			}
		}
		if best == nil {
			continue
		}
		if err := s.repo.MarkSuggested(ctx, t.ID); err != nil {
			return nil, err
		}
		refreshed, err := s.repo.GetByID(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, MatchSuggestion{
			Transfer:            NewTransferResponse(refreshed),
			WalletPaymentID:     best.ID,
			WalletPaymentCode:   best.Code,
			WalletPaymentAmount: best.Amount,
			UniqueCode:          bestUniqueCode,
			AmountMismatch:      bestMismatch,
		})
	}
	return suggestions, nil
}

// classifyDiff decides whether a transfer-vs-topup amount difference is a plausible manual-
// transfer unique code (small, e.g. Rp 123 on a Rp 34.000 request) or a genuine mismatch that
// needs a human decision rather than an auto-suggested unique code.
func classifyDiff(transferCents, topupCents int64) (uniqueCode string, mismatch bool) {
	diff := transferCents - topupCents
	if diff == 0 {
		return "", false
	}
	absDiff := diff
	if absDiff < 0 {
		absDiff = -absDiff
	}
	if diff > 0 && absDiff <= uniqueCodeMaxRupiah*100 {
		return fmt.Sprintf("%03d", absDiff/100), false
	}
	return "", true
}

// ConfirmMatch is the explicit admin decision SuggestMatches always defers to: it records the
// match on the transfer AND accepts the underlying top-up (crediting the owner's wallet) as one
// logical operation. If AcceptTopup fails, the transfer-side match is not left dangling — see the
// rollback-on-failure handling below.
func (s *Service) ConfirmMatch(ctx context.Context, actor identity.User, transferID int64, req ConfirmMatchRequest) (TransferResponse, error) {
	if !isAdmin(actor) {
		return TransferResponse{}, ErrForbidden
	}
	transfer, err := s.repo.GetByID(ctx, transferID)
	if err != nil {
		return TransferResponse{}, err
	}
	if transfer.MatchStatus == MatchStatusMatched {
		return TransferResponse{}, ErrAlreadyMatched
	}

	uniqueCode := req.UniqueCode
	if uniqueCode == "" {
		payment, err := s.wallet.GetPayment(ctx, actor, req.WalletPaymentID)
		if err != nil {
			return TransferResponse{}, err
		}
		transferCents, err := parseMoneyToCents(transfer.Amount)
		if err != nil {
			return TransferResponse{}, err
		}
		topupCents, err := parseMoneyToCents(payment.Amount)
		if err != nil {
			return TransferResponse{}, err
		}
		uniqueCode, _ = classifyDiff(transferCents, topupCents)
	}

	updated, err := s.repo.ConfirmMatch(ctx, transferID, req.WalletPaymentID)
	if err != nil {
		return TransferResponse{}, err
	}

	transferDate := transfer.TransferDate
	if _, err := s.wallet.AcceptTopup(ctx, actor, req.WalletPaymentID, wallet.AcceptTopupRequest{
		UniqueCode:           uniqueCode,
		TransferDateOverride: &transferDate,
	}); err != nil {
		// Best-effort rollback of the transfer-side match so a failed AcceptTopup doesn't leave
		// this transfer stuck as falsely MATCHED with no corresponding credited top-up.
		_, _ = s.repo.RejectMatch(ctx, transferID, "auto-revert: AcceptTopup gagal: "+err.Error())
		return TransferResponse{}, err
	}
	return NewTransferResponse(updated), nil
}

func (s *Service) RejectMatch(ctx context.Context, actor identity.User, transferID int64, req RejectMatchRequest) (TransferResponse, error) {
	if !isAdmin(actor) {
		return TransferResponse{}, ErrForbidden
	}
	item, err := s.repo.RejectMatch(ctx, transferID, req.Note)
	if err != nil {
		return TransferResponse{}, err
	}
	return NewTransferResponse(item), nil
}
