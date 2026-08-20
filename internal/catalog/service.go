package catalog

import (
	"context"
	"regexp"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

var decimalPattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListPackages(ctx context.Context, params ListParams) (PackageListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListPackages(ctx, params)
	if err != nil {
		return PackageListResponse{}, err
	}
	return PackageListResponse{Items: packageResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) CreatePackage(ctx context.Context, actor identity.User, req CreatePackageRequest) (PackageResponse, error) {
	if !canManageCatalog(actor) {
		return PackageResponse{}, ErrForbidden
	}
	item, err := s.repo.CreatePackage(ctx, req)
	if err != nil {
		return PackageResponse{}, err
	}
	return NewPackageResponse(item), nil
}

func (s *Service) GetPackage(ctx context.Context, id int64) (PackageResponse, error) {
	item, err := s.repo.FindPackageByID(ctx, id)
	if err != nil {
		return PackageResponse{}, err
	}
	return NewPackageResponse(item), nil
}

func (s *Service) UpdatePackage(ctx context.Context, actor identity.User, id int64, req UpdatePackageRequest) (PackageResponse, error) {
	if !canManageCatalog(actor) {
		return PackageResponse{}, ErrForbidden
	}
	item, err := s.repo.UpdatePackage(ctx, id, req)
	if err != nil {
		return PackageResponse{}, err
	}
	return NewPackageResponse(item), nil
}

func (s *Service) DeletePackages(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.DeletePackages(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) RestorePackages(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.RestorePackages(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) ForceDeletePackages(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.ForceDeletePackages(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) ListPlans(ctx context.Context, params ListParams) (PlanListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListPlans(ctx, params)
	if err != nil {
		return PlanListResponse{}, err
	}
	return PlanListResponse{Items: planResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) CreatePlan(ctx context.Context, actor identity.User, req CreatePlanRequest) (PlanResponse, error) {
	if !canManageCatalog(actor) {
		return PlanResponse{}, ErrForbidden
	}
	if err := validateCreatePlan(req); err != nil {
		return PlanResponse{}, err
	}
	item, err := s.repo.CreatePlan(ctx, req)
	if err != nil {
		return PlanResponse{}, err
	}
	return NewPlanResponse(item), nil
}

func (s *Service) GetPlan(ctx context.Context, id int64) (PlanResponse, error) {
	item, err := s.repo.FindPlanByID(ctx, id)
	if err != nil {
		return PlanResponse{}, err
	}
	return NewPlanResponse(item), nil
}

func (s *Service) UpdatePlan(ctx context.Context, actor identity.User, id int64, req UpdatePlanRequest) (PlanResponse, error) {
	if !canManageCatalog(actor) {
		return PlanResponse{}, ErrForbidden
	}
	if err := validateUpdatePlan(req); err != nil {
		return PlanResponse{}, err
	}
	item, err := s.repo.UpdatePlan(ctx, id, req)
	if err != nil {
		return PlanResponse{}, err
	}
	return NewPlanResponse(item), nil
}

func (s *Service) DeletePlans(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.DeletePlans(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) RestorePlans(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.RestorePlans(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) ForceDeletePlans(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.ForceDeletePlans(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) ListPromotions(ctx context.Context, params ListParams) (PromotionListResponse, error) {
	params = normalizeListParams(params)
	items, total, err := s.repo.ListPromotions(ctx, params)
	if err != nil {
		return PromotionListResponse{}, err
	}
	return PromotionListResponse{Items: promotionResponses(items), Pagination: PaginationMeta{Page: params.Page, Limit: resolveReturnedLimit(params.All, params.Limit, len(items), total), Total: total}}, nil
}

func (s *Service) CreatePromotion(ctx context.Context, actor identity.User, req CreatePromotionRequest) (PromotionResponse, error) {
	if !canManageCatalog(actor) {
		return PromotionResponse{}, ErrForbidden
	}
	if err := validateCreatePromotion(req); err != nil {
		return PromotionResponse{}, err
	}
	item, err := s.repo.CreatePromotion(ctx, req)
	if err != nil {
		return PromotionResponse{}, err
	}
	return NewPromotionResponse(item), nil
}

func (s *Service) GetPromotion(ctx context.Context, id int64) (PromotionResponse, error) {
	item, err := s.repo.FindPromotionByID(ctx, id)
	if err != nil {
		return PromotionResponse{}, err
	}
	return NewPromotionResponse(item), nil
}

func (s *Service) UpdatePromotion(ctx context.Context, actor identity.User, id int64, req UpdatePromotionRequest) (PromotionResponse, error) {
	if !canManageCatalog(actor) {
		return PromotionResponse{}, ErrForbidden
	}
	if err := validateUpdatePromotion(req); err != nil {
		return PromotionResponse{}, err
	}
	item, err := s.repo.UpdatePromotion(ctx, id, req)
	if err != nil {
		return PromotionResponse{}, err
	}
	return NewPromotionResponse(item), nil
}

func (s *Service) DeletePromotions(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.DeletePromotions(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) RestorePromotions(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.RestorePromotions(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) ForceDeletePromotions(ctx context.Context, actor identity.User, ids []int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.ForceDeletePromotions(ctx, ids)
	return BulkActionResponse{IDs: ids, Affected: affected}, err
}

func (s *Service) ListBenefits(ctx context.Context, promotionID int64) ([]BenefitResponse, error) {
	items, err := s.repo.ListBenefits(ctx, promotionID)
	if err != nil {
		return nil, err
	}
	return benefitResponses(items), nil
}

func (s *Service) CreateBenefit(ctx context.Context, actor identity.User, promotionID int64, req CreateBenefitRequest) (BenefitResponse, error) {
	if !canManageCatalog(actor) {
		return BenefitResponse{}, ErrForbidden
	}
	item, err := s.repo.CreateBenefit(ctx, promotionID, req)
	if err != nil {
		return BenefitResponse{}, err
	}
	return NewBenefitResponse(item), nil
}

func (s *Service) DeleteBenefit(ctx context.Context, actor identity.User, promotionID, benefitID int64) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	affected, err := s.repo.DeleteBenefit(ctx, promotionID, benefitID)
	return BulkActionResponse{IDs: []int64{benefitID}, Affected: affected}, err
}

func (s *Service) SetEligibility(ctx context.Context, actor identity.User, promotionID int64, req SetEligibilityRequest) (PromotionResponse, error) {
	if !canManageCatalog(actor) {
		return PromotionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(req.PlanIDs)
	if err != nil {
		return PromotionResponse{}, err
	}
	if err := s.repo.SetEligibility(ctx, promotionID, ids); err != nil {
		return PromotionResponse{}, err
	}
	return s.GetPromotion(ctx, promotionID)
}

func (s *Service) EligiblePlans(ctx context.Context, promotionID int64) (EligiblePlanResponse, error) {
	items, err := s.repo.EligiblePlans(ctx, promotionID)
	if err != nil {
		return EligiblePlanResponse{}, err
	}
	return EligiblePlanResponse{Items: planResponses(items)}, nil
}

func (s *Service) EligiblePromotions(ctx context.Context, planID int64, asOf *time.Time) (EligiblePromotionResponse, error) {
	date := time.Now().UTC()
	if asOf != nil {
		date = *asOf
	}
	items, err := s.repo.EligiblePromotions(ctx, planID, date)
	if err != nil {
		return EligiblePromotionResponse{}, err
	}
	responses := promotionResponses(items)
	var recommended *PromotionResponse
	for i := range responses {
		if responses[i].ChargeType == ChargeFree {
			recommended = &responses[i]
			break
		}
	}
	return EligiblePromotionResponse{Items: responses, Recommended: recommended}, nil
}

func validateCreatePlan(req CreatePlanRequest) error {
	if req.TenureMonths < 1 {
		return ErrInvalidTenure
	}
	if !decimalPattern.MatchString(strings.TrimSpace(req.Price)) {
		return ErrInvalidDecimal
	}
	if _, err := parseDate(req.EffectiveFrom); err != nil {
		return err
	}
	if _, err := parseNullDate(req.EffectiveTo); err != nil {
		return err
	}
	return nil
}

func validateUpdatePlan(req UpdatePlanRequest) error {
	if req.TenureMonths != nil && *req.TenureMonths < 1 {
		return ErrInvalidTenure
	}
	if req.Price != nil && !decimalPattern.MatchString(strings.TrimSpace(*req.Price)) {
		return ErrInvalidDecimal
	}
	if req.EffectiveFrom != nil {
		if _, err := parseDate(*req.EffectiveFrom); err != nil {
			return err
		}
	}
	if req.EffectiveTo != nil {
		if _, err := parseNullDate(req.EffectiveTo); err != nil {
			return err
		}
	}
	return nil
}

func validateCreatePromotion(req CreatePromotionRequest) error {
	if err := validateChargeType(req.ChargeType); err != nil {
		return err
	}
	if strings.TrimSpace(req.AdditionalCharge) != "" && !decimalPattern.MatchString(strings.TrimSpace(req.AdditionalCharge)) {
		return ErrInvalidDecimal
	}
	if _, err := parseDate(req.EffectiveFrom); err != nil {
		return err
	}
	if _, err := parseNullDate(req.EffectiveTo); err != nil {
		return err
	}
	return nil
}

func validateUpdatePromotion(req UpdatePromotionRequest) error {
	if req.ChargeType != nil {
		if err := validateChargeType(*req.ChargeType); err != nil {
			return err
		}
	}
	if req.AdditionalCharge != nil && !decimalPattern.MatchString(strings.TrimSpace(*req.AdditionalCharge)) {
		return ErrInvalidDecimal
	}
	if req.EffectiveFrom != nil {
		if _, err := parseDate(*req.EffectiveFrom); err != nil {
			return err
		}
	}
	if req.EffectiveTo != nil {
		if _, err := parseNullDate(req.EffectiveTo); err != nil {
			return err
		}
	}
	return nil
}

func validateChargeType(value string) error {
	value = normalizeCode(value)
	if value != ChargeFree && value != ChargePaid {
		return ErrInvalidCharge
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
		if params.Limit > 10000 {
			params.Limit = 10000
		}
	}
	params.Query = strings.TrimSpace(params.Query)
	params.ChargeType = normalizeCode(params.ChargeType)
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

func canManageCatalog(actor identity.User) bool {
	return actor.RoleCode == RoleAdmin
}

func packageResponses(items []Package) []PackageResponse {
	responses := make([]PackageResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewPackageResponse(item))
	}
	return responses
}

func planResponses(items []Plan) []PlanResponse {
	responses := make([]PlanResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewPlanResponse(item))
	}
	return responses
}

func promotionResponses(items []Promotion) []PromotionResponse {
	responses := make([]PromotionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewPromotionResponse(item))
	}
	return responses
}

func benefitResponses(items []Benefit) []BenefitResponse {
	responses := make([]BenefitResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, NewBenefitResponse(item))
	}
	return responses
}

func normalizeIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, ErrEmptyBulk
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			return nil, ErrEmptyBulk
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrEmptyBulk
	}
	return out, nil
}
