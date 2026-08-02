package subscription

import (
	"context"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListOrders(ctx context.Context, actor identity.User, params ListParams) (OrderListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListOrders(ctx, actor, params)
	if err != nil {
		return OrderListResponse{}, err
	}
	return OrderListResponse{Items: orderResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) GetOrder(ctx context.Context, actor identity.User, id int64) (SubscriptionOrderResponse, error) {
	item, err := s.repo.FindOrderByID(ctx, actor, id)
	if err != nil {
		return SubscriptionOrderResponse{}, err
	}
	response := NewOrderResponse(item)
	promotions, err := s.repo.ListOrderPromotions(ctx, id)
	if err != nil {
		return SubscriptionOrderResponse{}, err
	}
	response.Promotions = promotions
	return response, nil
}

func (s *Service) CreateOrder(ctx context.Context, actor identity.User, ownerID int64, req CreateOrderRequest) (OrderCreateResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return OrderCreateResponse{}, ErrForbidden
	}
	if err := validateCreateOrderRequest(req); err != nil {
		return OrderCreateResponse{}, err
	}
	result, err := s.repo.CreateOrder(ctx, actor, ownerID, req)
	if err != nil {
		return OrderCreateResponse{}, err
	}
	response := OrderCreateResponse{Order: NewOrderResponse(result.Order), Subscription: NewSubscriptionResponse(result.Subscription), Period: NewPeriodResponse(result.Period), Idempotent: result.Idempotent}
	promotions, err := s.repo.ListOrderPromotions(ctx, result.Order.ID)
	if err != nil {
		return OrderCreateResponse{}, err
	}
	response.Order.Promotions = promotions
	if result.Reconciliation != nil {
		value := NewReconciliationResponse(*result.Reconciliation)
		response.Reconciliation = &value
	}
	if result.Issue != nil {
		value := NewIssueResponse(*result.Issue)
		response.Issue = &value
	}
	return response, nil
}

func (s *Service) ReconcileOrder(ctx context.Context, actor identity.User, orderID int64, req ManualReconcileRequest) (ReconciliationActionResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return ReconciliationActionResponse{}, ErrForbidden
	}
	if err := validateManualReconcileRequest(req); err != nil {
		return ReconciliationActionResponse{}, err
	}
	result, err := s.repo.ReconcileOrder(ctx, actor, orderID, req)
	if err != nil {
		return ReconciliationActionResponse{}, err
	}
	response := ReconciliationActionResponse{Order: NewOrderResponse(result.Order), Reconciliation: NewReconciliationResponse(result.Reconciliation)}
	if result.Issue != nil {
		value := NewIssueResponse(*result.Issue)
		response.Issue = &value
	}
	return response, nil
}

func (s *Service) ListSubscriptions(ctx context.Context, actor identity.User, params ListParams) (SubscriptionListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListSubscriptions(ctx, actor, params)
	if err != nil {
		return SubscriptionListResponse{}, err
	}
	return SubscriptionListResponse{Items: subscriptionResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) GetSubscription(ctx context.Context, actor identity.User, id int64) (SubscriptionResponse, error) {
	item, err := s.repo.FindSubscriptionByID(ctx, actor, id)
	if err != nil {
		return SubscriptionResponse{}, err
	}
	return NewSubscriptionResponse(item), nil
}

func (s *Service) ListReconciliations(ctx context.Context, actor identity.User, params ListParams) (ReconciliationListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListReconciliations(ctx, actor, params)
	if err != nil {
		return ReconciliationListResponse{}, err
	}
	return ReconciliationListResponse{Items: reconciliationResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) ListIssues(ctx context.Context, actor identity.User, params ListParams) (ReconciliationIssueListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListIssues(ctx, actor, params)
	if err != nil {
		return ReconciliationIssueListResponse{}, err
	}
	return ReconciliationIssueListResponse{Items: issueResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func validateCreateOrderRequest(req CreateOrderRequest) error {
	if req.ClosingID == nil && req.PlanID < 1 {
		return ErrInvalidRequest
	}
	if req.ClosingID != nil && *req.ClosingID < 1 {
		return ErrInvalidRequest
	}
	if req.OutletID != nil && *req.OutletID < 1 {
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
	if _, err := parseMoneyToCents(req.DiscountAmount); err != nil {
		return err
	}
	if _, err := orderIdempotencyKey(req); err != nil {
		return err
	}
	if _, err := subscriptionStartDate(req.SubscriptionStartDate, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func validateManualReconcileRequest(req ManualReconcileRequest) error {
	if req.ClosingID < 1 {
		return ErrInvalidRequest
	}
	if normalizeAction(req.Action) == "" {
		return ErrInvalidAction
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
	params.Status = strings.ToUpper(strings.TrimSpace(params.Status))
	params.IssueType = strings.ToUpper(strings.TrimSpace(params.IssueType))
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

func orderResponses(items []SubscriptionOrder) []SubscriptionOrderResponse {
	responses := make([]SubscriptionOrderResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewOrderResponse(item))
	}
	return responses
}
func subscriptionResponses(items []Subscription) []SubscriptionResponse {
	responses := make([]SubscriptionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewSubscriptionResponse(item))
	}
	return responses
}
func reconciliationResponses(items []Reconciliation) []ReconciliationResponse {
	responses := make([]ReconciliationResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewReconciliationResponse(item))
	}
	return responses
}
func issueResponses(items []ReconciliationIssue) []ReconciliationIssueResponse {
	responses := make([]ReconciliationIssueResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewIssueResponse(item))
	}
	return responses
}
