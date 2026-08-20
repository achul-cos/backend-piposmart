package activity

import (
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
	api.GET("/customer-interactions", h.listInteractions)
	api.GET("/customer-interactions/all", h.listAllInteractions)
	api.GET("/customer-interactions/all-deleted", h.listAllInteractions)
	api.GET("/customer-interactions/:interaction_id", h.getInteraction)
	api.GET("/follow-ups", h.listFollowUps)
	api.GET("/follow-ups/all", h.listAllFollowUps)
	api.GET("/follow-ups/all-deleted", h.listAllFollowUps)
	api.GET("/leads/:lead_id/interactions", h.listLeadInteractions)
	api.GET("/leads/:lead_id/interactions/all", h.listAllLeadInteractions)
	api.GET("/leads/:lead_id/interactions/all-deleted", h.listAllLeadInteractions)
	api.POST("/leads/:lead_id/interactions", h.createInteraction)
	api.GET("/leads/:lead_id/stage-history", h.stageHistory)

	api.GET("/trainings", h.listTrainings)
	api.GET("/trainings/all", h.listAllTrainings)
	api.GET("/trainings/all-deleted", h.listAllTrainings)
	api.GET("/trainings/:training_id", h.getTraining)
	api.GET("/leads/:lead_id/trainings", h.listLeadTrainings)
	api.GET("/leads/:lead_id/trainings/all", h.listAllLeadTrainings)
	api.GET("/leads/:lead_id/trainings/all-deleted", h.listAllLeadTrainings)
	api.POST("/leads/:lead_id/trainings", h.scheduleTraining)
	api.POST("/trainings/:training_id/reschedule", h.rescheduleTraining)
	api.POST("/trainings/:training_id/complete", h.completeTraining)
	api.POST("/trainings/:training_id/cancel", h.cancelTraining)
}

func (h *Handler) listInteractions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := interactionListParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListInteractions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllInteractions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := interactionListParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListInteractions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listFollowUps(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := interactionListParams(c)
	if !ok {
		return
	}
	params.OnlyFollowUps = true
	response, err := h.service.ListInteractions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllFollowUps(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := interactionListParams(c)
	if !ok {
		return
	}
	params.OnlyFollowUps = true
	params.All = true
	response, err := h.service.ListInteractions(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listLeadInteractions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	params, ok := interactionListParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListLeadInteractions(c.Request.Context(), user, leadID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllLeadInteractions(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	params, ok := interactionListParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListLeadInteractions(c.Request.Context(), user, leadID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) createInteraction(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	var req CreateInteractionRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.CreateInteraction(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) stageHistory(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	response, err := h.service.StageHistory(c.Request.Context(), user, leadID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listTrainings(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := trainingListParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListTrainings(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getInteraction(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	interactionID, ok := parsePathID(c, "interaction_id", "ID interaksi tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetInteraction(c.Request.Context(), user, interactionID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) getTraining(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	trainingID, ok := parsePathID(c, "training_id", "ID training tidak valid")
	if !ok {
		return
	}
	response, err := h.service.GetTraining(c.Request.Context(), user, trainingID)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllTrainings(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	params, ok := trainingListParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListTrainings(c.Request.Context(), user, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listLeadTrainings(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	params, ok := trainingListParams(c)
	if !ok {
		return
	}
	response, err := h.service.ListLeadTrainings(c.Request.Context(), user, leadID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) listAllLeadTrainings(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	params, ok := trainingListParams(c)
	if !ok {
		return
	}
	params.All = true
	response, err := h.service.ListLeadTrainings(c.Request.Context(), user, leadID, params)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) scheduleTraining(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	leadID, ok := parsePathID(c, "lead_id", "ID lead tidak valid")
	if !ok {
		return
	}
	var req ScheduleTrainingRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.ScheduleTraining(c.Request.Context(), user, leadID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

func (h *Handler) rescheduleTraining(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	trainingID, ok := parsePathID(c, "training_id", "ID training tidak valid")
	if !ok {
		return
	}
	var req RescheduleTrainingRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.service.RescheduleTraining(c.Request.Context(), user, trainingID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) completeTraining(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	trainingID, ok := parsePathID(c, "training_id", "ID training tidak valid")
	if !ok {
		return
	}
	var req CompleteTrainingRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	response, err := h.service.CompleteTraining(c.Request.Context(), user, trainingID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func (h *Handler) cancelTraining(c *gin.Context) {
	user, _ := identity.CurrentUser(c)
	trainingID, ok := parsePathID(c, "training_id", "ID training tidak valid")
	if !ok {
		return
	}
	var req CancelTrainingRequest
	if !bindOptionalJSON(c, &req) {
		return
	}
	response, err := h.service.CancelTraining(c.Request.Context(), user, trainingID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func interactionListParams(c *gin.Context) (InteractionListParams, bool) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := InteractionListParams{
		Type:  c.Query("type"),
		All:   c.Query("all") == "true",
		Page:  page,
		Limit: limit,
		Sort:  c.Query("sort"),
	}
	if !parseOptionalInt64Query(c, "lead_id", &params.LeadID) ||
		!parseOptionalInt64Query(c, "score", &params.Score) ||
		!parseOptionalInt64Query(c, "sales_id", &params.SalesID) {
		return InteractionListParams{}, false
	}
	if params.Score != nil && (*params.Score < 0 || *params.Score > 3) {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "score harus angka 0 sampai 3", nil)
		return InteractionListParams{}, false
	}
	var err error
	params.InteractionFrom, err = parseDate(c.Query("interaction_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "interaction_from harus format YYYY-MM-DD", nil)
		return InteractionListParams{}, false
	}
	params.InteractionTo, err = parseDate(c.Query("interaction_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "interaction_to harus format YYYY-MM-DD", nil)
		return InteractionListParams{}, false
	}
	params.FollowUpFrom, err = parseDate(c.Query("follow_up_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "follow_up_from harus format YYYY-MM-DD", nil)
		return InteractionListParams{}, false
	}
	params.FollowUpTo, err = parseDate(c.Query("follow_up_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "follow_up_to harus format YYYY-MM-DD", nil)
		return InteractionListParams{}, false
	}
	params.CreatedFrom, err = parseDate(c.Query("created_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "created_from harus format YYYY-MM-DD", nil)
		return InteractionListParams{}, false
	}
	params.CreatedTo, err = parseDate(c.Query("created_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "created_to harus format YYYY-MM-DD", nil)
		return InteractionListParams{}, false
	}
	return params, true
}

func trainingListParams(c *gin.Context) (TrainingListParams, bool) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	params := TrainingListParams{
		Status:       c.Query("status"),
		TrainingType: c.Query("training_type"),
		All:          c.Query("all") == "true",
		Page:         page,
		Limit:        limit,
		Sort:         c.Query("sort"),
	}
	if !parseOptionalInt64Query(c, "lead_id", &params.LeadID) ||
		!parseOptionalInt64Query(c, "sales_id", &params.SalesID) {
		return TrainingListParams{}, false
	}
	var err error
	params.ScheduledFrom, err = parseDate(c.Query("scheduled_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "scheduled_from harus format YYYY-MM-DD", nil)
		return TrainingListParams{}, false
	}
	params.ScheduledTo, err = parseDate(c.Query("scheduled_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "scheduled_to harus format YYYY-MM-DD", nil)
		return TrainingListParams{}, false
	}
	params.CreatedFrom, err = parseDate(c.Query("created_from"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "created_from harus format YYYY-MM-DD", nil)
		return TrainingListParams{}, false
	}
	params.CreatedTo, err = parseDate(c.Query("created_to"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "created_to harus format YYYY-MM-DD", nil)
		return TrainingListParams{}, false
	}
	return params, true
}

func parseOptionalInt64Query(c *gin.Context, name string, target **int64) bool {
	raw := c.Query(name)
	if raw == "" {
		return true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
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
	case errors.Is(err, ErrInvalidScore):
		httpx.Error(c, http.StatusBadRequest, "INVALID_SCORE", err.Error(), nil)
	case errors.Is(err, ErrInteractionEmpty):
		httpx.Error(c, http.StatusBadRequest, "INTERACTION_CHANNEL_REQUIRED", err.Error(), gin.H{
			"root_cause":       "request interaction tidak mengirim status call/chat dan tidak mengirim fallback type lama",
			"solution":         "kirim minimal salah satu field call_status atau chat_status. Field type lama masih didukung untuk kompatibilitas, tetapi bukan format utama.",
			"frontend_prevent": "pastikan form mengharuskan minimal satu status channel sebelum submit",
		})
	case errors.Is(err, ErrInvalidType):
		httpx.Error(c, http.StatusBadRequest, "INVALID_TYPE", err.Error(), nil)
	case errors.Is(err, ErrRemarkNotFound):
		httpx.Error(c, http.StatusBadRequest, "REMARK_REASON_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(c, http.StatusBadRequest, "INVALID_TRANSITION", err.Error(), nil)
	case errors.Is(err, ErrLeadHasNoPIC):
		httpx.Error(c, http.StatusBadRequest, "LEAD_HAS_NO_PIC", err.Error(), nil)
	default:
		httpx.InternalServerError(c, "Terjadi kesalahan pada server", err)
	}
}
