package target

import (
	"errors"
	"net/http"
	"strconv"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// Handler handles sales-target HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new target Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers sales-target routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	targets := rg.Group("/sales-targets")
	{
		targets.POST("/bulk", h.BulkSetTarget)
		targets.PUT("/:salesID", h.OverrideTarget)
		targets.GET("", h.ListTargets)
		targets.GET("/all", h.ListAllTargets)
		targets.GET("/all-deleted", h.ListAllTargets)
	}
}

// BulkSetTarget godoc
// @Summary Bulk-set a monthly sales target
// @Description Sets a default target for every active Sales rep who does not already have one for the given metric/period. Never overwrites an existing row.
// @Tags sales-targets
// @Accept json
// @Produce json
// @Param request body BulkSetTargetRequest true "Bulk target data"
// @Success 200 {object} BulkSetTargetResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /sales-targets/bulk [post]
func (h *Handler) BulkSetTarget(c *gin.Context) {
	var req BulkSetTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.BulkSetTarget(c.Request.Context(), actor, req)
	if err != nil {
		writeTargetError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// OverrideTarget godoc
// @Summary Override a single Sales rep's monthly target
// @Description Always wins over a bulk-set value for the given sales rep.
// @Tags sales-targets
// @Accept json
// @Produce json
// @Param salesID path int64 true "Sales user ID"
// @Param request body OverrideTargetRequest true "Override target data"
// @Success 200 {object} SalesTargetResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /sales-targets/{salesID} [put]
func (h *Handler) OverrideTarget(c *gin.Context) {
	salesID, err := strconv.ParseInt(c.Param("salesID"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid sales ID", nil)
		return
	}
	var req OverrideTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.OverrideTarget(c.Request.Context(), actor, salesID, req)
	if err != nil {
		writeTargetError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListTargets godoc
// @Summary List sales targets
// @Description Sales sees only their own targets; Admin/Supervisor see all, optionally filtered.
// @Tags sales-targets
// @Produce json
// @Param sales_id query int64 false "Filter by Sales user ID"
// @Param period_year query int false "Filter by period year"
// @Param period_month query int false "Filter by period month"
// @Param metric_code query string false "Filter by metric code"
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Success 200 {object} SalesTargetListResponse
// @Router /sales-targets [get]
func (h *Handler) ListTargets(c *gin.Context) {
	params := ListTargetsParams{
		MetricCode: c.Query("metric_code"),
		All:        c.Query("all") == "true",
	}
	if v := c.Query("sales_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.SalesID = &id
		}
	}
	if v := c.Query("period_year"); v != "" {
		if year, err := strconv.Atoi(v); err == nil {
			params.PeriodYear = &year
		}
	}
	if v := c.Query("period_month"); v != "" {
		if month, err := strconv.Atoi(v); err == nil {
			params.PeriodMonth = &month
		}
	}
	params.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	params.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))

	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListTargets(c.Request.Context(), actor, params)
	if err != nil {
		writeTargetError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) ListAllTargets(c *gin.Context) {
	params := ListTargetsParams{
		MetricCode: c.Query("metric_code"),
		All:        true,
	}
	if v := c.Query("sales_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.SalesID = &id
		}
	}
	if v := c.Query("period_year"); v != "" {
		if year, err := strconv.Atoi(v); err == nil {
			params.PeriodYear = &year
		}
	}
	if v := c.Query("period_month"); v != "" {
		if month, err := strconv.Atoi(v); err == nil {
			params.PeriodMonth = &month
		}
	}
	params.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	params.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListTargets(c.Request.Context(), actor, params)
	if err != nil {
		writeTargetError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func writeTargetError(c *gin.Context, err error) {
	switch err {
	case nil:
		return
	case ErrNotFound:
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case ErrForbidden:
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case ErrInvalidMetric:
		httpx.Error(c, http.StatusBadRequest, "INVALID_METRIC", err.Error(), nil)
	case ErrInvalidPeriod:
		httpx.Error(c, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
	case ErrInvalidValue:
		httpx.Error(c, http.StatusBadRequest, "INVALID_VALUE", err.Error(), nil)
	case ErrSalesNotEligible:
		httpx.Error(c, http.StatusBadRequest, "SALES_NOT_ELIGIBLE", err.Error(), nil)
	default:
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		case errors.Is(err, ErrForbidden):
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		case errors.Is(err, ErrInvalidMetric):
			httpx.Error(c, http.StatusBadRequest, "INVALID_METRIC", err.Error(), nil)
		case errors.Is(err, ErrInvalidPeriod):
			httpx.Error(c, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
		case errors.Is(err, ErrInvalidValue):
			httpx.Error(c, http.StatusBadRequest, "INVALID_VALUE", err.Error(), nil)
		case errors.Is(err, ErrSalesNotEligible):
			httpx.Error(c, http.StatusBadRequest, "SALES_NOT_ELIGIBLE", err.Error(), nil)
		default:
			httpx.InternalServerError(c, "Terjadi kesalahan pada server", err)
		}
	}
}
