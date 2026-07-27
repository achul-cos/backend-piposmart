package catalog

import (
	"context"
	"errors"

	"backend_crm_piposmart/internal/identity"
)

func (s *Service) ListPackageHistories(ctx context.Context, id int64) (HistoryListResponse, error) {
	if _, err := s.repo.FindPackageByIDAny(ctx, id); err != nil {
		return HistoryListResponse{}, err
	}
	records, err := s.repo.ListHistories(ctx, entityTypeCatalogPackage, id)
	if err != nil {
		return HistoryListResponse{}, err
	}
	return historyResponses(records), nil
}

func (s *Service) ListPlanHistories(ctx context.Context, id int64) (HistoryListResponse, error) {
	if _, err := s.repo.FindPlanByIDAny(ctx, id); err != nil {
		return HistoryListResponse{}, err
	}
	records, err := s.repo.ListHistories(ctx, entityTypeCatalogPlan, id)
	if err != nil {
		return HistoryListResponse{}, err
	}
	return historyResponses(records), nil
}

func (s *Service) ListPromotionHistories(ctx context.Context, id int64) (HistoryListResponse, error) {
	if _, err := s.repo.FindPromotionByIDAny(ctx, id); err != nil {
		return HistoryListResponse{}, err
	}
	records, err := s.repo.ListHistories(ctx, entityTypeCatalogPromotion, id)
	if err != nil {
		return HistoryListResponse{}, err
	}
	return historyResponses(records), nil
}

func (s *Service) CreatePackageWithMeta(ctx context.Context, actor identity.User, req CreatePackageRequest, meta RequestMeta) (PackageResponse, error) {
	if !canManageCatalog(actor) {
		return PackageResponse{}, ErrForbidden
	}
	item, err := s.repo.CreatePackage(ctx, req)
	if err != nil {
		return PackageResponse{}, err
	}
	response := NewPackageResponse(item)
	_ = s.auditCatalogChange(ctx, actor, "catalog.package.create", entityTypeCatalogPackage, item.ID, nil, response, meta)
	return response, nil
}

func (s *Service) UpdatePackageWithMeta(ctx context.Context, actor identity.User, id int64, req UpdatePackageRequest, meta RequestMeta) (PackageResponse, error) {
	if !canManageCatalog(actor) {
		return PackageResponse{}, ErrForbidden
	}
	before, err := s.packageSnapshot(ctx, id)
	if err != nil {
		return PackageResponse{}, err
	}
	after, err := s.repo.UpdatePackage(ctx, id, req)
	if err != nil {
		return PackageResponse{}, err
	}
	response := NewPackageResponse(after)
	_ = s.auditCatalogChange(ctx, actor, "catalog.package.update", entityTypeCatalogPackage, id, before, response, meta)
	return response, nil
}

func (s *Service) DeletePackagesWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPackageBulkAction(ctx, actor, ids, meta, "catalog.package.soft_delete", s.repo.DeletePackages, true)
}

func (s *Service) RestorePackagesWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPackageBulkAction(ctx, actor, ids, meta, "catalog.package.restore", s.repo.RestorePackages, true)
}

func (s *Service) ForceDeletePackagesWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPackageBulkAction(ctx, actor, ids, meta, "catalog.package.force_delete", s.repo.ForceDeletePackages, false)
}

func (s *Service) CreatePlanWithMeta(ctx context.Context, actor identity.User, req CreatePlanRequest, meta RequestMeta) (PlanResponse, error) {
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
	response := NewPlanResponse(item)
	_ = s.auditCatalogChange(ctx, actor, "catalog.plan.create", entityTypeCatalogPlan, item.ID, nil, response, meta)
	return response, nil
}

func (s *Service) UpdatePlanWithMeta(ctx context.Context, actor identity.User, id int64, req UpdatePlanRequest, meta RequestMeta) (PlanResponse, error) {
	if !canManageCatalog(actor) {
		return PlanResponse{}, ErrForbidden
	}
	if err := validateUpdatePlan(req); err != nil {
		return PlanResponse{}, err
	}
	before, err := s.planSnapshot(ctx, id)
	if err != nil {
		return PlanResponse{}, err
	}
	after, err := s.repo.UpdatePlan(ctx, id, req)
	if err != nil {
		return PlanResponse{}, err
	}
	response := NewPlanResponse(after)
	_ = s.auditCatalogChange(ctx, actor, "catalog.plan.update", entityTypeCatalogPlan, id, before, response, meta)
	return response, nil
}

func (s *Service) DeletePlansWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPlanBulkAction(ctx, actor, ids, meta, "catalog.plan.soft_delete", s.repo.DeletePlans, true)
}

func (s *Service) RestorePlansWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPlanBulkAction(ctx, actor, ids, meta, "catalog.plan.restore", s.repo.RestorePlans, true)
}

func (s *Service) ForceDeletePlansWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPlanBulkAction(ctx, actor, ids, meta, "catalog.plan.force_delete", s.repo.ForceDeletePlans, false)
}

func (s *Service) CreatePromotionWithMeta(ctx context.Context, actor identity.User, req CreatePromotionRequest, meta RequestMeta) (PromotionResponse, error) {
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
	response := NewPromotionResponse(item)
	after, err := s.promotionAuditSnapshot(ctx, item.ID)
	if err != nil {
		return response, nil
	}
	_ = s.auditCatalogChange(ctx, actor, "catalog.promotion.create", entityTypeCatalogPromotion, item.ID, nil, after, meta)
	return response, nil
}

func (s *Service) UpdatePromotionWithMeta(ctx context.Context, actor identity.User, id int64, req UpdatePromotionRequest, meta RequestMeta) (PromotionResponse, error) {
	if !canManageCatalog(actor) {
		return PromotionResponse{}, ErrForbidden
	}
	if err := validateUpdatePromotion(req); err != nil {
		return PromotionResponse{}, err
	}
	before, err := s.promotionAuditSnapshot(ctx, id)
	if err != nil {
		return PromotionResponse{}, err
	}
	afterPromotion, err := s.repo.UpdatePromotion(ctx, id, req)
	if err != nil {
		return PromotionResponse{}, err
	}
	response := NewPromotionResponse(afterPromotion)
	after, err := s.promotionAuditSnapshot(ctx, id)
	if err == nil {
		_ = s.auditCatalogChange(ctx, actor, "catalog.promotion.update", entityTypeCatalogPromotion, id, before, after, meta)
	}
	return response, nil
}

func (s *Service) DeletePromotionsWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPromotionBulkAction(ctx, actor, ids, meta, "catalog.promotion.soft_delete", s.repo.DeletePromotions, true)
}

func (s *Service) RestorePromotionsWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPromotionBulkAction(ctx, actor, ids, meta, "catalog.promotion.restore", s.repo.RestorePromotions, true)
}

func (s *Service) ForceDeletePromotionsWithMeta(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta) (BulkActionResponse, error) {
	return s.auditPromotionBulkAction(ctx, actor, ids, meta, "catalog.promotion.force_delete", s.repo.ForceDeletePromotions, false)
}

func (s *Service) CreateBenefitWithMeta(ctx context.Context, actor identity.User, promotionID int64, req CreateBenefitRequest, meta RequestMeta) (BenefitResponse, error) {
	if !canManageCatalog(actor) {
		return BenefitResponse{}, ErrForbidden
	}
	before, err := s.promotionAuditSnapshot(ctx, promotionID)
	if err != nil {
		return BenefitResponse{}, err
	}
	item, err := s.repo.CreateBenefit(ctx, promotionID, req)
	if err != nil {
		return BenefitResponse{}, err
	}
	after, err := s.promotionAuditSnapshot(ctx, promotionID)
	if err == nil {
		_ = s.auditCatalogChange(ctx, actor, "catalog.promotion.add_benefit", entityTypeCatalogPromotion, promotionID, before, after, meta)
	}
	return NewBenefitResponse(item), nil
}

func (s *Service) DeleteBenefitWithMeta(ctx context.Context, actor identity.User, promotionID, benefitID int64, meta RequestMeta) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	before, err := s.promotionAuditSnapshot(ctx, promotionID)
	if err != nil {
		return BulkActionResponse{}, err
	}
	affected, err := s.repo.DeleteBenefit(ctx, promotionID, benefitID)
	if err != nil {
		return BulkActionResponse{}, err
	}
	after, err := s.promotionAuditSnapshot(ctx, promotionID)
	if err == nil {
		_ = s.auditCatalogChange(ctx, actor, "catalog.promotion.delete_benefit", entityTypeCatalogPromotion, promotionID, before, after, meta)
	}
	return BulkActionResponse{IDs: []int64{benefitID}, Affected: affected}, nil
}

func (s *Service) SetEligibilityWithMeta(ctx context.Context, actor identity.User, promotionID int64, req SetEligibilityRequest, meta RequestMeta) (PromotionResponse, error) {
	if !canManageCatalog(actor) {
		return PromotionResponse{}, ErrForbidden
	}
	ids, err := normalizeIDs(req.PlanIDs)
	if err != nil {
		return PromotionResponse{}, err
	}
	before, err := s.promotionAuditSnapshot(ctx, promotionID)
	if err != nil {
		return PromotionResponse{}, err
	}
	if err := s.repo.SetEligibility(ctx, promotionID, ids); err != nil {
		return PromotionResponse{}, err
	}
	response, err := s.GetPromotion(ctx, promotionID)
	if err != nil {
		return PromotionResponse{}, err
	}
	after, err := s.promotionAuditSnapshot(ctx, promotionID)
	if err == nil {
		_ = s.auditCatalogChange(ctx, actor, "catalog.promotion.set_eligibility", entityTypeCatalogPromotion, promotionID, before, after, meta)
	}
	return response, nil
}

func (s *Service) auditCatalogChange(ctx context.Context, actor identity.User, action, entityType string, entityID int64, before, after any, meta RequestMeta) error {
	return s.repo.Audit(ctx, CatalogAuditEntry{
		ActorUserID: actor.ID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Before:      before,
		After:       after,
		IPAddress:   meta.IPAddress,
		UserAgent:   meta.UserAgent,
		RequestID:   meta.RequestID,
	})
}

func (s *Service) packageSnapshot(ctx context.Context, id int64) (PackageResponse, error) {
	item, err := s.repo.FindPackageByIDAny(ctx, id)
	if err != nil {
		return PackageResponse{}, err
	}
	return NewPackageResponse(item), nil
}

func (s *Service) planSnapshot(ctx context.Context, id int64) (PlanResponse, error) {
	item, err := s.repo.FindPlanByIDAny(ctx, id)
	if err != nil {
		return PlanResponse{}, err
	}
	return NewPlanResponse(item), nil
}

func (s *Service) promotionAuditSnapshot(ctx context.Context, id int64) (PromotionAuditSnapshot, error) {
	item, err := s.repo.FindPromotionByIDAny(ctx, id)
	if err != nil {
		return PromotionAuditSnapshot{}, err
	}
	eligiblePlanIDs, err := s.repo.ListEligiblePlanIDs(ctx, id)
	if err != nil {
		return PromotionAuditSnapshot{}, err
	}
	return PromotionAuditSnapshot{Promotion: NewPromotionResponse(item), EligiblePlanIDs: eligiblePlanIDs}, nil
}

func (s *Service) auditPackageBulkAction(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta, action string, run func(context.Context, []int64) (int64, error), captureAfter bool) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	normalized, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	before := make(map[int64]PackageResponse, len(normalized))
	for _, id := range normalized {
		if snapshot, err := s.packageSnapshot(ctx, id); err == nil {
			before[id] = snapshot
		}
	}
	affected, err := run(ctx, normalized)
	if err != nil {
		return BulkActionResponse{}, err
	}
	for _, id := range normalized {
		var beforeValue any
		if snapshot, ok := before[id]; ok {
			beforeValue = snapshot
		}
		var afterValue any
		if captureAfter {
			if snapshot, err := s.packageSnapshot(ctx, id); err == nil {
				afterValue = snapshot
			}
		}
		if beforeValue == nil && afterValue == nil {
			continue
		}
		_ = s.auditCatalogChange(ctx, actor, action, entityTypeCatalogPackage, id, beforeValue, afterValue, meta)
	}
	return BulkActionResponse{IDs: normalized, Affected: affected}, nil
}

func (s *Service) auditPlanBulkAction(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta, action string, run func(context.Context, []int64) (int64, error), captureAfter bool) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	normalized, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	before := make(map[int64]PlanResponse, len(normalized))
	for _, id := range normalized {
		if snapshot, err := s.planSnapshot(ctx, id); err == nil {
			before[id] = snapshot
		}
	}
	affected, err := run(ctx, normalized)
	if err != nil {
		return BulkActionResponse{}, err
	}
	for _, id := range normalized {
		var beforeValue any
		if snapshot, ok := before[id]; ok {
			beforeValue = snapshot
		}
		var afterValue any
		if captureAfter {
			if snapshot, err := s.planSnapshot(ctx, id); err == nil {
				afterValue = snapshot
			}
		}
		if beforeValue == nil && afterValue == nil {
			continue
		}
		_ = s.auditCatalogChange(ctx, actor, action, entityTypeCatalogPlan, id, beforeValue, afterValue, meta)
	}
	return BulkActionResponse{IDs: normalized, Affected: affected}, nil
}

func (s *Service) auditPromotionBulkAction(ctx context.Context, actor identity.User, ids []int64, meta RequestMeta, action string, run func(context.Context, []int64) (int64, error), captureAfter bool) (BulkActionResponse, error) {
	if !canManageCatalog(actor) {
		return BulkActionResponse{}, ErrForbidden
	}
	normalized, err := normalizeIDs(ids)
	if err != nil {
		return BulkActionResponse{}, err
	}
	before := make(map[int64]PromotionAuditSnapshot, len(normalized))
	for _, id := range normalized {
		if snapshot, err := s.promotionAuditSnapshot(ctx, id); err == nil {
			before[id] = snapshot
		}
	}
	affected, err := run(ctx, normalized)
	if err != nil {
		return BulkActionResponse{}, err
	}
	for _, id := range normalized {
		var beforeValue any
		if snapshot, ok := before[id]; ok {
			beforeValue = snapshot
		}
		var afterValue any
		if captureAfter {
			if snapshot, err := s.promotionAuditSnapshot(ctx, id); err == nil {
				afterValue = snapshot
			}
		}
		if beforeValue == nil && afterValue == nil {
			continue
		}
		_ = s.auditCatalogChange(ctx, actor, action, entityTypeCatalogPromotion, id, beforeValue, afterValue, meta)
	}
	return BulkActionResponse{IDs: normalized, Affected: affected}, nil
}

func snapshotAfterNotFound[T any](snapshot T, err error) (*T, error) {
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}
