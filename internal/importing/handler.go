package importing

import (
	"net/http"
	"strconv"

	"backend_crm_piposmart/internal/identity"
	"backend_crm_piposmart/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// Handler handles import-framework HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new importing Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers import routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	imports := rg.Group("/imports")
	{
		imports.POST("", h.Upload)
		imports.GET("", h.ListBatches)
		imports.GET("/:id", h.GetBatch)
		imports.GET("/:id/rows", h.ListRows)
		imports.GET("/:id/rejected-rows/export", h.ExportRejectedRows)
		imports.POST("/:id/commit", h.Commit)
	}
}

// Upload godoc
// @Summary Upload an Excel file for import
// @Description Admin only. Deduplicated by file SHA-256 — re-uploading the same file returns the existing batch. Validation runs asynchronously via the background worker.
// @Tags imports
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel (.xlsx) file"
// @Param profile formData string false "OWNER_OUTLET or NON_REGISTER; omit to auto-detect from headers"
// @Success 201 {object} ImportBatchResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /imports [post]
func (h *Handler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "file wajib diunggah", nil)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "gagal membuka file", nil)
		return
	}
	defer file.Close()

	profile := c.PostForm("profile")
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.Upload(c.Request.Context(), actor, file, fileHeader, profile)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// ListBatches godoc
// @Summary List import batches
// @Tags imports
// @Produce json
// @Param status query string false "Filter by status"
// @Param profile query string false "Filter by profile"
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Success 200 {object} ImportBatchListResponse
// @Router /imports [get]
func (h *Handler) ListBatches(c *gin.Context) {
	params := ListBatchesParams{
		Status:  c.Query("status"),
		Profile: c.Query("profile"),
	}
	params.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	params.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))

	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListBatches(c.Request.Context(), actor, params)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// GetBatch godoc
// @Summary Get an import batch
// @Description Batch status/counts serve as the preview before deciding whether to commit.
// @Tags imports
// @Produce json
// @Param id path int64 true "Batch ID"
// @Success 200 {object} ImportBatchResponse
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /imports/{id} [get]
func (h *Handler) GetBatch(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.GetBatch(c.Request.Context(), actor, id)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ListRows godoc
// @Summary List rows within an import batch
// @Tags imports
// @Produce json
// @Param id path int64 true "Batch ID"
// @Param status query string false "Filter by row status (PENDING/VALID/INVALID/COMMITTED/COMMIT_FAILED)"
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Success 200 {object} ImportRowListResponse
// @Router /imports/{id}/rows [get]
func (h *Handler) ListRows(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	params := ListRowsParams{Status: c.Query("status")}
	params.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	params.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))

	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.ListRows(c.Request.Context(), actor, id, params)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// ExportRejectedRows godoc
// @Summary Download rejected (INVALID) rows as CSV
// @Tags imports
// @Produce text/csv
// @Param id path int64 true "Batch ID"
// @Success 200 {file} file
// @Router /imports/{id}/rejected-rows/export [get]
func (h *Handler) ExportRejectedRows(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	content, filename, err := h.service.ExportRejectedRows(c.Request.Context(), actor, id)
	if err != nil {
		writeImportError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "text/csv", content)
}

// Commit godoc
// @Summary Commit a validated import batch
// @Description Admin only. Guarded: only a VALIDATED batch can be committed — this is the "preview approved" gate. Runs asynchronously via the background worker; poll GET /imports/{id} for status.
// @Tags imports
// @Produce json
// @Param id path int64 true "Batch ID"
// @Success 202 {object} ImportBatchResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /imports/{id}/commit [post]
func (h *Handler) Commit(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.TriggerCommit(c.Request.Context(), actor, id)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusAccepted, resp)
}

func parseBatchID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid batch ID", nil)
		return 0, false
	}
	return id, true
}

func writeImportError(c *gin.Context, err error) {
	switch err {
	case ErrNotFound:
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case ErrForbidden:
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case ErrInvalidFileType:
		httpx.Error(c, http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error(), nil)
	case ErrFileTooLarge:
		httpx.Error(c, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), nil)
	case ErrEmptyFile:
		httpx.Error(c, http.StatusBadRequest, "EMPTY_FILE", err.Error(), nil)
	case ErrProfileRequired:
		httpx.Error(c, http.StatusBadRequest, "PROFILE_REQUIRED", err.Error(), nil)
	case ErrProfileHeaderMismatch:
		httpx.Error(c, http.StatusBadRequest, "PROFILE_HEADER_MISMATCH", err.Error(), nil)
	case ErrUnknownProfile:
		httpx.Error(c, http.StatusBadRequest, "UNKNOWN_PROFILE", err.Error(), nil)
	case ErrInvalidBatchStatus:
		httpx.Error(c, http.StatusBadRequest, "INVALID_BATCH_STATUS", err.Error(), nil)
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
	}
}
