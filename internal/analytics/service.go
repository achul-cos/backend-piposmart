package analytics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

var ErrDiagramNotFound = errors.New("diagram tidak ditemukan")

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		now:  currentNowUTC,
	}
}

func (s *Service) Catalog() []DiagramCatalogItem {
	return Catalog()
}

func (s *Service) CatalogByModule(module string) []DiagramCatalogItem {
	return CatalogByModule(module)
}

func (s *Service) Diagram(module, key string) (DiagramCatalogItem, bool) {
	return FindDiagram(module, key)
}

func (s *Service) Query(ctx context.Context, actor identity.User, module, key string, req QueryRequest) (QueryResult, error) {
	item, ok := FindDiagram(module, key)
	if !ok {
		return QueryResult{}, ErrDiagramNotFound
	}
	timeFilter, err := resolveTimeFilter(req.TimeFilter, s.now())
	if err != nil {
		return QueryResult{}, err
	}
	timeFilter.Label = humanLabel(timeFilter)
	currentData, err := s.queryByKey(ctx, actor, item, req, timeFilter)
	if err != nil {
		return QueryResult{}, err
	}
	comparison := ComparisonSummary{}
	var baseline *ResolvedTimeFilter
	if req.Comparison.Enabled && req.Comparison.Mode == "series_to_series" {
		comparison = ComparisonSummary{
			Enabled:      true,
			Mode:         req.Comparison.Mode,
			CurrentValue: round2(currentData.Value),
			PolarityRule: item.PolarityRule,
		}
	} else {
		comparison, baseline, err = resolveComparison(timeFilter, req.Comparison, s.now())
		if err != nil {
			return QueryResult{}, err
		}
		if baseline != nil {
			baselineData, baselineErr := s.queryByKey(ctx, actor, item, req, *baseline)
			if baselineErr != nil {
				return QueryResult{}, baselineErr
			}
			comparison = buildComparison(comparison, currentData.Value, baselineData.Value, item.PolarityRule)
		} else {
			comparison.CurrentValue = round2(currentData.Value)
			comparison.PolarityRule = item.PolarityRule
		}
	}
	result := QueryResult{
		Diagram: DiagramMetadata{
			Key:          item.Key,
			Module:       item.Module,
			Name:         item.Name,
			Type:         item.Type,
			Function:     item.Function,
			Purpose:      item.Purpose,
			HowToRead:    item.HowToRead,
			AnalysisGoal: item.AnalysisGoal,
		},
		TimeFilter: copyTimeFilterForResponse(timeFilter),
		Comparison: comparison,
		Series:     currentData.Series,
		Table:      currentData.Table,
		Extra:      currentData.Extra,
	}
	if req.Options.IncludeSummary || req.Options.IncludeSummary == false {
		result.Insight = buildInsight(item, comparison)
	}
	return result, nil
}

func (s *Service) queryByKey(ctx context.Context, actor identity.User, item DiagramCatalogItem, req QueryRequest, timeFilter ResolvedTimeFilter) (queryData, error) {
	switch item.Module + "/" + item.Key {
	case "owners/growth-trend":
		return s.repo.OwnerGrowthTrend(ctx, actor, req, timeFilter)
	case "owners/ownership-distribution":
		return s.repo.OwnerOwnershipDistribution(ctx, actor, req, timeFilter)
	case "owners/province-distribution":
		return s.repo.OwnerProvinceDistribution(ctx, actor, req, timeFilter)
	case "owners/city-top10":
		return s.repo.OwnerCityTop10(ctx, actor, req, timeFilter)
	case "owners/soft-delete-trend":
		return s.repo.OwnerSoftDeleteTrend(ctx, actor, req, timeFilter)
	case "outlets/growth-trend":
		return s.repo.OutletGrowthTrend(ctx, actor, req, timeFilter)
	case "outlets/outlet-per-owner-histogram":
		return s.repo.OutletPerOwnerHistogram(ctx, actor, req, timeFilter)
	case "outlets/subscription-status-recap":
		return s.repo.OutletSubscriptionStatusRecap(ctx, actor, req, timeFilter)
	case "outlets/not-subscribe-trend":
		return s.repo.OutletNotSubscribeTrend(ctx, actor, req, timeFilter)
	case "owners/indonesia-distribution-map":
		return s.repo.OwnerDistributionMap(ctx, actor, req, timeFilter)
	case "outlets/indonesia-distribution-map":
		return s.repo.OutletDistributionMap(ctx, actor, req, timeFilter)
	case "leads/funnel":
		return s.repo.LeadFunnel(ctx, actor, req, timeFilter)
	case "leads/aging-by-stage":
		return s.repo.LeadAgingByStage(ctx, actor, req, timeFilter)
	case "leads/assignment-distribution":
		return s.repo.LeadAssignmentDistribution(ctx, actor, req, timeFilter)
	case "leads/ownership-transfer-sankey":
		return s.repo.LeadOwnershipTransferSankey(ctx, actor, req, timeFilter)
	case "interactions/volume-trend":
		return s.repo.InteractionVolumeTrend(ctx, actor, req, timeFilter)
	case "interactions/remark-distribution":
		return s.repo.RemarkDistribution(ctx, actor, req, timeFilter)
	case "interactions/follow-up-compliance":
		return s.repo.FollowUpCompliance(ctx, actor, req, timeFilter)
	case "interactions/first-response-lag":
		return s.repo.FirstResponseLag(ctx, actor, req, timeFilter)
	case "trainings/scheduled-vs-completed":
		return s.repo.TrainingScheduledVsCompleted(ctx, actor, req, timeFilter)
	case "trainings/training-to-closing-conversion":
		return s.repo.TrainingToClosingConversion(ctx, actor, req, timeFilter)
	case "catalog/package-popularity":
		return s.repo.PackagePopularity(ctx, actor, req, timeFilter)
	case "catalog/tenure-popularity":
		return s.repo.TenurePopularity(ctx, actor, req, timeFilter)
	case "catalog/package-tenure-heatmap":
		return s.repo.PackageTenureHeatmap(ctx, actor, req, timeFilter)
	case "catalog/promo-adoption-rate":
		return s.repo.PromoAdoptionRate(ctx, actor, req, timeFilter)
	case "catalog/additional-charge-adoption":
		return s.repo.AdditionalChargeAdoption(ctx, actor, req, timeFilter)
	case "catalog/price-history-timeline":
		return s.repo.PriceHistoryTimeline(ctx, actor, req, timeFilter)
	case "catalog/promotion-history-timeline":
		return s.repo.PromotionHistoryTimeline(ctx, actor, req, timeFilter)
	case "closings/closing-trend":
		return s.repo.ClosingTrend(ctx, actor, req, timeFilter)
	case "closings/closing-by-sales":
		return s.repo.ClosingBySales(ctx, actor, req, timeFilter)
	case "closings/closing-by-supervisor":
		return s.repo.ClosingBySupervisor(ctx, actor, req, timeFilter)
	case "closings/closing-by-package":
		return s.repo.ClosingByPackage(ctx, actor, req, timeFilter)
	case "closings/closing-by-tenure":
		return s.repo.ClosingByTenure(ctx, actor, req, timeFilter)
	case "closings/status-distribution":
		return s.repo.ClosingStatusDistribution(ctx, actor, req, timeFilter)
	case "closings/average-ticket-size-trend":
		return s.repo.AverageTicketSizeTrend(ctx, actor, req, timeFilter)
	case "closings/closing-amount-waterfall":
		return s.repo.ClosingAmountWaterfall(ctx, actor, req, timeFilter)
	case "targets/target-vs-actual":
		return s.repo.TargetVsActual(ctx, actor, req, timeFilter)
	case "targets/target-burnup":
		return s.repo.TargetBurnup(ctx, actor, req, timeFilter)
	case "kpi/leaderboard":
		return s.repo.KpiLeaderboard(ctx, actor, req, timeFilter)
	case "kpi/activity-vs-closing-scatter":
		return s.repo.ActivityVsClosingScatter(ctx, actor, req, timeFilter)
	case "wallets/topup-revenue-trend":
		return s.repo.TopupRevenueTrend(ctx, actor, req, timeFilter)
	case "wallets/topup-transaction-count":
		return s.repo.TopupTransactionCount(ctx, actor, req, timeFilter)
	case "wallets/owner-balance-distribution":
		return s.repo.OwnerBalanceDistribution(ctx, actor, req, timeFilter)
	case "wallets/topup-used-vs-unused":
		return s.repo.TopupUsedVsUnused(ctx, actor, req, timeFilter)
	case "wallets/topup-to-subscribe-lag":
		return s.repo.TopupToSubscribeLag(ctx, actor, req, timeFilter)
	case "wallets/zero-vs-nonzero-balance":
		return s.repo.ZeroVsNonZeroBalance(ctx, actor, req, timeFilter)
	case "subscriptions/active-subscription-trend":
		return s.repo.ActiveSubscriptionTrend(ctx, actor, req, timeFilter)
	case "subscriptions/activation-vs-expiry-trend":
		return s.repo.ActivationVsExpiryTrend(ctx, actor, req, timeFilter)
	case "subscriptions/renewal-rate":
		return s.repo.RenewalRate(ctx, actor, req, timeFilter)
	case "subscriptions/expiry-forecast":
		return s.repo.ExpiryForecast(ctx, actor, req, timeFilter)
	case "subscriptions/package-mix":
		return s.repo.SubscriptionPackageMix(ctx, actor, req, timeFilter)
	case "subscriptions/tenure-mix":
		return s.repo.SubscriptionTenureMix(ctx, actor, req, timeFilter)
	case "subscriptions/days-remaining-histogram":
		return s.repo.DaysRemainingHistogram(ctx, actor, req, timeFilter)
	case "subscriptions/churn-bucket-trend":
		return s.repo.ChurnBucketTrend(ctx, actor, req, timeFilter)
	case "reconciliation/success-rate":
		return s.repo.ReconciliationSuccessRate(ctx, actor, req, timeFilter)
	case "reconciliation/issue-by-type":
		return s.repo.ReconciliationIssueByType(ctx, actor, req, timeFilter)
	case "reconciliation/issue-aging":
		return s.repo.ReconciliationIssueAging(ctx, actor, req, timeFilter)
	case "reconciliation/auto-vs-manual":
		return s.repo.ReconciliationAutoVsManual(ctx, actor, req, timeFilter)
	case "reconciliation/hanging-transaction-trend":
		return s.repo.HangingTransactionTrend(ctx, actor, req, timeFilter)
	case "reconciliation/revenue-vs-closing-period-compare":
		return s.repo.RevenueVsClosingPeriodCompare(ctx, actor, req, timeFilter)
	case "partners/partner-growth-trend":
		return s.repo.PartnerGrowthTrend(ctx, actor, req, timeFilter)
	case "partners/partner-type-distribution":
		return s.repo.PartnerTypeDistribution(ctx, actor, req, timeFilter)
	case "partners/referral-count-per-partner":
		return s.repo.ReferralCountPerPartner(ctx, actor, req, timeFilter)
	case "partners/referral-conversion-per-partner":
		return s.repo.ReferralConversionPerPartner(ctx, actor, req, timeFilter)
	case "partners/partner-pic-workload":
		return s.repo.PartnerPICWorkload(ctx, actor, req, timeFilter)
	case "partners/call-mitra-frequency":
		return s.repo.CallMitraFrequency(ctx, actor, req, timeFilter)
	case "partners/partner-inactivity-aging":
		return s.repo.PartnerInactivityAging(ctx, actor, req, timeFilter)
	case "partners/partner-region-distribution":
		return s.repo.PartnerRegionDistribution(ctx, actor, req, timeFilter)
	case "commissions/commission-earned-trend":
		return s.repo.CommissionEarnedTrend(ctx, actor, req, timeFilter)
	case "commissions/paid-vs-unpaid":
		return s.repo.PaidVsUnpaidCommission(ctx, actor, req, timeFilter)
	case "commissions/commission-aging":
		return s.repo.CommissionAging(ctx, actor, req, timeFilter)
	case "commissions/commission-by-partner-type":
		return s.repo.CommissionByPartnerType(ctx, actor, req, timeFilter)
	case "commissions/commission-by-package":
		return s.repo.CommissionByPackage(ctx, actor, req, timeFilter)
	case "commissions/payout-waterfall":
		return s.repo.PayoutWaterfall(ctx, actor, req, timeFilter)
	case "commissions/rule-history-timeline":
		return s.repo.CommissionRuleHistoryTimeline(ctx, actor, req, timeFilter)
	case "commissions/snapshot-vs-current":
		return s.repo.SnapshotVsCurrentCommission(ctx, actor, req, timeFilter)
	case "audit/log-volume-by-module":
		return s.repo.AuditLogVolumeByModule(ctx, actor, req, timeFilter)
	case "audit/actor-activity-chart":
		return s.repo.ActorActivityChart(ctx, actor, req, timeFilter)
	case "audit/restore-vs-delete-trend":
		return s.repo.RestoreVsDeleteTrend(ctx, actor, req, timeFilter)
	case "audit/backend-error-code-frequency":
		return s.repo.BackendErrorCodeFrequency(ctx, actor, req, timeFilter)
	case "imports/batches-per-profile":
		return s.repo.ImportBatchesPerProfile(ctx, actor, req, timeFilter)
	case "imports/success-vs-failed":
		return s.repo.ImportSuccessVsFailed(ctx, actor, req, timeFilter)
	case "imports/invalid-rows-distribution":
		return s.repo.InvalidRowsDistribution(ctx, actor, req, timeFilter)
	case "imports/validation-error-by-profile":
		return s.repo.ValidationErrorByProfile(ctx, actor, req, timeFilter)
	case "imports/duplicate-detection-rate":
		return s.repo.DuplicateDetectionRate(ctx, actor, req, timeFilter)
	case "imports/import-duration-trend":
		return s.repo.ImportDurationTrend(ctx, actor, req, timeFilter)
	case "imports/batch-status-funnel":
		return s.repo.BatchStatusFunnel(ctx, actor, req, timeFilter)
	case "imports/uploader-activity":
		return s.repo.ImportUploaderActivity(ctx, actor, req, timeFilter)
	case "imports/file-history-usage":
		return s.repo.FileHistoryUsage(ctx, actor, req, timeFilter)
	case "executive/end-to-end-funnel":
		return s.repo.EndToEndBusinessFunnel(ctx, actor, req, timeFilter)
	case "executive/revenue-closing-active-subscription-board":
		return s.repo.RevenueClosingActiveSubscriptionBoard(ctx, actor, req, timeFilter)
	case "executive/monthly-operating-review-board":
		return s.repo.MonthlyOperatingReviewBoard(ctx, actor, req, timeFilter)
	case "executive/north-star-kpi-trend":
		return s.repo.NorthStarKPITrend(ctx, actor, req, timeFilter)
	case "executive/data-quality-score-by-module":
		return s.repo.DataQualityScoreByModule(ctx, actor, req, timeFilter)
	case "custom/multi-series-trend":
		return s.repo.CustomMultiSeriesTrend(ctx, actor, req, timeFilter)
	case "custom/metric-comparison-board":
		return s.repo.MetricComparisonBoard(ctx, actor, req, timeFilter)
	case "custom/region-comparison-board":
		return s.repo.RegionComparisonBoard(ctx, actor, req, timeFilter)
	case "subscriptions/cohort-retention":
		return s.repo.SubscriptionCohortRetention(ctx, actor, req, timeFilter)
	case "executive/forecast-summary-board":
		return s.repo.ForecastSummaryBoard(ctx, actor, req, timeFilter)
	case "custom/comparison-impact-summary":
		return s.repo.ComparisonImpactSummary(ctx, actor, req, timeFilter)
	default:
		return queryData{}, fmt.Errorf("diagram belum didukung")
	}
}
