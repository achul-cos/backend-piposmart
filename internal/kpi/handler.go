package kpi

import (
	"errors"
	"net/http"
	"strconv"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// Handler handles KPI-related HTTP requests: definitions, recompute jobs, results, ranking.
type Handler struct {
	service *Service
}

// NewHandler creates a new kpi Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers KPI routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	definitions := rg.Group("/kpi-definitions")
	{
		definitions.POST("", h.CreateDefinition)
		definitions.GET("", h.ListDefinitions)
		definitions.GET("/:id", h.GetDefinition)
		definitions.PATCH("/:id/deactivate", h.DeactivateDefinition)
	}

	kpiGroup := rg.Group("/kpi")
	{
		kpiGroup.POST("/recompute", h.TriggerRecompute)
		kpiGroup.GET("/jobs/:id", h.GetJob)
		kpiGroup.GET("/results", h.ListResults)
		kpiGroup.GET("/ranking", h.ListRanking)
	}
}

/* ---------- KPI Definition ---------- */

// CreateDefinition godoc
// @Summary Create a KPI definition
// @Description Define a metric's weight and achievement thresholds for a period. Active definitions for a period must sum to 100% weight (validated at recompute time).
// @Tags kpi-definitions
// @Accept json
// @Produce json
// @Param request body CreateKpiDefinitionRequest true "KPI definition data"
// @Success 201 {object} KpiDefinitionResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /kpi-definitions [post]
func (h *Handler) CreateDefinition(c *gin.Context) {
	var req CreateKpiDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CreateDefinition(c.Request.Context(), actor, req)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// ListDefinitions godoc
// @Summary List KPI definitions
// @Tags kpi-definitions
// @Produce json
// @Param period_year query int false "Filter by period year"
// @Param period_month query int false "Filter by period month"
// @Param active_only query bool false "Only active definitions"
// @Success 200 {object} KpiDefinitionListResponse
// @Router /kpi-definitions [get]
func (h *Handler) ListDefinitions(c *gin.Context) {
	var periodYear, periodMonth *int
	if v := c.Query("period_year"); v != "" {
		if year, err := strconv.Atoi(v); err == nil {
			periodYear = &year
		}
	}
	if v := c.Query("period_month"); v != "" {
		if month, err := strconv.Atoi(v); err == nil {
			periodMonth = &month
		}
	}
	activeOnly := c.Query("active_only") == "true"
	resp, err := h.service.ListDefinitions(c.Request.Context(), periodYear, periodMonth, activeOnly)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// GetDefinition godoc
// @Summary Get a KPI definition
// @Tags kpi-definitions
// @Produce json
// @Param id path int64 true "KPI definition ID"
// @Success 200 {object} KpiDefinitionResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /kpi-definitions/{id} [get]
func (h *Handler) GetDefinition(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid definition ID", nil)
		return
	}
	resp, err := h.service.GetDefinition(c.Request.Context(), id)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// DeactivateDefinition godoc
// @Summary Deactivate a KPI definition
// @Tags kpi-definitions
// @Produce json
// @Param id path int64 true "KPI definition ID"
// @Success 204
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /kpi-definitions/{id}/deactivate [patch]
func (h *Handler) DeactivateDefinition(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid definition ID", nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	if err := h.service.DeactivateDefinition(c.Request.Context(), actor, id); err != nil {
		writeKpiError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

/* ---------- Recompute / Results / Ranking ---------- */

// TriggerRecompute godoc
// @Summary Enqueue a KPI recompute for a period
// @Description Asynchronous — processed by the background worker. Recompute is idempotent: re-running for the same period always produces identical results.
// @Tags kpi
// @Accept json
// @Produce json
// @Param request body RecomputeRequest true "Period to recompute"
// @Success 202 {object} JobResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /kpi/recompute [post]
func (h *Handler) TriggerRecompute(c *gin.Context) {
	var req RecomputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.TriggerRecompute(c.Request.Context(), actor, req)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusAccepted, resp)
}

// GetJob godoc
// @Summary Check a recompute job's status
// @Tags kpi
// @Produce json
// @Param id path int64 true "Job ID"
// @Success 200 {object} JobResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /kpi/jobs/{id} [get]
func (h *Handler) GetJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID", nil)
		return
	}
	resp, err := h.service.GetJob(c.Request.Context(), id)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListResults godoc
// @Summary List KPI results
// @Description Sales sees only their own result; Admin/Supervisor see all, optionally filtered.
// @Tags kpi
// @Produce json
// @Param sales_id query int64 false "Filter by Sales user ID"
// @Param period_year query int false "Filter by period year"
// @Param period_month query int false "Filter by period month"
// @Success 200 {object} SalesKpiResultListResponse
// @Router /kpi/results [get]
func (h *Handler) ListResults(c *gin.Context) {
	params := ListResultsParams{}
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
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListResults(c.Request.Context(), actor, params)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListRanking godoc
// @Summary List the full sales ranking for a period
// @Description Admin/Supervisor only.
// @Tags kpi
// @Produce json
// @Param period_year query int true "Period year"
// @Param period_month query int true "Period month"
// @Success 200 {object} SalesKpiResultListResponse
// @Failure 403 {object} httpx.ErrorEnvelope
// @Router /kpi/ranking [get]
func (h *Handler) ListRanking(c *gin.Context) {
	periodYear, err := strconv.Atoi(c.Query("period_year"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "period_year is required", nil)
		return
	}
	periodMonth, err := strconv.Atoi(c.Query("period_month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "period_month is required", nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListRanking(c.Request.Context(), actor, periodYear, periodMonth)
	if err != nil {
		writeKpiError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func writeKpiError(c *gin.Context, err error) {
	switch err {
	case nil:
		return
	case ErrNotFound, ErrJobNotFound:
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case ErrForbidden:
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case ErrInvalidMetric:
		httpx.Error(c, http.StatusBadRequest, "INVALID_METRIC", err.Error(), nil)
	case ErrInvalidPeriod:
		httpx.Error(c, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
	case ErrInvalidWeight:
		httpx.Error(c, http.StatusBadRequest, "INVALID_WEIGHT", err.Error(), nil)
	case ErrInvalidThreshold:
		httpx.Error(c, http.StatusBadRequest, "INVALID_THRESHOLD", err.Error(), nil)
	case ErrDuplicateDefinition:
		httpx.Error(c, http.StatusConflict, "DUPLICATE_DEFINITION", err.Error(), nil)
	case ErrNoActiveDefinitions:
		httpx.Error(c, http.StatusBadRequest, "NO_ACTIVE_DEFINITIONS", err.Error(), nil)
	case ErrWeightNotHundred:
		httpx.Error(c, http.StatusBadRequest, "WEIGHT_NOT_HUNDRED", err.Error(), nil)
	case ErrInconsistentThreshold:
		httpx.Error(c, http.StatusBadRequest, "INCONSISTENT_THRESHOLD", err.Error(), nil)
	case ErrUnsupportedMetric:
		httpx.Error(c, http.StatusBadRequest, "UNSUPPORTED_METRIC", err.Error(), nil)
	default:
		switch {
		case errors.Is(err, ErrNotFound), errors.Is(err, ErrJobNotFound):
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		case errors.Is(err, ErrForbidden):
			httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		case errors.Is(err, ErrInvalidMetric):
			httpx.Error(c, http.StatusBadRequest, "INVALID_METRIC", err.Error(), nil)
		case errors.Is(err, ErrInvalidPeriod):
			httpx.Error(c, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
		case errors.Is(err, ErrInvalidWeight):
			httpx.Error(c, http.StatusBadRequest, "INVALID_WEIGHT", err.Error(), nil)
		case errors.Is(err, ErrInvalidThreshold):
			httpx.Error(c, http.StatusBadRequest, "INVALID_THRESHOLD", err.Error(), nil)
		case errors.Is(err, ErrDuplicateDefinition):
			httpx.Error(c, http.StatusConflict, "DUPLICATE_DEFINITION", err.Error(), nil)
		case errors.Is(err, ErrNoActiveDefinitions):
			httpx.Error(c, http.StatusBadRequest, "NO_ACTIVE_DEFINITIONS", err.Error(), nil)
		case errors.Is(err, ErrWeightNotHundred):
			httpx.Error(c, http.StatusBadRequest, "WEIGHT_NOT_HUNDRED", err.Error(), nil)
		case errors.Is(err, ErrInconsistentThreshold):
			httpx.Error(c, http.StatusBadRequest, "INCONSISTENT_THRESHOLD", err.Error(), nil)
		case errors.Is(err, ErrUnsupportedMetric):
			httpx.Error(c, http.StatusBadRequest, "UNSUPPORTED_METRIC", err.Error(), nil)
		default:
			httpx.InternalServerError(c, "Terjadi kesalahan pada server", err)
		}
	}
}
