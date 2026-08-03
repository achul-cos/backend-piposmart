package kpi

import (
	"context"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/jobqueue"
)

// Service applies role gating and validation around the kpi Repository, and enqueues recompute
// jobs through the shared jobqueue.
type Service struct {
	repo *Repository
	jobs *jobqueue.Repository
}

// NewService creates a new kpi Service.
func NewService(repo *Repository, jobs *jobqueue.Repository) *Service {
	return &Service{repo: repo, jobs: jobs}
}

func isAdminOrSupervisor(actor identity.User) bool {
	return actor.RoleCode == RoleAdmin || actor.RoleCode == RoleSupervisor
}

/* ---------- KPI Definition ---------- */

func (s *Service) CreateDefinition(ctx context.Context, actor identity.User, req CreateKpiDefinitionRequest) (*KpiDefinitionResponse, error) {
	if !isAdminOrSupervisor(actor) {
		return nil, ErrForbidden
	}
	if req.PeriodMonth < 1 || req.PeriodMonth > 12 {
		return nil, ErrInvalidPeriod
	}
	if req.ThresholdAchieved == "" {
		req.ThresholdAchieved = "100.00"
	}
	if req.ThresholdNear == "" {
		req.ThresholdNear = "80.00"
	}
	if _, err := parsePercent(req.Weight); err != nil {
		return nil, err
	}
	achieved, err := parsePercent(req.ThresholdAchieved)
	if err != nil {
		return nil, ErrInvalidThreshold
	}
	near, err := parsePercent(req.ThresholdNear)
	if err != nil {
		return nil, ErrInvalidThreshold
	}
	if near > achieved {
		return nil, ErrInvalidThreshold
	}

	id, err := s.repo.CreateDefinition(ctx, req, actor.ID)
	if err != nil {
		return nil, err
	}
	def, err := s.repo.GetDefinitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := NewKpiDefinitionResponse(*def)
	return &resp, nil
}

func (s *Service) ListDefinitions(ctx context.Context, periodYear, periodMonth *int, activeOnly bool) (*KpiDefinitionListResponse, error) {
	items, err := s.repo.ListDefinitions(ctx, periodYear, periodMonth, activeOnly)
	if err != nil {
		return nil, err
	}
	responses := make([]KpiDefinitionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewKpiDefinitionResponse(item))
	}
	return &KpiDefinitionListResponse{Items: responses}, nil
}

func (s *Service) GetDefinition(ctx context.Context, id int64) (*KpiDefinitionResponse, error) {
	def, err := s.repo.GetDefinitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := NewKpiDefinitionResponse(*def)
	return &resp, nil
}

func (s *Service) DeactivateDefinition(ctx context.Context, actor identity.User, id int64) error {
	if !isAdminOrSupervisor(actor) {
		return ErrForbidden
	}
	return s.repo.DeactivateDefinition(ctx, id)
}

/* ---------- Recompute ---------- */

func (s *Service) TriggerRecompute(ctx context.Context, actor identity.User, req RecomputeRequest) (*JobResponse, error) {
	if !isAdminOrSupervisor(actor) {
		return nil, ErrForbidden
	}
	if req.PeriodMonth < 1 || req.PeriodMonth > 12 {
		return nil, ErrInvalidPeriod
	}
	actorID := actor.ID
	jobID, err := s.jobs.Enqueue(ctx, JobTypeRecompute, RecomputeJobPayload{
		PeriodYear:  req.PeriodYear,
		PeriodMonth: req.PeriodMonth,
	}, &actorID)
	if err != nil {
		return nil, err
	}
	return s.GetJob(ctx, jobID)
}

func (s *Service) GetJob(ctx context.Context, jobID int64) (*JobResponse, error) {
	job, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrJobNotFound
	}
	resp := &JobResponse{
		ID:          job.ID,
		JobType:     job.JobType,
		Status:      job.Status,
		Attempts:    job.Attempts,
		MaxAttempts: job.MaxAttempts,
		CreatedAt:   job.CreatedAt,
	}
	if job.LastError.Valid {
		resp.LastError = &job.LastError.String
	}
	if job.CompletedAt.Valid {
		resp.CompletedAt = &job.CompletedAt.Time
	}
	return resp, nil
}

/* ---------- Results & Ranking ---------- */

func (s *Service) ListResults(ctx context.Context, actor identity.User, params ListResultsParams) (*SalesKpiResultListResponse, error) {
	items, err := s.repo.ListResults(ctx, actor.ID, actor.RoleCode, params)
	if err != nil {
		return nil, err
	}
	responses := make([]SalesKpiResultResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewSalesKpiResultResponse(item))
	}
	return &SalesKpiResultListResponse{Items: responses}, nil
}

func (s *Service) ListRanking(ctx context.Context, actor identity.User, periodYear, periodMonth int) (*SalesKpiResultListResponse, error) {
	items, err := s.repo.ListRanking(ctx, periodYear, periodMonth)
	if err != nil {
		return nil, err
	}
	responses := make([]SalesKpiResultResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewSalesKpiResultResponse(item))
	}
	return &SalesKpiResultListResponse{Items: responses}, nil
}
