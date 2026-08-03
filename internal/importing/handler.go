package importing

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		imports.GET("/all", h.ListAllBatches)
		imports.GET("/all-deleted", h.ListAllBatches)
		imports.GET("/:id", h.GetBatch)
		imports.GET("/:id/file", h.ViewOriginalFile)
		imports.GET("/:id/file/download", h.DownloadOriginalFile)
		imports.GET("/:id/rows", h.ListRows)
		imports.GET("/:id/rows/all", h.ListAllRows)
		imports.GET("/:id/rows/all-deleted", h.ListAllRows)
		imports.GET("/:id/rejected-rows/export", h.ExportRejectedRows)
		imports.POST("/:id/commit", h.Commit)
		imports.POST("/:id/rows/:row_id/relink", h.RelinkRow)
		imports.GET("/summary", h.GetSummary)
	}
}

// Upload godoc
// @Summary Upload an Excel file for import
// @Description Admin only. Deduplicated by file SHA-256 — re-uploading the same file returns the existing batch. Validation runs asynchronously via the background worker.
// @Tags imports
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel (.xlsx) file"
// @Param profile formData string false "OWNER_OUTLET, NON_REGISTER, NEW_SUBSCRIBE, MONTHLY_ACTIVE, BONUS_MITRA, SALES_CALL_CHAT, or SALES_TARGET; omit to auto-detect from headers"
// @Param sheet_name formData string false "Required for SALES_CALL_CHAT/SALES_TARGET (workbook has multiple similar sheets); optional/ignored otherwise"
// @Param target_sales_user_id formData int false "Required for SALES_CALL_CHAT/SALES_TARGET (sales rep is only encoded in the sheet name, not a data column)"
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
	sheetName := c.PostForm("sheet_name")
	var targetSalesUserID *int64
	if raw := c.PostForm("target_sales_user_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "target_sales_user_id tidak valid", nil)
			return
		}
		targetSalesUserID = &id
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.Upload(c.Request.Context(), actor, file, fileHeader, profile, sheetName, targetSalesUserID)
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
		All:     false,
	}
	var ok bool
	params.CreatedFrom, params.CreatedTo, ok = parseDateRangeQuery(c, "created_from", "created_to")
	if !ok {
		return
	}
	params.UploadedFrom, params.UploadedTo, ok = parseDateRangeQuery(c, "uploaded_from", "uploaded_to")
	if !ok {
		return
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

func (h *Handler) ListAllBatches(c *gin.Context) {
	params := ListBatchesParams{
		Status:  c.Query("status"),
		Profile: c.Query("profile"),
		All:     true,
	}
	var ok bool
	params.CreatedFrom, params.CreatedTo, ok = parseDateRangeQuery(c, "created_from", "created_to")
	if !ok {
		return
	}
	params.UploadedFrom, params.UploadedTo, ok = parseDateRangeQuery(c, "uploaded_from", "uploaded_to")
	if !ok {
		return
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

// ViewOriginalFile godoc
// @Summary View the originally uploaded Excel file
// @Description Admin only. Returns the same uploaded workbook so frontend can render its own Excel viewer page.
// @Tags imports
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path int64 true "Batch ID"
// @Success 200 {file} file
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /imports/{id}/file [get]
func (h *Handler) ViewOriginalFile(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	file, err := h.service.GetOriginalFile(c.Request.Context(), actor, id)
	if err != nil {
		writeImportError(c, err)
		return
	}
	c.Header("Content-Disposition", `inline; filename="`+file.OriginalFilename+`"`)
	if len(file.Content) > 0 {
		c.Data(http.StatusOK, file.ContentType, file.Content)
		return
	}
	c.File(file.Path)
}

// DownloadOriginalFile godoc
// @Summary Download the originally uploaded Excel file
// @Description Admin only. Returns the raw workbook as an attachment.
// @Tags imports
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path int64 true "Batch ID"
// @Success 200 {file} file
// @Failure 404 {object} httpx.ErrorEnvelope
// @Router /imports/{id}/file/download [get]
func (h *Handler) DownloadOriginalFile(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	actor, _ := identity.CurrentUser(c)
	file, err := h.service.GetOriginalFile(c.Request.Context(), actor, id)
	if err != nil {
		writeImportError(c, err)
		return
	}
	if len(file.Content) > 0 {
		c.Header("Content-Disposition", `attachment; filename="`+file.OriginalFilename+`"`)
		c.Data(http.StatusOK, file.ContentType, file.Content)
		return
	}
	c.FileAttachment(file.Path, file.OriginalFilename)
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
	params := ListRowsParams{Status: c.Query("status"), All: false}
	var rangeOK bool
	params.CreatedFrom, params.CreatedTo, rangeOK = parseDateRangeQuery(c, "created_from", "created_to")
	if !rangeOK {
		return
	}
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

func (h *Handler) ListAllRows(c *gin.Context) {
	id, ok := parseBatchID(c)
	if !ok {
		return
	}
	params := ListRowsParams{Status: c.Query("status"), All: true}
	var rangeOK bool
	params.CreatedFrom, params.CreatedTo, rangeOK = parseDateRangeQuery(c, "created_from", "created_to")
	if !rangeOK {
		return
	}
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
	statusCode := http.StatusAccepted
	if resp.Status == BatchStatusCommitted {
		statusCode = http.StatusOK
	}
	httpx.Success(c, statusCode, resp)
}

// RelinkRow godoc
// @Summary Manually resolve an UNMATCHED reconciliation-candidate row
// @Description Admin only. Supplies the owner/outlet/lead ID the row's own data couldn't resolve at commit time, moving it back to VALID for the next batch commit to pick up.
// @Tags imports
// @Accept json
// @Produce json
// @Param id path int64 true "Batch ID"
// @Param row_id path int64 true "Row ID"
// @Param request body RelinkRowRequest true "Entity IDs to link"
// @Success 200 {object} ImportRowResponse
// @Failure 400 {object} httpx.ErrorEnvelope
// @Router /imports/{id}/rows/{row_id}/relink [post]
func (h *Handler) RelinkRow(c *gin.Context) {
	batchID, ok := parseBatchID(c)
	if !ok {
		return
	}
	rowID, err := strconv.ParseInt(c.Param("row_id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid row ID", nil)
		return
	}
	var req RelinkRowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "payload tidak valid", gin.H{"error": err.Error()})
		return
	}
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.RelinkRow(c.Request.Context(), actor, batchID, rowID, req)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// GetSummary godoc
// @Summary Aggregate batch counts per status
// @Description Admin only. Reports how many batches need attention (VALIDATION_FAILED/COMMIT_FAILED) without paging through GET /imports?status=... one status at a time.
// @Tags imports
// @Produce json
// @Success 200 {object} BatchSummaryResponse
// @Router /imports/summary [get]
func (h *Handler) GetSummary(c *gin.Context) {
	actor, _ := identity.CurrentUser(c)
	resp, err := h.service.GetSummary(c.Request.Context(), actor)
	if err != nil {
		writeImportError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
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
	var batchStatusErr *BatchStatusActionError
	requestID := httpx.RequestID(c)
	logImportError(c, err)

	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error(), importErrorDetails("NOT_FOUND", err, requestID))
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), importErrorDetails("FORBIDDEN", err, requestID))
	case errors.Is(err, ErrInvalidFileType):
		httpx.Error(c, http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error(), importErrorDetails("INVALID_FILE_TYPE", err, requestID))
	case errors.Is(err, ErrFileTooLarge):
		httpx.Error(c, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), importErrorDetails("FILE_TOO_LARGE", err, requestID))
	case errors.Is(err, ErrFileUnavailable):
		httpx.Error(c, http.StatusNotFound, "FILE_UNAVAILABLE", err.Error(), importErrorDetails("FILE_UNAVAILABLE", err, requestID))
	case errors.Is(err, ErrEmptyFile):
		httpx.Error(c, http.StatusBadRequest, "EMPTY_FILE", err.Error(), importErrorDetails("EMPTY_FILE", err, requestID))
	case errors.Is(err, ErrProfileRequired):
		httpx.Error(c, http.StatusBadRequest, "PROFILE_REQUIRED", err.Error(), importErrorDetails("PROFILE_REQUIRED", err, requestID))
	case errors.Is(err, ErrProfileHeaderMismatch):
		httpx.Error(c, http.StatusBadRequest, "PROFILE_HEADER_MISMATCH", err.Error(), importErrorDetails("PROFILE_HEADER_MISMATCH", err, requestID))
	case errors.Is(err, ErrUnknownProfile):
		httpx.Error(c, http.StatusBadRequest, "UNKNOWN_PROFILE", err.Error(), importErrorDetails("UNKNOWN_PROFILE", err, requestID))
	case errors.Is(err, ErrSheetNameRequired):
		httpx.Error(c, http.StatusBadRequest, "SHEET_NAME_REQUIRED", err.Error(), importErrorDetails("SHEET_NAME_REQUIRED", err, requestID))
	case errors.Is(err, ErrSheetNameNeedsProfile):
		httpx.Error(c, http.StatusBadRequest, "SHEET_NAME_NEEDS_PROFILE", err.Error(), importErrorDetails("SHEET_NAME_NEEDS_PROFILE", err, requestID))
	case errors.Is(err, ErrSheetNotFound):
		httpx.Error(c, http.StatusBadRequest, "SHEET_NOT_FOUND", err.Error(), importErrorDetails("SHEET_NOT_FOUND", err, requestID))
	case errors.Is(err, ErrTargetSalesUserRequired):
		httpx.Error(c, http.StatusBadRequest, "TARGET_SALES_USER_REQUIRED", err.Error(), importErrorDetails("TARGET_SALES_USER_REQUIRED", err, requestID))
	case errors.As(err, &batchStatusErr):
		httpx.Error(c, http.StatusBadRequest, "INVALID_BATCH_STATUS", err.Error(), gin.H{
			"root_cause":       "Frontend memanggil aksi import pada status batch yang belum sesuai alur backend.",
			"solution":         "Poll GET /imports/{id} dan hanya aktifkan aksi yang sesuai dengan status batch saat ini.",
			"frontend_prevent": "Gunakan status batch sebagai source of truth utama. Tombol commit hanya aktif saat status VALIDATED.",
			"action":           batchStatusErr.Action,
			"current_status":   batchStatusErr.CurrentStatus,
			"allowed_statuses": batchStatusErr.AllowedStatuses,
			"hint":             "poll GET /imports/{id} until status VALIDATED before first commit; retry commit safely if status COMMITTING or COMMITTED",
			"technical_error":  err.Error(),
			"request_id":       requestID,
		})
	case errors.Is(err, ErrInvalidBatchStatus):
		httpx.Error(c, http.StatusBadRequest, "INVALID_BATCH_STATUS", err.Error(), importErrorDetails("INVALID_BATCH_STATUS", err, requestID))
	case errors.Is(err, ErrRowNotUnmatched):
		httpx.Error(c, http.StatusConflict, "ROW_NOT_UNMATCHED", err.Error(), importErrorDetails("ROW_NOT_UNMATCHED", err, requestID))
	case errors.Is(err, ErrRelinkEntityRequired):
		httpx.Error(c, http.StatusBadRequest, "RELINK_ENTITY_REQUIRED", err.Error(), importErrorDetails("RELINK_ENTITY_REQUIRED", err, requestID))
	default:
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", importErrorDetails("INTERNAL_ERROR", err, requestID))
	}
}

func parseDateRangeQuery(c *gin.Context, fromKey, toKey string) (*time.Time, *time.Time, bool) {
	from, err := parseOptionalDate(c.Query(fromKey))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", fromKey+" harus format YYYY-MM-DD", nil)
		return nil, nil, false
	}
	to, err := parseOptionalDate(c.Query(toKey))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", toKey+" harus format YYYY-MM-DD", nil)
		return nil, nil, false
	}
	return from, to, true
}

func parseOptionalDate(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func logImportError(c *gin.Context, err error) {
	slog.Error("import request failed",
		slog.String("request_id", httpx.RequestID(c)),
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("client_ip", c.ClientIP()),
		slog.String("error", err.Error()),
	)
}

func importErrorDetails(code string, err error, requestID string) gin.H {
	details := gin.H{
		"request_id":      requestID,
		"technical_error": err.Error(),
	}

	switch code {
	case "NOT_FOUND":
		details["root_cause"] = "Batch import atau file yang diminta tidak ditemukan."
		details["solution"] = "Gunakan ID batch yang valid dan pastikan batch masih tersedia."
		details["frontend_prevent"] = "Simpan ID batch dari response upload/list, lalu refresh data bila batch sudah lama."
	case "FORBIDDEN":
		details["root_cause"] = "User yang sedang login tidak memiliki hak akses modul import."
		details["solution"] = "Lakukan login sebagai Admin."
		details["frontend_prevent"] = "Sembunyikan modul import untuk role non-Admin dan redirect bila token tidak punya akses."
	case "INVALID_FILE_TYPE":
		details["root_cause"] = "File yang diunggah bukan format Excel .xlsx."
		details["solution"] = "Unggah file .xlsx yang valid."
		details["frontend_prevent"] = "Validasi ekstensi file di frontend sebelum upload."
	case "FILE_TOO_LARGE":
		details["root_cause"] = "Ukuran file melebihi batas maksimum upload backend."
		details["solution"] = "Perkecil file atau pecah menjadi beberapa file import."
		details["frontend_prevent"] = "Cek ukuran file sebelum upload dan tampilkan batas maksimum ke user."
	case "FILE_UNAVAILABLE":
		details["root_cause"] = "File asli batch tidak lagi tersedia di storage server."
		details["solution"] = "Unggah ulang file import baru."
		details["frontend_prevent"] = "Tampilkan fallback ketika file viewer gagal, lalu arahkan user untuk re-upload."
	case "EMPTY_FILE":
		details["root_cause"] = "File upload kosong atau gagal terbaca."
		details["solution"] = "Pastikan file berisi workbook Excel yang valid."
		details["frontend_prevent"] = "Blok upload file berukuran 0 byte."
	case "PROFILE_REQUIRED":
		details["root_cause"] = "Backend tidak bisa menebak profil import dari header file."
		details["solution"] = "Kirim parameter profile secara eksplisit, mis. OWNER_OUTLET atau NON_REGISTER."
		details["frontend_prevent"] = "Sediakan selector profile ketika auto-detect tidak yakin atau file berasal dari template manual."
	case "PROFILE_HEADER_MISMATCH":
		details["root_cause"] = "Header kolom file tidak cocok dengan profile yang dipilih."
		details["solution"] = "Pilih profile yang benar atau gunakan template Excel yang sesuai."
		details["frontend_prevent"] = "Pasangkan pilihan profile dengan template file yang tepat dan tampilkan contoh header yang diharapkan."
	case "UNKNOWN_PROFILE":
		details["root_cause"] = "Nilai profile yang dikirim frontend tidak dikenal backend."
		details["solution"] = "Gunakan hanya profile yang didukung backend."
		details["frontend_prevent"] = "Gunakan enum profile yang dibekukan dari API/OpenAPI, jangan hardcode bebas."
	case "SHEET_NAME_REQUIRED":
		details["root_cause"] = "Profile import ini memakai workbook multi-sheet sehingga backend butuh nama sheet yang eksplisit."
		details["solution"] = "Kirim parameter sheet_name yang sama dengan nama sheet pada file Excel."
		details["frontend_prevent"] = "Wajibkan user memilih sheet ketika profile adalah SALES_CALL_CHAT atau SALES_TARGET."
	case "SHEET_NAME_NEEDS_PROFILE":
		details["root_cause"] = "Frontend mengirim sheet_name tanpa profile eksplisit."
		details["solution"] = "Kirim profile yang sesuai bersama sheet_name."
		details["frontend_prevent"] = "Jangan tampilkan input sheet_name bila profile belum dipilih."
	case "SHEET_NOT_FOUND":
		details["root_cause"] = "Nama sheet yang dikirim tidak ditemukan di workbook yang diunggah."
		details["solution"] = "Periksa ejaan, spasi, dan kapitalisasi nama sheet lalu unggah ulang request."
		details["frontend_prevent"] = "Baca daftar sheet dari file terlebih dahulu atau tampilkan helper nama sheet yang valid ke user."
	case "TARGET_SALES_USER_REQUIRED":
		details["root_cause"] = "Profile import ini membutuhkan target_sales_user_id karena sales hanya diketahui dari konteks sheet."
		details["solution"] = "Kirim target_sales_user_id dari akun Sales yang sesuai."
		details["frontend_prevent"] = "Wajibkan pemilihan Sales saat profile adalah SALES_CALL_CHAT atau SALES_TARGET."
	case "INVALID_BATCH_STATUS":
		details["root_cause"] = "Aksi import dipanggil saat status batch belum sesuai."
		details["solution"] = "Ikuti state machine batch: upload -> validating -> validated -> committing -> committed."
		details["frontend_prevent"] = "Enable/disable tombol berdasarkan status batch dari GET /imports/{id}."
	default:
		details["root_cause"] = inferImportRootCause(err)
		details["solution"] = inferImportSolution(err)
		details["frontend_prevent"] = inferImportFrontendPrevent(err)
	}

	return details
}

func inferImportRootCause(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unknown column") && strings.Contains(message, "progress_percentage"):
		return "Skema database production belum sesuai dengan kode backend terbaru; kolom progress_percentage belum ada."
	case strings.Contains(message, "unknown column"):
		return "Skema database tidak sinkron dengan versi backend yang sedang berjalan."
	case strings.Contains(message, "create upload directory"):
		return "Backend gagal membuat folder penyimpanan file upload."
	case strings.Contains(message, "store upload"):
		return "Backend gagal menyimpan file upload ke storage server."
	case strings.Contains(message, "find batch by sha256"):
		return "Backend gagal membaca data batch import dari database saat deduplikasi file."
	case strings.Contains(message, "create batch"):
		return "Backend gagal membuat row import_batches di database."
	case strings.Contains(message, "set validate job id"):
		return "Backend gagal menyimpan relasi job validasi ke batch import."
	case strings.Contains(message, "jobqueue") || strings.Contains(message, "enqueue"):
		return "Backend gagal memasukkan job import ke antrean worker."
	default:
		return "Terjadi error teknis internal pada proses upload/orkestrasi import."
	}
}

func inferImportSolution(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unknown column") && strings.Contains(message, "progress_percentage"):
		return "Jalankan migration terbaru di environment production sampai versi 20260728000100 atau lebih baru."
	case strings.Contains(message, "unknown column"):
		return "Pastikan migration production sudah sejajar dengan commit backend yang sedang dideploy."
	case strings.Contains(message, "create upload directory"), strings.Contains(message, "store upload"):
		return "Periksa konfigurasi UPLOAD_DIR, permission filesystem, dan persistent volume pada environment server."
	case strings.Contains(message, "find batch by sha256"), strings.Contains(message, "create batch"), strings.Contains(message, "set validate job id"):
		return "Periksa koneksi database, struktur tabel import_batches, dan foreign key/job_queue yang dibutuhkan."
	case strings.Contains(message, "jobqueue"), strings.Contains(message, "enqueue"):
		return "Pastikan tabel job_queue tersedia dan worker berjalan normal di environment tersebut."
	default:
		return "Periksa app logs menggunakan request_id ini untuk melihat technical_error dan perbaiki resource server/database yang disebutkan."
	}
}

func inferImportFrontendPrevent(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unknown column"):
		return "Frontend tidak perlu retry buta; tampilkan pesan maintenance/server mismatch dan arahkan user menghubungi admin sistem."
	case strings.Contains(message, "create upload directory"), strings.Contains(message, "store upload"):
		return "Jika server gagal menyimpan file, frontend cukup tampilkan error final; retry otomatis tidak akan banyak membantu sampai storage server diperbaiki."
	case strings.Contains(message, "jobqueue"), strings.Contains(message, "enqueue"):
		return "Frontend dapat menyarankan retry manual beberapa saat lagi, tetapi tetap tampilkan bahwa worker/import queue backend sedang bermasalah."
	default:
		return "Tampilkan technical_error, root_cause, dan solution dari response agar user office bisa melapor dengan konteks yang lengkap."
	}
}
