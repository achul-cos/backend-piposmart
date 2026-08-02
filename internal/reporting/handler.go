package reporting

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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	reports := rg.Group("/reports")
	{
		reports.GET("/dashboard", h.dashboard)
		reports.POST("/exports", h.createExport)
		reports.GET("/exports", h.listExports)
		reports.GET("/exports/:id", h.getExport)
		reports.GET("/exports/:id/download", h.downloadExport)
		reports.GET("/:report_key", h.listReport)
	}
}

func (h *Handler) dashboard(c *gin.Context) {
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.Dashboard(c.Request.Context(), actor, DashboardParams{
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
	})
	if err != nil {
		writeReportingError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) listReport(c *gin.Context) {
	actor, _ := identity.CurrentUser(c)
	params := ListReportsParams{
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
		Status:   c.Query("status"),
		Query:    c.Query("q"),
		Province: c.Query("province"),
		City:     c.Query("city"),
		All:      c.Query("all") == "true",
	}
	params.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	params.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if v := c.Query("sales_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil && id > 0 {
			params.SalesID = &id
		}
	}
	if v := c.Query("supervisor_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil && id > 0 {
			params.SupervisorID = &id
		}
	}
	resp, err := h.service.ListReport(c.Request.Context(), actor, c.Param("report_key"), params)
	if err != nil {
		writeReportingError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) createExport(c *gin.Context) {
	var req CreateExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.CreateExport(c.Request.Context(), actor, req)
	if err != nil {
		writeReportingError(c, err)
		return
	}
	httpx.Success(c, http.StatusAccepted, resp)
}

func (h *Handler) listExports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListExports(c.Request.Context(), actor, page, limit)
	if err != nil {
		writeReportingError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) getExport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "id export tidak valid", nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.GetExport(c.Request.Context(), actor, id)
	if err != nil {
		writeReportingError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) downloadExport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "id export tidak valid", nil)
		return
	}
	actor, _ := identity.CurrentUser(c)
	file, err := h.service.DownloadExport(c.Request.Context(), actor, id)
	if err != nil {
		writeReportingError(c, err)
		return
	}
	if len(file.Content) > 0 {
		c.Header("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
		c.Data(http.StatusOK, file.ContentType, file.Content)
		return
	}
	c.Header("Content-Type", file.ContentType)
	c.FileAttachment(file.Path, file.FileName)
}

func writeReportingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", "akses ditolak", nil)
	case errors.Is(err, ErrInvalidReportKey), errors.Is(err, ErrInvalidFormat):
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, ErrExportNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrExportNotReady):
		httpx.Error(c, http.StatusConflict, "EXPORT_NOT_READY", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
