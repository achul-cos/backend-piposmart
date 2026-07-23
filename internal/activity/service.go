package activity

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

func (s *Service) ListInteractions(ctx context.Context, actor identity.User, params InteractionListParams) (InteractionListResponse, error) {
	params = normalizeInteractionListParams(params)
	items, total, err := s.repo.ListInteractions(ctx, actor, params)
	if err != nil {
		return InteractionListResponse{}, err
	}
	return InteractionListResponse{
		Items:      interactionResponses(items),
		Pagination: PaginationMeta{Page: params.Page, Limit: params.Limit, Total: total},
	}, nil
}

func (s *Service) ListLeadInteractions(ctx context.Context, actor identity.User, leadID int64, params InteractionListParams) (InteractionListResponse, error) {
	params.LeadID = &leadID
	return s.ListInteractions(ctx, actor, params)
}

func (s *Service) CreateInteraction(ctx context.Context, actor identity.User, leadID int64, req CreateInteractionRequest) (CustomerInteractionResponse, error) {
	if err := validateInteractionRequest(req); err != nil {
		return CustomerInteractionResponse{}, err
	}
	item, err := s.repo.CreateInteraction(ctx, actor, leadID, req)
	if err != nil {
		return CustomerInteractionResponse{}, err
	}
	return NewInteractionResponse(item), nil
}

func (s *Service) StageHistory(ctx context.Context, actor identity.User, leadID int64) (StageHistoryListResponse, error) {
	items, err := s.repo.StageHistory(ctx, actor, leadID)
	if err != nil {
		return StageHistoryListResponse{}, err
	}
	responses := make([]StageHistoryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewStageHistoryResponse(item))
	}
	return StageHistoryListResponse{Items: responses, Total: len(responses)}, nil
}

func (s *Service) ListTrainings(ctx context.Context, actor identity.User, params TrainingListParams) (TrainingListResponse, error) {
	params = normalizeTrainingListParams(params)
	items, total, err := s.repo.ListTrainings(ctx, actor, params)
	if err != nil {
		return TrainingListResponse{}, err
	}
	return TrainingListResponse{
		Items:      trainingResponses(items),
		Pagination: PaginationMeta{Page: params.Page, Limit: params.Limit, Total: total},
	}, nil
}

func (s *Service) ListLeadTrainings(ctx context.Context, actor identity.User, leadID int64, params TrainingListParams) (TrainingListResponse, error) {
	params.LeadID = &leadID
	return s.ListTrainings(ctx, actor, params)
}

func (s *Service) ScheduleTraining(ctx context.Context, actor identity.User, leadID int64, req ScheduleTrainingRequest) (TrainingReportResponse, error) {
	if err := validateScheduleTrainingRequest(req); err != nil {
		return TrainingReportResponse{}, err
	}
	item, err := s.repo.ScheduleTraining(ctx, actor, leadID, req)
	if err != nil {
		return TrainingReportResponse{}, err
	}
	return NewTrainingReportResponse(item), nil
}

func (s *Service) RescheduleTraining(ctx context.Context, actor identity.User, trainingID int64, req RescheduleTrainingRequest) (TrainingReportResponse, error) {
	if req.ScheduledAt.IsZero() {
		return TrainingReportResponse{}, ErrInvalidTransition
	}
	item, err := s.repo.RescheduleTraining(ctx, actor, trainingID, req)
	if err != nil {
		return TrainingReportResponse{}, err
	}
	return NewTrainingReportResponse(item), nil
}

func (s *Service) CompleteTraining(ctx context.Context, actor identity.User, trainingID int64, req CompleteTrainingRequest) (TrainingReportResponse, error) {
	item, err := s.repo.CompleteTraining(ctx, actor, trainingID, req)
	if err != nil {
		return TrainingReportResponse{}, err
	}
	return NewTrainingReportResponse(item), nil
}

func (s *Service) CancelTraining(ctx context.Context, actor identity.User, trainingID int64, req CancelTrainingRequest) (TrainingReportResponse, error) {
	item, err := s.repo.CancelTraining(ctx, actor, trainingID, req)
	if err != nil {
		return TrainingReportResponse{}, err
	}
	return NewTrainingReportResponse(item), nil
}

func validateInteractionRequest(req CreateInteractionRequest) error {
	if _, err := normalizeInteractionType(req.Type); err != nil {
		return err
	}
	if req.RemarkScore != nil && (*req.RemarkScore < 0 || *req.RemarkScore > 3) {
		return ErrInvalidScore
	}
	if req.DurationSeconds != nil && *req.DurationSeconds < 0 {
		return ErrInvalidTransition
	}
	return nil
}

func validateScheduleTrainingRequest(req ScheduleTrainingRequest) error {
	if _, err := normalizeTrainingType(req.TrainingType); err != nil {
		return err
	}
	if req.ScheduledAt.IsZero() {
		return ErrInvalidTransition
	}
	return nil
}

func normalizeInteractionListParams(params InteractionListParams) InteractionListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	params.Type = strings.ToUpper(strings.TrimSpace(params.Type))
	params.Sort = strings.TrimSpace(params.Sort)
	return params
}

func normalizeTrainingListParams(params TrainingListParams) TrainingListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	params.Status = strings.ToUpper(strings.TrimSpace(params.Status))
	params.TrainingType = strings.ToUpper(strings.TrimSpace(params.TrainingType))
	params.Sort = strings.TrimSpace(params.Sort)
	return params
}

func interactionResponses(items []CustomerInteraction) []CustomerInteractionResponse {
	responses := make([]CustomerInteractionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewInteractionResponse(item))
	}
	return responses
}

func trainingResponses(items []TrainingReport) []TrainingReportResponse {
	responses := make([]TrainingReportResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewTrainingReportResponse(item))
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
