package transfer

import (
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
	api.POST("/owners/:owner_id/transfers", h.createTransfer)
	api.GET("/owners/:owner_id/transfers", h.listOwnerTransfers)
	api.GET("/owners/:owner_id/transfers/suggestions", h.suggestMatches)
	api.GET("/transfers", h.listTransfers)
	api.GET("/transfers/:transfer_id", h.getTransfer)
	api.POST("/transfers/:transfer_id/confirm-match", h.confirmMatch)
	api.POST("/transfers/:transfer_id/reject-match", h.rejectMatch)
}

func parsePathID(c *gin.Context, param, message string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || value < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", message, nil)
		return 0, false
	}
	return value, true
}

func (h *Handler) createTransfer(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	var req CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "payload tidak valid", gin.H{"error": err.Error()})
		return
	}
	response, err := h.service.CreateTransfer(c.Request.Context(), user, ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) listOwnerTransfers(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	params := listParams(c)
	params.OwnerID = &ownerID
	response, err := h.service.ListTransfers(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listTransfers(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params := listParams(c)
	if raw := c.Query("owner_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			params.OwnerID = &v
		}
	}
	response, err := h.service.ListTransfers(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getTransfer(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	transferID, ok := parsePathID(c, "transfer_id", "ID transfer tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetTransfer(c.Request.Context(), user, transferID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) suggestMatches(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	ownerID, ok := parsePathID(c, "owner_id", "ID owner tidak valid")
	if !ok {
		return
	}
	suggestions, err := h.service.SuggestMatches(c.Request.Context(), user, ownerID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"items": suggestions})
}

func (h *Handler) confirmMatch(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	transferID, ok := parsePathID(c, "transfer_id", "ID transfer tidak valid")
	if !ok {
		return
	}
	var req ConfirmMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "payload tidak valid", gin.H{"error": err.Error()})
		return
	}
	response, err := h.service.ConfirmMatch(c.Request.Context(), user, transferID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) rejectMatch(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	transferID, ok := parsePathID(c, "transfer_id", "ID transfer tidak valid")
	if !ok {
		return
	}
	var req RejectMatchRequest
	_ = c.ShouldBindJSON(&req)
	response, err := h.service.RejectMatch(c.Request.Context(), user, transferID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func listParams(c *gin.Context) ListParams {
	params := ListParams{
		MatchStatus: c.Query("match_status"),
		Sort:        c.Query("sort"),
	}
	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		params.Page = page
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		params.Limit = limit
	}
	if c.Query("all") == "true" {
		params.All = true
	}
	return params
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrOwnerNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, ErrInvalidMoney):
		httpx.Error(c, http.StatusBadRequest, "INVALID_DECIMAL", err.Error(), nil)
	case errors.Is(err, ErrAlreadyMatched):
		httpx.Error(c, http.StatusConflict, "ALREADY_MATCHED", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
