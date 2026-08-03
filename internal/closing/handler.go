package closing

import (
	"context"
	"errors"
	"io"
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
	api.GET("/closings", h.listClosings)
	api.GET("/closings/all", h.listAllClosings)
	api.GET("/closings/all-deleted", h.listAllClosingsDeleted)
	api.GET("/closings/trash", h.listClosingsTrash)
	api.GET("/closings/unscoped", h.listClosingsUnscoped)
	api.DELETE("/closings/bulk", h.bulkDeleteClosings)
	api.PATCH("/closings/bulk/restore", h.bulkRestoreClosings)
	api.DELETE("/closings/bulk/force", h.bulkForceDeleteClosings)
	api.GET("/closings/:closing_id", h.getClosing)
	api.POST("/closings/:closing_id/confirm", h.confirmClosing)
	api.POST("/closings/:closing_id/reject", h.rejectClosing)
	api.DELETE("/closings/:closing_id", h.deleteClosing)
	api.PATCH("/closings/:closing_id/restore", h.restoreClosing)
	api.DELETE("/closings/:closing_id/force", h.forceDeleteClosing)
	api.POST("/leads/:lead_id/closings", h.createLeadClosing)
}

func (h *Handler) listClosings(c *gin.Context) {
	h.listClosingsWithScope(c, ScopeActive)
}

func (h *Handler) listAllClosings(c *gin.Context) {
	h.listClosingsWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllClosingsDeleted(c *gin.Context) {
	h.listClosingsWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listClosingsTrash(c *gin.Context) {
	h.listClosingsWithScope(c, ScopeDeleted)
}

func (h *Handler) listClosingsUnscoped(c *gin.Context) {
	h.listClosingsWithScope(c, ScopeAll)
}

func (h *Handler) listClosingsWithScope(c *gin.Context, scope string) {
	h.listClosingsWithScopeAndAll(c, scope, false)
}

func (h *Handler) listClosingsWithScopeAndAll(c *gin.Context, scope string, all bool) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.Scope = scope
	params.All = all
	response, err := h.service.ListClosings(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getClosing(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parsePathID(c, "closing_id", "ID closing tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetClosing(c.Request.Context(), user, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createLeadClosing(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	var req CreateClosingRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateClosing(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) confirmClosing(c *gin.Context) {
	h.updateStatus(c, h.service.ConfirmClosing)
}

func (h *Handler) rejectClosing(c *gin.Context) {
	h.updateStatus(c, h.service.RejectClosing)
}

type statusAction func(context.Context, identity.User, int64, UpdateClosingStatusRequest) (ClosingResponse, error)

func (h *Handler) updateStatus(c *gin.Context, action statusAction) {
	user, _ := identity.CurrentUser(c)
	id, ok := parsePathID(c, "closing_id", "ID closing tidak valid")
	if !ok {
		return
	}
	var req UpdateClosingStatusRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	response, err := action(c.Request.Context(), user, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) deleteClosing(c *gin.Context) {
	h.bulkFromPath(c, h.service.DeleteClosings)
}

func (h *Handler) restoreClosing(c *gin.Context) {
	h.bulkFromPath(c, h.service.RestoreClosings)
}

func (h *Handler) forceDeleteClosing(c *gin.Context) {
	h.bulkFromPath(c, h.service.ForceDeleteClosings)
}

func (h *Handler) bulkDeleteClosings(c *gin.Context) {
	h.bulkFromBody(c, h.service.DeleteClosings)
}

func (h *Handler) bulkRestoreClosings(c *gin.Context) {
	h.bulkFromBody(c, h.service.RestoreClosings)
}

func (h *Handler) bulkForceDeleteClosings(c *gin.Context) {
	h.bulkFromBody(c, h.service.ForceDeleteClosings)
}

type bulkAction func(context.Context, identity.User, []int64) (BulkActionResponse, error)

func (h *Handler) bulkFromPath(c *gin.Context, action bulkAction) {
	user, _ := identity.CurrentUser(c)
	id, ok := parsePathID(c, "closing_id", "ID closing tidak valid")
	if !ok {
		return
	}
	response, err := action(c.Request.Context(), user, []int64{id})
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkFromBody(c *gin.Context, action bulkAction) {
	user, _ := identity.CurrentUser(c)
	var req BulkIDRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := action(c.Request.Context(), user, req.IDs)
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
		Query:  c.Query("q"),
		Status: c.Query("status"),
		All:    false,
		Page:   page,
		Limit:  limit,
		Sort:   c.Query("sort"),
	}
	if !parseOptionalInt64Query(c, "lead_id", &params.LeadID) ||
		!parseOptionalInt64Query(c, "owner_id", &params.OwnerID) ||
		!parseOptionalInt64Query(c, "sales_id", &params.SalesID) ||
		!parseOptionalInt64Query(c, "supervisor_id", &params.SupervisorID) ||
		!parseOptionalInt64Query(c, "plan_id", &params.PlanID) {
		return ListParams{}, false
	}
	var err error
	params.ClosedFrom, err = parseDate(c.Query("closed_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "closed_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.ClosedTo, err = parseDate(c.Query("closed_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "closed_to harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.CreatedFrom, err = parseDate(c.Query("created_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "created_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.CreatedTo, err = parseDate(c.Query("created_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "created_to harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	return params, true
}

func parseOptionalInt64Query(c *gin.Context, name string, target **int64) bool {
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

func parsePathID(c *gin.Context, name, message string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", message, nil)
		return 0, false
	}
	return value, true
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func bindOptionalJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, ErrInvalidSort):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SORT", err.Error(), nil)
	case errors.Is(err, ErrInvalidMoney):
		httpx.Error(c, http.StatusBadRequest, "INVALID_DECIMAL", err.Error(), nil)
	case errors.Is(err, ErrInvalidStatus):
		httpx.Error(c, http.StatusBadRequest, "INVALID_STATUS", err.Error(), nil)
	case errors.Is(err, ErrInvalidPromotion):
		httpx.Error(c, http.StatusBadRequest, "INVALID_PROMOTION", err.Error(), nil)
	case errors.Is(err, ErrAlreadyHasClosing):
		httpx.Error(c, http.StatusConflict, "LEAD_ALREADY_HAS_CLOSING", err.Error(), nil)
	case errors.Is(err, ErrFinalAmountNegative):
		httpx.Error(c, http.StatusBadRequest, "FINAL_AMOUNT_NEGATIVE", err.Error(), nil)
	case errors.Is(err, ErrInvalidRequest):
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, ErrLeadHasNoPIC):
		httpx.Error(c, http.StatusBadRequest, "LEAD_HAS_NO_PIC", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
