package closing

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

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListClosings(ctx context.Context, actor identity.User, params ListParams) (ClosingListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListClosings(ctx, actor, params)
	if err != nil {
		return ClosingListResponse{}, err
	}
	return ClosingListResponse{
		Items:      closingResponses(items),
		Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total},
	}, nil
}

func (s *Service) GetClosing(ctx context.Context, actor identity.User, id int64) (ClosingResponse, error) {
	item, err := s.repo.FindClosingByID(ctx, actor, id)
	if err != nil {
		return ClosingResponse{}, err
	}
	response := NewClosingResponse(item)
	promotions, err := s.repo.ListClosingPromotions(ctx, id)
	if err != nil {
		return ClosingResponse{}, err
	}
	response.Promotions = promotions
	return response, nil
}

func (s *Service) CreateClosing(ctx context.Context, actor identity.User, leadID int64, req CreateClosingRequest) (ClosingResponse, error) {
	if actor.RoleCode != RoleSales {
		return ClosingResponse{}, ErrForbidden
	}
	if err := validateCreateClosingRequest(req); err != nil {
		return ClosingResponse{}, err
	}
	item, err := s.repo.CreateClosing(ctx, actor, leadID, req)
	if err != nil {
		return ClosingResponse{}, err
	}
	response := NewClosingResponse(item)
	promotions, err := s.repo.ListClosingPromotions(ctx, item.ID)
	if err != nil {
		return ClosingResponse{}, err
	}
	response.Promotions = promotions
	return response, nil
}

func (s *Service) ConfirmClosing(ctx context.Context, actor identity.User, id int64, req UpdateClosingStatusRequest) (ClosingResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return ClosingResponse{}, ErrForbidden
	}
	item, err := s.repo.ConfirmClosing(ctx, actor, id, req)
	if err != nil {
		return ClosingResponse{}, err
	}
	return NewClosingResponse(item), nil
}

func (s *Service) RejectClosing(ctx context.Context, actor identity.User, id int64, req UpdateClosingStatusRequest) (ClosingResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return ClosingResponse{}, ErrForbidden
	}
	item, err := s.repo.RejectClosing(ctx, actor, id, req)
	if err != nil {
		return ClosingResponse{}, err
	}
	return NewClosingResponse(item), nil
}

func (s *Service) DeleteClosings(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.DeleteClosings(ctx, actor, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func (s *Service) RestoreClosings(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.RestoreClosings(ctx, actor, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func (s *Service) ForceDeleteClosings(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.ForceDeleteClosings(ctx, actor, ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	return BulkActionResponse{IDs: ids, Affected: affected}, nil
}

func validateCreateClosingRequest(req CreateClosingRequest) error {
	if req.PlanID < 1 {
		return ErrInvalidRequest
	}
	if req.PromotionID != nil && *req.PromotionID < 1 {
		return ErrInvalidRequest
	}
	for _, id := range req.PromotionIDs {
		if id < 1 {
			return ErrInvalidRequest
		}
	}
	if req.UniqueTransferCode != nil && (*req.UniqueTransferCode < 0 || *req.UniqueTransferCode > 999) {
		return ErrInvalidRequest
	}
	if _, err := resolveInteractionChannels(req.InteractionType, req.CallStatus, req.ChatStatus); err != nil {
		return err
	}
	if _, err := parseMoneyToCents(req.DiscountAmount); err != nil {
		return err
	}
	return nil
}

func resolveInteractionChannels(legacyType string, callStatus string, chatStatus string) (string, error) {
	call := strings.TrimSpace(callStatus)
	chat := strings.TrimSpace(chatStatus)
	if call != "" && chat != "" {
		return InteractionCallChat, nil
	}
	if call != "" {
		return InteractionCall, nil
	}
	if chat != "" {
		return InteractionChat, nil
	}
	if strings.TrimSpace(legacyType) == "" {
		return "", ErrInvalidRequest
	}
	return normalizeInteractionType(legacyType)
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
	params.Status = normalizeStatus(params.Status)
	params.Scope = normalizeScope(params.Scope)
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

func normalizeScope(scope string) string {
	switch strings.ToUpper(strings.TrimSpace(scope)) {
	case ScopeDeleted:
		return ScopeDeleted
	case ScopeAll:
		return ScopeAll
	default:
		return ScopeActive
	}
}

func normalizeIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, ErrInvalidRequest
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			return nil, ErrInvalidRequest
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrInvalidRequest
	}
	return out, nil
}

func closingResponses(items []Closing) []ClosingResponse {
	responses := make([]ClosingResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewClosingResponse(item))
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
