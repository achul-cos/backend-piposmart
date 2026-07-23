package lead

import (
	"errors"
	"net/http"
	"strconv"

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
	leads := api.Group("/leads")
	leads.GET("", h.listLeads)
	leads.POST("", h.createLead)
	leads.POST("/bulk/assign-supervisor", h.bulkAssignSupervisor)
	leads.POST("/bulk/assign-sales", h.bulkAssignSales)
	leads.POST("/bulk/release", h.bulkRelease)
	leads.GET("/:lead_id", h.getLead)
	leads.GET("/:lead_id/assignment-history", h.assignmentHistory)
	leads.POST("/:lead_id/assign-supervisor", h.assignSupervisor)
	leads.POST("/:lead_id/assign-sales", h.assignSales)
	leads.POST("/:lead_id/release", h.release)
	leads.POST("/:lead_id/mark-invalid", h.markInvalid)
}

func (h *Handler) listLeads(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListLeads(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createLead(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req CreateLeadRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateLead(c.Request.Context(), user, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) getLead(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parseLeadID(c)
	if !ok {
		return
	}
	response, err := h.service.GetLead(c.Request.Context(), user, leadID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) assignmentHistory(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parseLeadID(c)
	if !ok {
		return
	}
	response, err := h.service.AssignmentHistory(c.Request.Context(), user, leadID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) assignSupervisor(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parseLeadID(c)
	if !ok {
		return
	}
	var req AssignSupervisorRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.AssignSupervisor(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkAssignSupervisor(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req BulkAssignSupervisorRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkAssignSupervisor(c.Request.Context(), user, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) assignSales(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parseLeadID(c)
	if !ok {
		return
	}
	var req AssignSalesRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.AssignSales(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkAssignSales(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req BulkAssignSalesRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkAssignSales(c.Request.Context(), user, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) release(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parseLeadID(c)
	if !ok {
		return
	}
	var req ReleaseRequest
	_ = c.ShouldBindJSON(&req)
	response, err := h.service.Release(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkRelease(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	var req BulkReleaseRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkRelease(c.Request.Context(), user, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) markInvalid(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parseLeadID(c)
	if !ok {
		return
	}
	var req MarkInvalidRequest
	_ = c.ShouldBindJSON(&req)
	response, err := h.service.MarkInvalid(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func listParams(c *gin.Context) (ListParams, bool) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := ListParams{
		Query:     c.Query("q"),
		Ownership: c.Query("ownership"),
		Stage:     c.Query("stage"),
		Status:    c.Query("status"),
		Page:      page,
		Limit:     limit,
		Sort:      c.Query("sort"),
	}
	if score := c.Query("score"); score != "" {
		value, err := strconv.ParseInt(score, 10, 64)
		if err != nil || value < 0 || value > 3 {
			httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "score harus angka 0 sampai 3", nil)
			return ListParams{}, false
		}
		params.Score = &value
	}
	if supervisorID := c.Query("supervisor_id"); supervisorID != "" {
		value, err := strconv.ParseInt(supervisorID, 10, 64)
		if err != nil || value < 1 {
			httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "supervisor_id tidak valid", nil)
			return ListParams{}, false
		}
		params.SupervisorID = &value
	}
	if salesID := c.Query("sales_id"); salesID != "" {
		value, err := strconv.ParseInt(salesID, 10, 64)
		if err != nil || value < 1 {
			httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "sales_id tidak valid", nil)
			return ListParams{}, false
		}
		params.SalesID = &value
	}
	followUpFrom, err := parseDate(c.Query("follow_up_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "follow_up_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	followUpTo, err := parseDate(c.Query("follow_up_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "follow_up_to harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.FollowUpFrom = followUpFrom
	params.FollowUpTo = followUpTo
	return params, true
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func parseLeadID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("lead_id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID lead tidak valid", nil)
		return 0, false
	}
	return id, true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(c, http.StatusBadRequest, "INVALID_TRANSITION", err.Error(), nil)
	case errors.Is(err, ErrInvalidSort):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SORT", err.Error(), nil)
	case errors.Is(err, ErrEmptyBulk):
		httpx.Error(c, http.StatusBadRequest, "EMPTY_BULK", err.Error(), nil)
	case errors.Is(err, ErrUserNotValid):
		httpx.Error(c, http.StatusBadRequest, "USER_NOT_VALID", err.Error(), nil)
	case errors.Is(err, ErrLeadAlreadyExists):
		httpx.Error(c, http.StatusConflict, "LEAD_ALREADY_EXISTS", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
