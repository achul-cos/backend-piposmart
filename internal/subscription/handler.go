package subscription

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/subscription-orders", h.listOrders)
	api.GET("/subscription-orders/all", h.listAllOrders)
	api.GET("/subscription-orders/all-deleted", h.listAllOrders)
	api.GET("/subscription-orders/:order_id", h.getOrder)
	api.POST("/owners/:owner_id/subscription-orders", h.createOrder)
	api.POST("/subscription-orders/:order_id/reconcile", h.reconcileOrder)
	api.GET("/subscriptions", h.listSubscriptions)
	api.GET("/subscriptions/all", h.listAllSubscriptions)
	api.GET("/subscriptions/all-deleted", h.listAllSubscriptions)
	api.GET("/subscriptions/:subscription_id", h.getSubscription)
	api.GET("/reconciliations", h.listReconciliations)
	api.GET("/reconciliations/all", h.listAllReconciliations)
	api.GET("/reconciliations/all-deleted", h.listAllReconciliations)
	api.GET("/reconciliation-issues", h.listIssues)
	api.GET("/reconciliation-issues/all", h.listAllIssues)
	api.GET("/reconciliation-issues/all-deleted", h.listAllIssues)
}

func (h *Handler) listOrders(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListOrders(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllOrders(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListOrders(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getOrder(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parsePathID(c, "order_id", "ID subscription order tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetOrder(c.Request.Context(), user, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createOrder(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	var req CreateOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateOrder(c.Request.Context(), user, ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) reconcileOrder(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	orderID, ok := parsePathID(c, "order_id", "ID subscription order tidak valid")
	if !ok {
		return
	}
	var req ManualReconcileRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.ReconcileOrder(c.Request.Context(), user, orderID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listSubscriptions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListSubscriptions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllSubscriptions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListSubscriptions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getSubscription(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parsePathID(c, "subscription_id", "ID subscription tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetSubscription(c.Request.Context(), user, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listReconciliations(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListReconciliations(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllReconciliations(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListReconciliations(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listIssues(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListIssues(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllIssues(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListIssues(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func listParams(c *gin.Context) (ListParams, bool) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := ListParams{Query: c.Query("q"), Status: c.Query("status"), IssueType: c.Query("issue_type"), All: false, Page: page, Limit: limit, Sort: c.Query("sort")}
	if !parseOptionalInt64Query(c, "owner_id", &params.OwnerID) || !parseOptionalInt64Query(c, "outlet_id", &params.OutletID) || !parseOptionalInt64Query(c, "order_id", &params.OrderID) || !parseOptionalInt64Query(c, "closing_id", &params.ClosingID) || !parseOptionalInt64Query(c, "sales_id", &params.SalesID) || !parseOptionalInt64Query(c, "supervisor_id", &params.SupervisorID) || !parseOptionalInt64Query(c, "plan_id", &params.PlanID) {
		return ListParams{}, false
	}
	var err error
	params.PurchasedFrom, err = parseDate(c.Query("purchased_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "purchased_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.PurchasedTo, err = parseDate(c.Query("purchased_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "purchased_to harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.ActiveFrom, err = parseDate(c.Query("active_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "active_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.ActiveTo, err = parseDate(c.Query("active_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "active_to harus format YYYY-MM-DD", nil)
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

func parseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrOwnerNotFound), errors.Is(err, ErrWalletNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, ErrInvalidSort):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SORT", err.Error(), nil)
	case errors.Is(err, ErrInvalidMoney):
		httpx.Error(c, http.StatusBadRequest, "INVALID_DECIMAL", err.Error(), nil)
	case errors.Is(err, ErrInsufficientBalance):
		httpx.Error(c, http.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error(), nil)
	case errors.Is(err, ErrIdempotencyRequired):
		httpx.Error(c, http.StatusBadRequest, "IDEMPOTENCY_REQUIRED", err.Error(), nil)
	case errors.Is(err, ErrInvalidPromotion):
		httpx.Error(c, http.StatusBadRequest, "INVALID_PROMOTION", err.Error(), nil)
	case errors.Is(err, ErrClosingMismatch):
		httpx.Error(c, http.StatusBadRequest, "CLOSING_MISMATCH", err.Error(), nil)
	case errors.Is(err, ErrOrderAlreadyReconciled):
		httpx.Error(c, http.StatusConflict, "ORDER_ALREADY_RECONCILED", err.Error(), nil)
	case errors.Is(err, ErrInvalidAction):
		httpx.Error(c, http.StatusBadRequest, "INVALID_ACTION", err.Error(), nil)
	case errors.Is(err, ErrInvalidDate):
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, ErrInvalidRequest):
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
