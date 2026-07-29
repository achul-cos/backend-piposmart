package lead

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

func (s *Service) ListLeads(ctx context.Context, actor identity.User, params ListParams) (LeadListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListLeads(ctx, actor, params)
	if err != nil {
		return LeadListResponse{}, err
	}
	return LeadListResponse{
		Items:      leadResponses(items),
		Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total},
	}, nil
}

func (s *Service) CreateLead(ctx context.Context, actor identity.User, req CreateLeadRequest) (LeadResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return LeadResponse{}, ErrForbidden
	}
	item, err := s.repo.CreateLead(ctx, actor, req)
	if err != nil {
		return LeadResponse{}, err
	}
	return NewLeadResponse(item), nil
}

func (s *Service) GetLead(ctx context.Context, actor identity.User, id int64) (LeadResponse, error) {
	item, err := s.repo.FindLeadByID(ctx, actor, id)
	if err != nil {
		return LeadResponse{}, err
	}
	return NewLeadResponse(item), nil
}

func (s *Service) AssignmentHistory(ctx context.Context, actor identity.User, id int64) (AssignmentHistoryListResponse, error) {
	items, err := s.repo.AssignmentHistory(ctx, actor, id)
	if err != nil {
		return AssignmentHistoryListResponse{}, err
	}
	responses := make([]AssignmentHistoryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewAssignmentHistoryResponse(item))
	}
	return AssignmentHistoryListResponse{Items: responses, Total: len(responses)}, nil
}

func (s *Service) AssignSupervisor(ctx context.Context, actor identity.User, leadID int64, req AssignSupervisorRequest) (LeadBulkResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return LeadBulkResponse{}, ErrForbidden
	}
	return s.assignSupervisor(ctx, actor, []int64{leadID}, req.SupervisorID, req.Reason)
}

func (s *Service) BulkAssignSupervisor(ctx context.Context, actor identity.User, req BulkAssignSupervisorRequest) (LeadBulkResponse, error) {
	if actor.RoleCode != RoleAdmin {
		return LeadBulkResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(req.LeadIDs)
	if err != nil {
		return LeadBulkResponse{}, err
	}
	return s.assignSupervisor(ctx, actor, ids, req.SupervisorID, req.Reason)
}

func (s *Service) assignSupervisor(ctx context.Context, actor identity.User, ids []int64, supervisorID int64, reason string) (LeadBulkResponse, error) {
	items, err := s.repo.AssignSupervisor(ctx, actor, ids, supervisorID, reason)
	if err != nil {
		return LeadBulkResponse{}, err
	}
	return LeadBulkResponse{Items: leadResponses(items), Total: len(items)}, nil
}

func (s *Service) AssignSales(ctx context.Context, actor identity.User, leadID int64, req AssignSalesRequest) (LeadBulkResponse, error) {
	if actor.RoleCode != RoleSupervisor {
		return LeadBulkResponse{}, ErrForbidden
	}
	return s.assignSales(ctx, actor, []int64{leadID}, req.SalesID, req.Reason)
}

func (s *Service) BulkAssignSales(ctx context.Context, actor identity.User, req BulkAssignSalesRequest) (LeadBulkResponse, error) {
	if actor.RoleCode != RoleSupervisor {
		return LeadBulkResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(req.LeadIDs)
	if err != nil {
		return LeadBulkResponse{}, err
	}
	return s.assignSales(ctx, actor, ids, req.SalesID, req.Reason)
}

func (s *Service) assignSales(ctx context.Context, actor identity.User, ids []int64, salesID int64, reason string) (LeadBulkResponse, error) {
	items, err := s.repo.AssignSales(ctx, actor, ids, salesID, reason)
	if err != nil {
		return LeadBulkResponse{}, err
	}
	return LeadBulkResponse{Items: leadResponses(items), Total: len(items)}, nil
}

func (s *Service) Release(ctx context.Context, actor identity.User, leadID int64, req ReleaseRequest) (LeadBulkResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return LeadBulkResponse{}, ErrForbidden
	}
	return s.release(ctx, actor, []int64{leadID}, nullableID(req.AdminID), req.Reason)
}

func (s *Service) BulkRelease(ctx context.Context, actor identity.User, req BulkReleaseRequest) (LeadBulkResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return LeadBulkResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(req.LeadIDs)
	if err != nil {
		return LeadBulkResponse{}, err
	}
	return s.release(ctx, actor, ids, nullableID(req.AdminID), req.Reason)
}

func (s *Service) release(ctx context.Context, actor identity.User, ids []int64, adminID sql.NullInt64, reason string) (LeadBulkResponse, error) {
	items, err := s.repo.ReleaseToAdmin(ctx, actor, ids, adminID, reason)
	if err != nil {
		return LeadBulkResponse{}, err
	}
	return LeadBulkResponse{Items: leadResponses(items), Total: len(items)}, nil
}

func (s *Service) MarkInvalid(ctx context.Context, actor identity.User, leadID int64, req MarkInvalidRequest) (LeadResponse, error) {
	if actor.RoleCode != RoleSales {
		return LeadResponse{}, ErrForbidden
	}
	item, err := s.repo.MarkInvalid(ctx, actor, leadID, req.Reason)
	if err != nil {
		return LeadResponse{}, err
	}
	return NewLeadResponse(item), nil
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
	params.Ownership = strings.TrimSpace(params.Ownership)
	params.Stage = strings.TrimSpace(params.Stage)
	params.Status = strings.TrimSpace(params.Status)
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

func leadResponses(items []Lead) []LeadResponse {
	responses := make([]LeadResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewLeadResponse(item))
	}
	return responses
}

func nullableID(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
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
