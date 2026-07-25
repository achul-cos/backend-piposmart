package target

import (
	"context"

	"backend_crm_piposmart/internal/identity"
)

// Service applies role gating and validation around the target Repository.
type Service struct {
	repo *Repository
}

// NewService creates a new target Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func isAdminOrSupervisor(actor identity.User) bool {
	return actor.RoleCode == RoleAdmin || actor.RoleCode == RoleSupervisor
}

// BulkSetTarget sets a default target for every active Sales rep lacking one for the period.
func (s *Service) BulkSetTarget(ctx context.Context, actor identity.User, req BulkSetTargetRequest) (*BulkSetTargetResponse, error) {
	if !isAdminOrSupervisor(actor) {
		return nil, ErrForbidden
	}
	if req.PeriodMonth < 1 || req.PeriodMonth > 12 {
		return nil, ErrInvalidPeriod
	}
	if err := validateDecimal(req.TargetValue); err != nil {
		return nil, err
	}
	resp, err := s.repo.BulkSet(ctx, req, actor.ID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// OverrideTarget upserts a single Sales rep's target, always winning over any bulk value.
func (s *Service) OverrideTarget(ctx context.Context, actor identity.User, salesID int64, req OverrideTargetRequest) (*SalesTargetResponse, error) {
	if !isAdminOrSupervisor(actor) {
		return nil, ErrForbidden
	}
	if req.PeriodMonth < 1 || req.PeriodMonth > 12 {
		return nil, ErrInvalidPeriod
	}
	if err := validateDecimal(req.TargetValue); err != nil {
		return nil, err
	}
	target, err := s.repo.Override(ctx, salesID, req, actor.ID)
	if err != nil {
		return nil, err
	}
	resp := NewSalesTargetResponse(*target)
	return &resp, nil
}

// ListTargets returns targets visible to actor (Sales sees only their own).
func (s *Service) ListTargets(ctx context.Context, actor identity.User, params ListTargetsParams) (*SalesTargetListResponse, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	params.Page = page
	params.Limit = limit

	items, total, err := s.repo.List(ctx, actor.ID, actor.RoleCode, params)
	if err != nil {
		return nil, err
	}
	responses := make([]SalesTargetResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewSalesTargetResponse(item))
	}
	return &SalesTargetListResponse{
		Items: responses,
		Meta:  ListMeta{Page: page, Limit: limit, Total: total},
	}, nil
}
