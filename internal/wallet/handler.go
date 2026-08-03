package wallet

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/wallets", h.listWallets)
	api.GET("/wallets/all", h.listAllWallets)
	api.GET("/wallets/all-deleted", h.listAllWallets)
	api.GET("/wallet-payments", h.listPayments)
	api.GET("/wallet-payments/all", h.listAllPayments)
	api.GET("/wallet-payments/all-deleted", h.listAllPayments)
	api.GET("/wallet-payments/:payment_id", h.getPayment)
	api.GET("/wallet-transactions", h.listTransactions)
	api.GET("/wallet-transactions/all", h.listAllTransactions)
	api.GET("/wallet-transactions/all-deleted", h.listAllTransactions)
	api.GET("/owners/:owner_id/wallet", h.getOwnerWallet)
	api.GET("/owners/:owner_id/wallet/transactions", h.listOwnerTransactions)
	api.GET("/owners/:owner_id/wallet/transactions/all", h.listAllOwnerTransactions)
	api.GET("/owners/:owner_id/wallet/transactions/all-deleted", h.listAllOwnerTransactions)
	api.POST("/owners/:owner_id/wallet/topups", h.createTopup)
	api.PATCH("/wallet-payments/:payment_id/accept", h.acceptTopup)
	api.PATCH("/wallet-payments/:payment_id/reject", h.rejectTopup)
	api.PATCH("/wallet-payments/:payment_id/transfer-date", h.setTransferDateOverride)
	api.POST("/owners/:owner_id/wallet/debits", h.createDebit)
	api.POST("/owners/:owner_id/wallet/adjustments", h.createAdjustment)
	api.POST("/owners/:owner_id/wallet/refunds", h.createRefund)
}

func (h *Handler) listWallets(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListWallets(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllWallets(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListWallets(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getOwnerWallet(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetOwnerWallet(c.Request.Context(), user, ownerID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listPayments(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListPayments(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllPayments(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListPayments(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getPayment(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	id, ok := parsePathID(c, "payment_id", "ID payment tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetPayment(c.Request.Context(), user, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listTransactions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListTransactions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllTransactions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListTransactions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listOwnerTransactions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	params, ok := listParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListOwnerTransactions(c.Request.Context(), user, ownerID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllOwnerTransactions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	params, ok := listParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListOwnerTransactions(c.Request.Context(), user, ownerID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createTopup(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	var req CreateTopupRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateTopup(c.Request.Context(), user, ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) acceptTopup(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	paymentID, ok := parsePathID(c, "payment_id", "ID top up tidak valid")
	if !ok {
		return
	}
	var req AcceptTopupRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.AcceptTopup(c.Request.Context(), user, paymentID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) rejectTopup(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	paymentID, ok := parsePathID(c, "payment_id", "ID top up tidak valid")
	if !ok {
		return
	}
	var req RejectTopupRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.RejectTopup(c.Request.Context(), user, paymentID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) setTransferDateOverride(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	paymentID, ok := parsePathID(c, "payment_id", "ID top up tidak valid")
	if !ok {
		return
	}
	var req SetTransferDateOverrideRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.SetTransferDateOverride(c.Request.Context(), user, paymentID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createDebit(c *gin.Context) { h.createManualTransaction(c, h.service.CreateDebit) }
func (h *Handler) createAdjustment(c *gin.Context) {
	h.createManualTransaction(c, h.service.CreateAdjustment)
}
func (h *Handler) createRefund(c *gin.Context) { h.createManualTransaction(c, h.service.CreateRefund) }

type manualAction func(gin.Context, identity.User, int64, CreateWalletTransactionRequest) (WalletTransactionResponse, error)

func (h *Handler) createManualTransaction(c *gin.Context, action func(context.Context, identity.User, int64, CreateWalletTransactionRequest) (WalletTransactionResponse, error)) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	var req CreateWalletTransactionRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := action(c.Request.Context(), user, ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func listParams(c *gin.Context) (ListParams, bool) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := ListParams{Query: c.Query("q"), Status: c.Query("status"), PaymentType: c.Query("payment_type"), Channel: c.Query("channel"), Direction: c.Query("direction"), Type: c.Query("type"), All: false, Page: page, Limit: limit, Sort: c.Query("sort")}
	if !parseOptionalInt64Query(c, "owner_id", &params.OwnerID) {
		return ListParams{}, false
	}
	var err error
	params.PaidFrom, err = parseDate(c.Query("paid_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "paid_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.PaidTo, err = parseDate(c.Query("paid_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "paid_to harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.OccurredFrom, err = parseDate(c.Query("occurred_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "occurred_from harus format YYYY-MM-DD", nil)
		return ListParams{}, false
	}
	params.OccurredTo, err = parseDate(c.Query("occurred_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "occurred_to harus format YYYY-MM-DD", nil)
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

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrOwnerNotFound):
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
	case errors.Is(err, ErrInvalidDirection):
		httpx.Error(c, http.StatusBadRequest, "INVALID_DIRECTION", err.Error(), nil)
	case errors.Is(err, ErrLedgerOutOfSync):
		httpx.Error(c, http.StatusConflict, "LEDGER_OUT_OF_SYNC", err.Error(), nil)
	case errors.Is(err, ErrTopupNotPending):
		httpx.Error(c, http.StatusConflict, "TOPUP_NOT_PENDING", err.Error(), nil)
	case errors.Is(err, ErrInvalidRequest):
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
