package customer

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
	owners := api.Group("/owners")
	owners.GET("", h.listOwners)
	owners.POST("", h.createOwner)
	owners.POST("/bulk", h.bulkCreateOwners)
	owners.PATCH("/bulk", h.bulkUpdateOwners)
	owners.DELETE("/bulk", h.bulkDeleteOwners)
	owners.DELETE("/bulk/force", h.bulkForceDeleteOwners)
	owners.GET("/:owner_id", h.getOwner)
	owners.PATCH("/:owner_id", h.updateOwner)
	owners.DELETE("/:owner_id", h.deleteOwner)
	owners.PATCH("/:owner_id/restore", h.restoreOwner)
	owners.DELETE("/:owner_id/force", h.forceDeleteOwner)

	owners.GET("/:owner_id/outlets", h.listOutlets)
	owners.POST("/:owner_id/outlets", h.createOutlet)
	owners.POST("/:owner_id/outlets/bulk", h.bulkCreateOutlets)
	owners.PATCH("/:owner_id/outlets/bulk", h.bulkUpdateOutlets)
	owners.DELETE("/:owner_id/outlets/bulk", h.bulkDeleteOutlets)
	owners.DELETE("/:owner_id/outlets/bulk/force", h.bulkForceDeleteOutlets)
	owners.GET("/:owner_id/outlets/:outlet_id", h.getOutlet)
	owners.PATCH("/:owner_id/outlets/:outlet_id", h.updateOutlet)
	owners.DELETE("/:owner_id/outlets/:outlet_id", h.deleteOutlet)
	owners.PATCH("/:owner_id/outlets/:outlet_id/restore", h.restoreOutlet)
	owners.DELETE("/:owner_id/outlets/:outlet_id/force", h.forceDeleteOutlet)
}

func (h *Handler) listOwners(c *gin.Context) {
	response, err := h.service.ListOwners(c.Request.Context(), currentActor(c), listParams(c))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createOwner(c *gin.Context) {
	var req CreateOwnerRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateOwner(c.Request.Context(), currentActor(c), req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) bulkCreateOwners(c *gin.Context) {
	var req BulkOwnerCreateRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkCreateOwners(c.Request.Context(), currentActor(c), req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) getOwner(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	response, err := h.service.GetOwner(c.Request.Context(), currentActor(c), ownerID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) restoreOwner(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	response, err := h.service.RestoreOwner(c.Request.Context(), currentActor(c), ownerID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) updateOwner(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req UpdateOwnerRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.UpdateOwner(c.Request.Context(), currentActor(c), ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkUpdateOwners(c *gin.Context) {
	var req BulkOwnerUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkUpdateOwners(c.Request.Context(), currentActor(c), req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) deleteOwner(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	if err := h.service.DeleteOwner(c.Request.Context(), currentActor(c), ownerID); err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) forceDeleteOwner(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	if err := h.service.ForceDeleteOwner(c.Request.Context(), currentActor(c), ownerID); err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"status": "force_deleted"})
}

func (h *Handler) bulkDeleteOwners(c *gin.Context) {
	var req BulkIDRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkDeleteOwners(c.Request.Context(), currentActor(c), req.IDs)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkForceDeleteOwners(c *gin.Context) {
	var req BulkIDRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkForceDeleteOwners(c.Request.Context(), currentActor(c), req.IDs)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listOutlets(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	response, err := h.service.ListOutlets(c.Request.Context(), currentActor(c), ownerID, listParams(c))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createOutlet(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req CreateOutletRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateOutlet(c.Request.Context(), currentActor(c), ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) bulkCreateOutlets(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req BulkOutletCreateRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkCreateOutlets(c.Request.Context(), currentActor(c), ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) getOutlet(c *gin.Context) {
	ownerID, outletID, ok := parseOwnerOutletID(c)
	if !ok {
		return
	}
	response, err := h.service.GetOutlet(c.Request.Context(), currentActor(c), ownerID, outletID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) restoreOutlet(c *gin.Context) {
	ownerID, outletID, ok := parseOwnerOutletID(c)
	if !ok {
		return
	}
	response, err := h.service.RestoreOutlet(c.Request.Context(), currentActor(c), ownerID, outletID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) updateOutlet(c *gin.Context) {
	ownerID, outletID, ok := parseOwnerOutletID(c)
	if !ok {
		return
	}
	var req UpdateOutletRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.UpdateOutlet(c.Request.Context(), currentActor(c), ownerID, outletID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkUpdateOutlets(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req BulkOutletUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkUpdateOutlets(c.Request.Context(), currentActor(c), ownerID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) deleteOutlet(c *gin.Context) {
	ownerID, outletID, ok := parseOwnerOutletID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteOutlet(c.Request.Context(), currentActor(c), ownerID, outletID); err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) forceDeleteOutlet(c *gin.Context) {
	ownerID, outletID, ok := parseOwnerOutletID(c)
	if !ok {
		return
	}
	if err := h.service.ForceDeleteOutlet(c.Request.Context(), currentActor(c), ownerID, outletID); err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, gin.H{"status": "force_deleted"})
}

func (h *Handler) bulkDeleteOutlets(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req BulkIDRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkDeleteOutlets(c.Request.Context(), currentActor(c), ownerID, req.IDs)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) bulkForceDeleteOutlets(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req BulkIDRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.BulkForceDeleteOutlets(c.Request.Context(), currentActor(c), ownerID, req.IDs)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Payload request tidak valid", gin.H{"error": err.Error()})
		return false
	}
	return true
}

func parseOwnerOutletID(c *gin.Context) (int64, int64, bool) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return 0, 0, false
	}
	outletID, ok := parseParamID(c, "outlet_id")
	if !ok {
		return 0, 0, false
	}
	return ownerID, outletID, true
}

func parseParamID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "ID tidak valid", nil)
		return 0, false
	}
	return id, true
}

func listParams(c *gin.Context) ListParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	return ListParams{
		Query:     c.Query("q"),
		Code:      c.Query("code"),
		Name:      c.Query("name"),
		Phone:     c.Query("phone"),
		BrandName: c.Query("brand_name"),
		Province:  c.Query("province"),
		City:      c.Query("city"),
		Page:      page,
		Limit:     limit,
		Sort:      c.Query("sort"),
	}
}

func currentActor(c *gin.Context) Actor {
	user, _ := identity.CurrentUser(c)
	return Actor{ID: user.ID, RoleCode: user.RoleCode}
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrCodeAlreadyUsed):
		httpx.Error(c, http.StatusConflict, "CODE_ALREADY_USED", err.Error(), nil)
	case errors.Is(err, ErrInvalidPhone):
		httpx.Error(c, http.StatusBadRequest, "INVALID_PHONE", err.Error(), nil)
	case errors.Is(err, ErrInvalidSort):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SORT", err.Error(), nil)
	case errors.Is(err, ErrEmptyBulk):
		httpx.Error(c, http.StatusBadRequest, "EMPTY_BULK", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil)
	}
}
