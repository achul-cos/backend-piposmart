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
	packages.POST("", h.createPackage)
	packages.DELETE("/bulk", h.bulkDeletePackages)
	packages.PATCH("/bulk/restore", h.bulkRestorePackages)
	packages.DELETE("/bulk/force", h.bulkForceDeletePackages)
	packages.GET("/:package_id", h.getPackage)
	packages.PATCH("/:package_id", h.updatePackage)
	packages.DELETE("/:package_id", h.deletePackage)
	packages.PATCH("/:package_id/restore", h.restorePackage)
	packages.DELETE("/:package_id/force", h.forceDeletePackage)

	plans := catalog.Group("/plans")
	plans.GET("", h.listPlans)
	plans.POST("", h.createPlan)
	plans.DELETE("/bulk", h.bulkDeletePlans)
	plans.PATCH("/bulk/restore", h.bulkRestorePlans)
	plans.DELETE("/bulk/force", h.bulkForceDeletePlans)
	plans.GET("/:plan_id", h.getPlan)
	plans.PATCH("/:plan_id", h.updatePlan)
	plans.DELETE("/:plan_id", h.deletePlan)
	plans.PATCH("/:plan_id/restore", h.restorePlan)
	plans.DELETE("/:plan_id/force", h.forceDeletePlan)
	plans.GET("/:plan_id/eligible-promotions", h.eligiblePromotions)

	promotions := catalog.Group("/promotions")
	promotions.GET("", h.listPromotions)
	promotions.POST("", h.createPromotion)
	promotions.DELETE("/bulk", h.bulkDeletePromotions)
	promotions.PATCH("/bulk/restore", h.bulkRestorePromotions)
	promotions.DELETE("/bulk/force", h.bulkForceDeletePromotions)
	promotions.GET("/:promotion_id", h.getPromotion)
	promotions.PATCH("/:promotion_id", h.updatePromotion)
	promotions.DELETE("/:promotion_id", h.deletePromotion)
	promotions.PATCH("/:promotion_id/restore", h.restorePromotion)
	promotions.DELETE("/:promotion_id/force", h.forceDeletePromotion)
	promotions.GET("/:promotion_id/benefits", h.listBenefits)
	promotions.POST("/:promotion_id/benefits", h.createBenefit)
	promotions.DELETE("/:promotion_id/benefits/:benefit_id", h.deleteBenefit)
	promotions.PUT("/:promotion_id/eligible-plans", h.setEligibility)
}

func (h *Handler) listPackages(c *gin.Context) {
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListPackages(c.Request.Context(), params)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) createPackage(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req CreatePackageRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreatePackage(c.Request.Context(), user, req)
	writeResult(c, http.StatusCreated, response, err)
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
	response, err := h.service.UpdatePackage(c.Request.Context(), user, id, req)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) deletePackage(c *gin.Context) {
	h.packageBulkAction(c, []int64FromParam{{Name: "package_id"}}, h.service.DeletePackages)
}

func (h *Handler) restorePackage(c *gin.Context) {
	h.packageBulkAction(c, []int64FromParam{{Name: "package_id"}}, h.service.RestorePackages)
}

func (h *Handler) forceDeletePackage(c *gin.Context) {
	h.packageBulkAction(c, []int64FromParam{{Name: "package_id"}}, h.service.ForceDeletePackages)
}

func (h *Handler) bulkDeletePackages(c *gin.Context) {
	h.packageBulkFromBody(c, h.service.DeletePackages)
}

func (h *Handler) bulkRestorePackages(c *gin.Context) {
	h.packageBulkFromBody(c, h.service.RestorePackages)
}

func (h *Handler) bulkForceDeletePackages(c *gin.Context) {
	h.packageBulkFromBody(c, h.service.ForceDeletePackages)
}

func (h *Handler) listPlans(c *gin.Context) {
	params, ok := listParams(c)
	if !ok {
		return
	}
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
	response, err := h.service.CreatePlan(c.Request.Context(), user, req)
	writeResult(c, http.StatusCreated, response, err)
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
	response, err := h.service.UpdatePlan(c.Request.Context(), user, id, req)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) deletePlan(c *gin.Context) {
	h.planBulkAction(c, []int64FromParam{{Name: "plan_id"}}, h.service.DeletePlans)
}

func (h *Handler) restorePlan(c *gin.Context) {
	h.planBulkAction(c, []int64FromParam{{Name: "plan_id"}}, h.service.RestorePlans)
}

func (h *Handler) forceDeletePlan(c *gin.Context) {
	h.planBulkAction(c, []int64FromParam{{Name: "plan_id"}}, h.service.ForceDeletePlans)
}

func (h *Handler) bulkDeletePlans(c *gin.Context) {
	h.planBulkFromBody(c, h.service.DeletePlans)
}

func (h *Handler) bulkRestorePlans(c *gin.Context) {
	h.planBulkFromBody(c, h.service.RestorePlans)
}

func (h *Handler) bulkForceDeletePlans(c *gin.Context) {
	h.planBulkFromBody(c, h.service.ForceDeletePlans)
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
	params, ok := listParams(c)
	if !ok {
		return
	}
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
	response, err := h.service.CreatePromotion(c.Request.Context(), user, req)
	writeResult(c, http.StatusCreated, response, err)
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
	response, err := h.service.UpdatePromotion(c.Request.Context(), user, id, req)
	writeResult(c, http.StatusOK, response, err)
}

func (h *Handler) deletePromotion(c *gin.Context) {
	h.promotionBulkAction(c, []int64FromParam{{Name: "promotion_id"}}, h.service.DeletePromotions)
}

func (h *Handler) restorePromotion(c *gin.Context) {
	h.promotionBulkAction(c, []int64FromParam{{Name: "promotion_id"}}, h.service.RestorePromotions)
}

func (h *Handler) forceDeletePromotion(c *gin.Context) {
	h.promotionBulkAction(c, []int64FromParam{{Name: "promotion_id"}}, h.service.ForceDeletePromotions)
}

func (h *Handler) bulkDeletePromotions(c *gin.Context) {
	h.promotionBulkFromBody(c, h.service.DeletePromotions)
}

func (h *Handler) bulkRestorePromotions(c *gin.Context) {
	h.promotionBulkFromBody(c, h.service.RestorePromotions)
}

func (h *Handler) bulkForceDeletePromotions(c *gin.Context) {
	h.promotionBulkFromBody(c, h.service.ForceDeletePromotions)
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
	response, err := h.service.CreateBenefit(c.Request.Context(), user, id, req)
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
	response, err := h.service.DeleteBenefit(c.Request.Context(), user, promotionID, benefitID)
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
	response, err := h.service.SetEligibility(c.Request.Context(), user, id, req)
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
