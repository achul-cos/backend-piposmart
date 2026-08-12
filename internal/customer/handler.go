package customer

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"
	"backend_crm_piposmart/internal/reporting"

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
	owners.GET("/all", h.listAllOwners)
	owners.GET("/all-deleted", h.listAllOwnersDeleted)
	owners.GET("/trash", h.listOwnersTrash)
	owners.GET("/unscoped", h.listOwnersUnscoped)
	owners.POST("", h.createOwner)
	owners.POST("/bulk", h.bulkCreateOwners)
	owners.PATCH("/bulk", h.bulkUpdateOwners)
	owners.DELETE("/bulk", h.bulkDeleteOwners)
	owners.DELETE("/bulk/force", h.bulkForceDeleteOwners)
	owners.GET("/export", h.exportOwners)
	owners.GET("/export/download", h.downloadOwnerOutletExcel)
	owners.GET("/export/download-owner", h.downloadOwnerExcel)
	owners.GET("/:owner_id", h.getOwner)
	owners.GET("/:owner_id/overview", h.getOwnerOverview)
	owners.GET("/:owner_id/history", h.listOwnerHistories)
	owners.PATCH("/:owner_id", h.updateOwner)
	owners.PATCH("/:owner_id/testing-account", h.setOwnerTestingAccount)
	owners.DELETE("/:owner_id", h.deleteOwner)
	owners.PATCH("/:owner_id/restore", h.restoreOwner)
	owners.DELETE("/:owner_id/force", h.forceDeleteOwner)

	owners.GET("/:owner_id/outlets", h.listOutlets)
	owners.GET("/:owner_id/outlets/all", h.listAllOutlets)
	owners.GET("/:owner_id/outlets/all-deleted", h.listAllOutletsDeleted)
	owners.GET("/:owner_id/outlets/trash", h.listOutletsTrash)
	owners.GET("/:owner_id/outlets/unscoped", h.listOutletsUnscoped)
	owners.POST("/:owner_id/outlets", h.createOutlet)
	owners.POST("/:owner_id/outlets/bulk", h.bulkCreateOutlets)
	owners.PATCH("/:owner_id/outlets/bulk", h.bulkUpdateOutlets)
	owners.DELETE("/:owner_id/outlets/bulk", h.bulkDeleteOutlets)
	owners.DELETE("/:owner_id/outlets/bulk/force", h.bulkForceDeleteOutlets)
	owners.GET("/:owner_id/outlets/:outlet_id", h.getOutlet)
	owners.GET("/:owner_id/outlets/:outlet_id/history", h.listOutletHistories)
	owners.PATCH("/:owner_id/outlets/:outlet_id", h.updateOutlet)
	owners.DELETE("/:owner_id/outlets/:outlet_id", h.deleteOutlet)
	owners.PATCH("/:owner_id/outlets/:outlet_id/restore", h.restoreOutlet)
	owners.DELETE("/:owner_id/outlets/:outlet_id/force", h.forceDeleteOutlet)

	outlets := api.Group("/outlets")
	outlets.GET("", h.listGlobalOutlets)
	outlets.GET("/all", h.listAllGlobalOutlets)
	outlets.GET("/all-deleted", h.listAllGlobalOutletsDeleted)
	outlets.GET("/trash", h.listGlobalOutletsTrash)
	outlets.GET("/unscoped", h.listGlobalOutletsUnscoped)
	outlets.GET("/export/download", h.downloadGlobalOutletExcel)
	outlets.GET("/subscription-statuses", h.listOutletSubscriptionStatuses)
	outlets.GET("/subscription-statuses/all", h.listAllOutletSubscriptionStatuses)
	outlets.GET("/:outlet_id", h.getGlobalOutlet)
}

func (h *Handler) listOwners(c *gin.Context) {
	h.listOwnersWithScope(c, ScopeActive)
}

func (h *Handler) listAllOwners(c *gin.Context) {
	h.listOwnersWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllOwnersDeleted(c *gin.Context) {
	h.listOwnersWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listOwnersTrash(c *gin.Context) {
	h.listOwnersWithScope(c, ScopeDeleted)
}

func (h *Handler) listOwnersUnscoped(c *gin.Context) {
	h.listOwnersWithScope(c, ScopeAll)
}

func (h *Handler) listOwnersWithScope(c *gin.Context, scope string) {
	h.listOwnersWithScopeAndAll(c, scope, false)
}

func (h *Handler) listOwnersWithScopeAndAll(c *gin.Context, scope string, all bool) {
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.Scope = scope
	params.All = all
	response, err := h.service.ListOwners(c.Request.Context(), currentActor(c), params)
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

func (h *Handler) exportOwners(c *gin.Context) {
	actor := currentActor(c)
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.All = true

	rows, err := h.service.ExportOwnerOutlets(c.Request.Context(), actor, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, rows)
}

// exportDateParams reads the same date_from/date_to (and start_date/end_date) aliases the download
// handlers have always accepted, on top of whatever listParams already recognizes
// (created_from/created_to), so switching these handlers to the DB-driven path doesn't silently
// drop a filter the frontend is already sending.
func exportDateParams(c *gin.Context, params *ListParams) (dateFromDisplay, dateToDisplay string) {
	dateFromDisplay = c.Query("date_from")
	if dateFromDisplay == "" {
		dateFromDisplay = c.Query("start_date")
	}
	dateToDisplay = c.Query("date_to")
	if dateToDisplay == "" {
		dateToDisplay = c.Query("end_date")
	}
	if params.CreatedFrom == nil && dateFromDisplay != "" {
		if t, err := parseDateOnly(dateFromDisplay); err == nil {
			params.CreatedFrom = t
		}
	}
	if params.CreatedTo == nil && dateToDisplay != "" {
		if t, err := parseDateOnly(dateToDisplay); err == nil {
			params.CreatedTo = t
		}
	}
	return dateFromDisplay, dateToDisplay
}

// downloadOwnerOutletExcel — download file Excel Data Owner-Outlet, sumber data dari database
// (bukan dari file Excel asli) supaya selalu mencerminkan state CRM saat ini, termasuk owner/outlet
// hasil migrasi seeder.
func (h *Handler) downloadOwnerOutletExcel(c *gin.Context) {
	if !actorCanManageOwners(currentActor(c)) {
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "not allowed to perform this action", nil)
		return
	}
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.All = true
	dateFrom, dateTo := exportDateParams(c, &params)

	rows, err := h.service.ExportOwnerOutlets(c.Request.Context(), currentActor(c), params)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "EXPORT_ERROR", err.Error(), nil)
		return
	}
	items := ToAdminOwnerOutletRows(rows)
	data, err := reporting.BuildAdminOwnerOutletXLSX("Report Owner & Outlet", "Data Owner Outlet", reporting.GetAdminOwnerOutletColumns(), items, dateFrom, dateTo)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "EXPORT_ERROR", err.Error(), nil)
		return
	}

	filename := "Data_Owner_Outlet.xlsx"
	if dateFrom != "" && dateTo != "" {
		filename = fmt.Sprintf("Data_Owner_Outlet_%s_%s.xlsx", dateFrom, dateTo)
	} else {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("Data_Owner_Outlet_%s.xlsx", timestamp)
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// downloadOwnerExcel — download file Excel Data Owner (unique owners), sumber data dari database.
func (h *Handler) downloadOwnerExcel(c *gin.Context) {
	if !actorCanManageOwners(currentActor(c)) {
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "not allowed to perform this action", nil)
		return
	}
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.All = true
	dateFrom, dateTo := exportDateParams(c, &params)

	rows, err := h.service.ExportOwnerOutlets(c.Request.Context(), currentActor(c), params)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "EXPORT_ERROR", err.Error(), nil)
		return
	}
	items := ToAdminOwnerRows(rows)
	data, err := reporting.BuildAdminOwnerOutletXLSX("Report Data Owner", "Data Owner", reporting.GetAdminOwnerColumns(), items, dateFrom, dateTo)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "EXPORT_ERROR", err.Error(), nil)
		return
	}

	filename := "Data_Owner.xlsx"
	if dateFrom != "" && dateTo != "" {
		filename = fmt.Sprintf("Data_Owner_%s_%s.xlsx", dateFrom, dateTo)
	} else {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("Data_Owner_%s.xlsx", timestamp)
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// downloadGlobalOutletExcel â€” download file Excel Data Owner-Outlet berdasarkan filter outlet global.
func (h *Handler) downloadGlobalOutletExcel(c *gin.Context) {
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.All = true
	dateFrom, dateTo := exportDateParams(c, &params)

	rows, err := h.service.ExportGlobalOutlets(c.Request.Context(), currentActor(c), params)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "EXPORT_ERROR", err.Error(), nil)
		return
	}
	items := ToAdminOwnerOutletRows(rows)
	data, err := reporting.BuildAdminOwnerOutletXLSX("Report Owner & Outlet", "Data Owner Outlet", reporting.GetAdminOwnerOutletColumns(), items, dateFrom, dateTo)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "EXPORT_ERROR", err.Error(), nil)
		return
	}

	filename := "Data_Owner_Outlet.xlsx"
	if dateFrom != "" && dateTo != "" {
		filename = fmt.Sprintf("Data_Owner_Outlet_%s_%s.xlsx", dateFrom, dateTo)
	} else {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("Data_Owner_Outlet_%s.xlsx", timestamp)
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
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

func (h *Handler) getOwnerOverview(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	response, err := h.service.GetOwnerOverview(c.Request.Context(), currentActor(c), ownerID)
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
	meta := RequestMeta{IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), RequestID: httpx.RequestID(c)}
	response, err := h.service.UpdateOwnerWithMeta(c.Request.Context(), currentActor(c), ownerID, req, meta)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listOwnerHistories(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	response, err := h.service.ListOwnerHistories(c.Request.Context(), currentActor(c), ownerID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

type setTestingAccountRequest struct {
	IsTestingAccount bool `json:"is_testing_account"`
}

func (h *Handler) setOwnerTestingAccount(c *gin.Context) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	var req setTestingAccountRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.SetTestingAccount(c.Request.Context(), currentActor(c), ownerID, req.IsTestingAccount)
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
	h.listOwnerOutletsWithScope(c, ScopeActive)
}

func (h *Handler) listAllOutlets(c *gin.Context) {
	h.listOwnerOutletsWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllOutletsDeleted(c *gin.Context) {
	h.listOwnerOutletsWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listOutletsTrash(c *gin.Context) {
	h.listOwnerOutletsWithScope(c, ScopeDeleted)
}

func (h *Handler) listOutletsUnscoped(c *gin.Context) {
	h.listOwnerOutletsWithScope(c, ScopeAll)
}

func (h *Handler) listOwnerOutletsWithScope(c *gin.Context, scope string) {
	h.listOwnerOutletsWithScopeAndAll(c, scope, false)
}

func (h *Handler) listOwnerOutletsWithScopeAndAll(c *gin.Context, scope string, all bool) {
	ownerID, ok := parseParamID(c, "owner_id")
	if !ok {
		return
	}
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.Scope = scope
	params.All = all
	response, err := h.service.ListOutlets(c.Request.Context(), currentActor(c), ownerID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listGlobalOutlets(c *gin.Context) {
	h.listGlobalOutletsWithScope(c, ScopeActive)
}

func (h *Handler) listAllGlobalOutlets(c *gin.Context) {
	h.listGlobalOutletsWithScopeAndAll(c, ScopeActive, true)
}

func (h *Handler) listAllGlobalOutletsDeleted(c *gin.Context) {
	h.listGlobalOutletsWithScopeAndAll(c, ScopeAll, true)
}

func (h *Handler) listGlobalOutletsTrash(c *gin.Context) {
	h.listGlobalOutletsWithScope(c, ScopeDeleted)
}

func (h *Handler) listGlobalOutletsUnscoped(c *gin.Context) {
	h.listGlobalOutletsWithScope(c, ScopeAll)
}

func (h *Handler) listGlobalOutletsWithScope(c *gin.Context, scope string) {
	h.listGlobalOutletsWithScopeAndAll(c, scope, false)
}

func (h *Handler) listGlobalOutletsWithScopeAndAll(c *gin.Context, scope string, all bool) {
	params, err := listParams(c)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.Scope = scope
	params.All = all
	response, err := h.service.ListGlobalOutlets(c.Request.Context(), currentActor(c), params)
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

func (h *Handler) getGlobalOutlet(c *gin.Context) {
	outletID, ok := parseParamID(c, "outlet_id")
	if !ok {
		return
	}
	response, err := h.service.GetGlobalOutlet(c.Request.Context(), currentActor(c), outletID)
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
	meta := RequestMeta{IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), RequestID: httpx.RequestID(c)}
	response, err := h.service.UpdateOutletWithMeta(c.Request.Context(), currentActor(c), ownerID, outletID, req, meta)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listOutletHistories(c *gin.Context) {
	ownerID, outletID, ok := parseOwnerOutletID(c)
	if !ok {
		return
	}
	response, err := h.service.ListOutletHistories(c.Request.Context(), currentActor(c), ownerID, outletID)
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

func listParams(c *gin.Context) (ListParams, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := ListParams{
		Query:        c.Query("q"),
		OwnerKeyword: c.Query("owner_keyword"),
		Code:         c.Query("code"),
		Name:         c.Query("name"),
		Phone:        c.Query("phone"),
		BrandName:    c.Query("brand_name"),
		Province:     c.Query("province"),
		City:         c.Query("city"),
		SubscriptionStatus: func() string {
			if s := c.Query("subscription_status"); s != "" {
				return s
			}
			return c.Query("status")
		}(),
		SubscriptionMonth: c.Query("subscription_month"),
		All:               false,
		Page:              page,
		Limit:             limit,
		Sort:              c.Query("sort"),
	}
	if ownerID, err := strconv.ParseInt(c.Query("owner_id"), 10, 64); err == nil && ownerID > 0 {
		params.OwnerID = &ownerID
	}
	var err error
	createdFromVal := c.Query("created_from")
	if createdFromVal == "" {
		createdFromVal = c.Query("start_date")
	}
	createdToVal := c.Query("created_to")
	if createdToVal == "" {
		createdToVal = c.Query("end_date")
	}
	params.CreatedFrom, err = parseDateOnly(createdFromVal)
	if err != nil {
		return ListParams{}, err
	}
	params.CreatedTo, err = parseDateOnly(createdToVal)
	if err != nil {
		return ListParams{}, err
	}
	return params, nil
}

func parseDateOnly(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err == nil {
		return &parsed, nil
	}
	parsedRFC, errRFC := time.Parse(time.RFC3339, value)
	if errRFC == nil {
		return &parsedRFC, nil
	}
	return nil, errors.New("format tanggal harus YYYY-MM-DD")
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
		httpx.Error(c, http.StatusConflict, "CODE_ALREADY_USED", err.Error(), gin.H{
			"possible_cause": "kode yang dikirim sudah dipakai oleh data lain atau data yang dimasukkan kemungkinan adalah duplikat dari owner/outlet yang sudah ada",
			"hint":           "jika ini memang data baru yang mirip owner lama, frontend boleh meminta user mengganti kode menjadi kode baru yang unik sebelum mengirim ulang",
		})
	case errors.Is(err, ErrInvalidPhone):
		httpx.Error(c, http.StatusBadRequest, "INVALID_PHONE", err.Error(), nil)
	case errors.Is(err, ErrInvalidSort):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SORT", err.Error(), nil)
	case errors.Is(err, ErrEmptyBulk):
		httpx.Error(c, http.StatusBadRequest, "EMPTY_BULK", err.Error(), nil)
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	default:
		httpx.InternalServerError(c, "Terjadi kesalahan pada server", err)
	}
}
