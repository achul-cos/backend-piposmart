package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	catalog := api.Group("/catalog")

	packages := catalog.Group("/packages")
	packages.GET("", h.listPackages)
	packages.GET("/all", h.listAllPackages)
	packages.GET("/all-deleted", h.listAllPackagesDeleted)
	packages.GET("/trash", h.listPackagesTrash)
	packages.GET("/unscoped", h.listPackagesUnscoped)
	packages.POST("", h.createPackage)
	packages.DELETE("/bulk", h.bulkDeletePackages)
	packages.PATCH("/bulk/restore", h.bulkRestorePackages)
	packages.DELETE("/bulk/force", h.bulkForceDeletePackages)
	packages.GET("/:package_id/histories", h.listPackageHistories)
	packages.GET("/:package_id", h.getPackage)
	packages.PATCH("/:package_id", h.updatePackage)
	packages.DELETE("/:package_id", h.deletePackage)
	packages.PATCH("/:package_id/restore", h.restorePackage)
	packages.DELETE("/:package_id/force", h.forceDeletePackage)

	plans := catalog.Group("/plans")
	plans.GET("", h.listPlans)
	plans.GET("/all", h.listAllPlans)
	plans.GET("/all-deleted", h.listAllPlansDeleted)
	plans.GET("/trash", h.listPlansTrash)
	plans.GET("/unscoped", h.listPlansUnscoped)
	plans.POST("", h.createPlan)
	plans.DELETE("/bulk", h.bulkDeletePlans)
	plans.PATCH("/bulk/restore", h.bulkRestorePlans)
	plans.DELETE("/bulk/force", h.bulkForceDeletePlans)
	plans.GET("/:plan_id/histories", h.listPlanHistories)
	plans.GET("/:plan_id", h.getPlan)
	plans.PATCH("/:plan_id", h.updatePlan)
	plans.DELETE("/:plan_id", h.deletePlan)
	plans.PATCH("/:plan_id/restore", h.restorePlan)
	plans.DELETE("/:plan_id/force", h.forceDeletePlan)
	plans.GET("/:plan_id/eligible-promotions", h.eligiblePromotions)

	promotions := catalog.Group("/promotions")
	promotions.GET("", h.listPromotions)
	promotions.GET("/all", h.listAllPromotions)
	promotions.GET("/all-deleted", h.listAllPromotionsDeleted)
	promotions.GET("/trash", h.listPromotionsTrash)
	promotions.GET("/unscoped", h.listPromotionsUnscoped)
	promotions.POST("", h.createPromotion)
	promotions.DELETE("/bulk", h.bulkDeletePromotions)
	promotions.PATCH("/bulk/restore", h.bulkRestorePromotions)
	promotions.DELETE("/bulk/force", h.bulkForceDeletePromotions)
	promotions.GET("/:promotion_id/histories", h.listPromotionHistories)
	promotions.GET("/:promotion_id", h.getPromotion)
	promotions.PATCH("/:promotion_id", h.updatePromotion)
	promotions.DELETE("/:promotion_id", h.deletePromotion)
	promotions.PATCH("/:promotion_id/restore", h.restorePromotion)
	promotions.DELETE("/:promotion_id/force", h.forceDeletePromotion)
	promotions.GET("/:promotion_id/benefits", h.listBenefits)
	promotions.POST("/:promotion_id/benefits", h.createBenefit)
	promotions.DELETE("/:promotion_id/benefits/:benefit_id", h.deleteBenefit)
	promotions.GET("/:promotion_id/eligible-plans", h.eligiblePlans)
	promotions.PUT("/:promotion_id/eligible-plans", h.setEligibility)
}

func (h *Handler) listPackages(c *gin.Context) {
	h.listPackagesWithScope(c, ScopeActive)
}

func (h *Handler) listAllPackages(c *gin.Context) {
	h.listPackagesWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllPackagesDeleted(c *gin.Context) {
	h.listPackagesWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listPackagesTrash(c *gin.Context) {
	h.listPackagesWithScope(c, ScopeDeleted)
}

func (h *Handler) listPackagesUnscoped(c *gin.Context) {
	h.listPackagesWithScope(c, ScopeAll)
}

func (h *Handler) listPackagesWithScope(c *gin.Context, scope string) {
	h.listPackagesWithScopeAndAll(c, scope, false)
}

func (h *Handler) listPackagesWithScopeAndAll(c *gin.Context, scope string, all bool) {
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.Scope = scope
	params.All = all
	response, err := h.service.ListPackages(c.Request.Context(), params)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) createPackage(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req CreatePackageRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreatePackageWithMeta(c.Request.Context(), user, req, requestMeta(c))
	writeResult(c, http.StatusCreated, response, err)
}

func (h *Handler) listPackageHistories(c *gin.Context) {
	id, ok := parseID(c, "package_id")
	if !ok {
		return
	}
	response, err := h.service.ListPackageHistories(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) getPackage(c *gin.Context) {
	id, ok := parseID(c, "package_id")
	if !ok {
		return
	}
	response, err := h.service.GetPackage(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) updatePackage(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parseID(c, "package_id")
	if !ok {
		return
	}
	var req UpdatePackageRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.UpdatePackageWithMeta(c.Request.Context(), user, id, req, requestMeta(c))
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) deletePackage(c *gin.Context) {
	h.packageBulkAction(c, []int64FromParam{{Name: "package_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.DeletePackagesWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) restorePackage(c *gin.Context) {
	h.packageBulkAction(c, []int64FromParam{{Name: "package_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.RestorePackagesWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) forceDeletePackage(c *gin.Context) {
	h.packageBulkAction(c, []int64FromParam{{Name: "package_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.ForceDeletePackagesWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkDeletePackages(c *gin.Context) {
	h.packageBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.DeletePackagesWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkRestorePackages(c *gin.Context) {
	h.packageBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.RestorePackagesWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkForceDeletePackages(c *gin.Context) {
	h.packageBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.ForceDeletePackagesWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) listPlans(c *gin.Context) {
	h.listPlansWithScope(c, ScopeActive)
}

func (h *Handler) listAllPlans(c *gin.Context) {
	h.listPlansWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllPlansDeleted(c *gin.Context) {
	h.listPlansWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listPlansTrash(c *gin.Context) {
	h.listPlansWithScope(c, ScopeDeleted)
}

func (h *Handler) listPlansUnscoped(c *gin.Context) {
	h.listPlansWithScope(c, ScopeAll)
}

func (h *Handler) listPlansWithScope(c *gin.Context, scope string) {
	h.listPlansWithScopeAndAll(c, scope, false)
}

func (h *Handler) listPlansWithScopeAndAll(c *gin.Context, scope string, all bool) {
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.Scope = scope
	params.All = all
	if !optionalInt64(c, "package_id", &params.PackageID) {
		return
	}
	response, err := h.service.ListPlans(c.Request.Context(), params)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) createPlan(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req CreatePlanRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreatePlanWithMeta(c.Request.Context(), user, req, requestMeta(c))
	writeResult(c, http.StatusCreated, response, err)
}

func (h *Handler) listPlanHistories(c *gin.Context) {
	id, ok := parseID(c, "plan_id")
	if !ok {
		return
	}
	response, err := h.service.ListPlanHistories(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) getPlan(c *gin.Context) {
	id, ok := parseID(c, "plan_id")
	if !ok {
		return
	}
	response, err := h.service.GetPlan(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) updatePlan(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parseID(c, "plan_id")
	if !ok {
		return
	}
	var req UpdatePlanRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.UpdatePlanWithMeta(c.Request.Context(), user, id, req, requestMeta(c))
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) deletePlan(c *gin.Context) {
	h.planBulkAction(c, []int64FromParam{{Name: "plan_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.DeletePlansWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) restorePlan(c *gin.Context) {
	h.planBulkAction(c, []int64FromParam{{Name: "plan_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.RestorePlansWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) forceDeletePlan(c *gin.Context) {
	h.planBulkAction(c, []int64FromParam{{Name: "plan_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.ForceDeletePlansWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkDeletePlans(c *gin.Context) {
	h.planBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.DeletePlansWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkRestorePlans(c *gin.Context) {
	h.planBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.RestorePlansWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkForceDeletePlans(c *gin.Context) {
	h.planBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.ForceDeletePlansWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) eligiblePromotions(c *gin.Context) {
	id, ok := parseID(c, "plan_id")
	if !ok {
		return
	}
	asOf, ok := parseOptionalDateQuery(c, "as_of")
	if !ok {
		return
	}
	response, err := h.service.EligiblePromotions(c.Request.Context(), id, asOf)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) listPromotions(c *gin.Context) {
	h.listPromotionsWithScope(c, ScopeActive)
}

func (h *Handler) listAllPromotions(c *gin.Context) {
	h.listPromotionsWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllPromotionsDeleted(c *gin.Context) {
	h.listPromotionsWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listPromotionsTrash(c *gin.Context) {
	h.listPromotionsWithScope(c, ScopeDeleted)
}

func (h *Handler) listPromotionsUnscoped(c *gin.Context) {
	h.listPromotionsWithScope(c, ScopeAll)
}

func (h *Handler) listPromotionsWithScope(c *gin.Context, scope string) {
	h.listPromotionsWithScopeAndAll(c, scope, false)
}

func (h *Handler) listPromotionsWithScopeAndAll(c *gin.Context, scope string, all bool) {
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.Scope = scope
	params.All = all
	params.ChargeType = c.Query("charge_type")
	response, err := h.service.ListPromotions(c.Request.Context(), params)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) createPromotion(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req CreatePromotionRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreatePromotionWithMeta(c.Request.Context(), user, req, requestMeta(c))
	writeResult(c, http.StatusCreated, response, err)
}

func (h *Handler) listPromotionHistories(c *gin.Context) {
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	response, err := h.service.ListPromotionHistories(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) getPromotion(c *gin.Context) {
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	response, err := h.service.GetPromotion(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) updatePromotion(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	var req UpdatePromotionRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.UpdatePromotionWithMeta(c.Request.Context(), user, id, req, requestMeta(c))
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) deletePromotion(c *gin.Context) {
	h.promotionBulkAction(c, []int64FromParam{{Name: "promotion_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.DeletePromotionsWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) restorePromotion(c *gin.Context) {
	h.promotionBulkAction(c, []int64FromParam{{Name: "promotion_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.RestorePromotionsWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) forceDeletePromotion(c *gin.Context) {
	h.promotionBulkAction(c, []int64FromParam{{Name: "promotion_id"}}, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.ForceDeletePromotionsWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkDeletePromotions(c *gin.Context) {
	h.promotionBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.DeletePromotionsWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkRestorePromotions(c *gin.Context) {
	h.promotionBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.RestorePromotionsWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) bulkForceDeletePromotions(c *gin.Context) {
	h.promotionBulkFromBody(c, func(ctx context.Context, user identity.User, ids []int64) (BulkActionResponse, error) {
		return h.service.ForceDeletePromotionsWithMeta(ctx, user, ids, requestMeta(c))
	})
}

func (h *Handler) listBenefits(c *gin.Context) {
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	response, err := h.service.ListBenefits(c.Request.Context(), id)
	writeResult(c, http.StatusOK, gin.H{"items": response, "total": len(response)}, err)
}

func (h *Handler) createBenefit(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	var req CreateBenefitRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateBenefitWithMeta(c.Request.Context(), user, id, req, requestMeta(c))
	writeResult(c, http.StatusCreated, response, err)
}

func (h *Handler) deleteBenefit(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	promotionID, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	benefitID, ok := parseID(c, "benefit_id")
	if !ok {
		return
	}
	response, err := h.service.DeleteBenefitWithMeta(c.Request.Context(), user, promotionID, benefitID, requestMeta(c))
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) eligiblePlans(c *gin.Context) {
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	response, err := h.service.EligiblePlans(c.Request.Context(), id)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) setEligibility(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parseID(c, "promotion_id")
	if !ok {
		return
	}
	var req SetEligibilityRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.SetEligibilityWithMeta(c.Request.Context(), user, id, req, requestMeta(c))
	writeResult(c, http.StatusOK, response, err)
}

type bulkAction func(context.Context, identity.User, []int64) (BulkActionResponse, error)

type int64FromParam struct {
	Name string
}

func (h *Handler) packageBulkAction(c *gin.Context, params []int64FromParam, action bulkAction) {
	h.bulkFromParams(c, params, action)
}

func (h *Handler) planBulkAction(c *gin.Context, params []int64FromParam, action bulkAction) {
	h.bulkFromParams(c, params, action)
}

func (h *Handler) promotionBulkAction(c *gin.Context, params []int64FromParam, action bulkAction) {
	h.bulkFromParams(c, params, action)
}

func (h *Handler) packageBulkFromBody(c *gin.Context, action bulkAction) {
	h.bulkFromBody(c, action)
}

func (h *Handler) planBulkFromBody(c *gin.Context, action bulkAction) {
	h.bulkFromBody(c, action)
}

func (h *Handler) promotionBulkFromBody(c *gin.Context, action bulkAction) {
	h.bulkFromBody(c, action)
}

func (h *Handler) bulkFromParams(c *gin.Context, params []int64FromParam, action bulkAction) {
	user, _ := identity.CurrentUser(c)
	ids := make([]int64, 0, len(params))
	for _, param := range params {
		id, ok := parseID(c, param.Name)
		if !ok {
			return
		}
		ids = append(ids, id)
	}
	response, err := action(c.Request.Context(), user, ids)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) bulkFromBody(c *gin.Context, action bulkAction) {
	user, _ := identity.CurrentUser(c)
	var req BulkIDRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := action(c.Request.Context(), user, req.IDs)
	writeResult(c, http.StatusOK, response, err)
}

func listParams(c *gin.Context) (ListParams, bool) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := ListParams{Query: c.Query("q"), Page: page, Limit: limit, Sort: c.Query("sort")}
	if !optionalBool(c, "active", &params.Active) {
		return ListParams{}, false
	}
	asOf, ok := parseOptionalDateQuery(c, "as_of")
	if !ok {
		return ListParams{}, false
	}
	params.AsOf = asOf
	return params, true
}

func optionalBool(c *gin.Context, name string, target **bool) bool {
	raw := c.Query(name)
	if raw == "" {
		return true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", name+" harus boolean", nil)
		return false
	}
	*target = &value
	return true
}

func optionalInt64(c *gin.Context, name string, target **int64) bool {
	raw := c.Query(name)
	if raw == "" {
		return true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", name+" tidak valid", nil)
		return false
	}
	*target = &value
	return true
}

func parseOptionalDateQuery(c *gin.Context, name string) (*time.Time, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	value, err := parseDate(raw)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "INVALID_DATE", err.Error(), nil)
		return nil, false
	}
	return &value, true
}

func parseID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return 0, false
	}
	return id, true
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func requestMeta(c *gin.Context) RequestMeta {
	return RequestMeta{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		RequestID: httpx.RequestID(c),
	}
}

func writeResult(c *gin.Context, status int, data any, err error) {
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, status, data)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, ErrInvalidSort):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SORT", err.Error(), nil)
	case errors.Is(err, ErrInvalidDecimal):
		httpx.Error(c, http.StatusBadRequest, "INVALID_DECIMAL", err.Error(), nil)
	case errors.Is(err, ErrInvalidDate):
		httpx.Error(c, http.StatusBadRequest, "INVALID_DATE", err.Error(), nil)
	case errors.Is(err, ErrInvalidTenure):
		httpx.Error(c, http.StatusBadRequest, "INVALID_TENURE", err.Error(), nil)
	case errors.Is(err, ErrInvalidCharge):
		httpx.Error(c, http.StatusBadRequest, "INVALID_CHARGE_TYPE", err.Error(), nil)
	case errors.Is(err, ErrCodeExists):
		httpx.Error(c, http.StatusConflict, "CODE_ALREADY_USED", err.Error(), nil)
	case errors.Is(err, ErrEmptyBulk):
		httpx.Error(c, http.StatusBadRequest, "EMPTY_BULK", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
